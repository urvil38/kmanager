package manager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/urvil38/kmanager/questions"
)

type GcloudAccount struct {
	Compute struct {
		Region string `json:"region"`
		Zone   string `json:"zone"`
	} `json:"compute"`
	Core struct {
		Account               string `json:"account"`
		DisableUsageReporting string `json:"disable_usage_reporting"`
		Project               string `json:"project"`
	} `json:"core"`
}

type ProjectList []struct {
	CreateTime     time.Time `json:"createTime"`
	LifecycleState string    `json:"lifecycleState"`
	Name           string    `json:"name"`
	ProjectID      string    `json:"projectId"`
	ProjectNumber  string    `json:"projectNumber"`
}

type DNSRecords []struct {
	Kind    string   `json:"kind"`
	Name    string   `json:"name"`
	Rrdatas []string `json:"rrdatas"`
	TTL     int      `json:"ttl"`
	Type    string   `json:"type"`
}

func (cc *ClusterConfig) initGCloudCmdSet() (*cmdSet, error) {
	gcloudCmds := newCmdSet(cc, "gcloud")

	cmds := []Command{
		Command{
			name:    "check-gcloud-login",
			rootCmd: "gcloud",
			args:    []string{"config", "list", "--format", "json"},
			runFn: func(cmd *Command) error {
				if cmd.succeed {
					var ga GcloudAccount
					err := json.NewDecoder(strings.NewReader(cmd.stdout)).Decode(&ga)
					if err != nil {
						return err
					}
					if ga.Core.Account == "" {
						loginCmd, err := gcloudCmds.getCommand("gcloud-login")
						if err != nil {
							return err
						}
						loginCmd.execute(context.Background(), cc)
						if !loginCmd.succeed {
							return loginCmd.stderr
						}
					}
					if ga.Compute.Region != "" {
						cc.Region = ga.Compute.Region
					}
					if ga.Compute.Zone != "" {
						cc.Zone = ga.Compute.Zone
					}
					if ga.Core.Account != "" {
						cc.Account = ga.Core.Account
					}
				} else {
					return cmd.stderr
				}
				return nil
			},
		},
		Command{
			name:     "gcloud-login",
			rootCmd:  "gcloud",
			args:     []string{"auth", "login"},
			internal: true,
		},
		Command{
			name:    "list-gcloud-accounts",
			rootCmd: "gcloud",
			args:    []string{"projects", "list", "--filter", "lifecycleState:ACTIVE", "--format", "json"},
			runFn: func(cmd *Command) error {
				if cmd.succeed {
					var pl ProjectList
					err := json.NewDecoder(strings.NewReader(cmd.stdout)).Decode(&pl)
					if err != nil {
						return err
					}

					var projectsOpts []string

					for _, p := range pl {
						projectsOpts = append(projectsOpts, fmt.Sprintf("%s (%s)", p.Name, p.ProjectID))
					}

					var selectedProj string
					survey.Ask(questions.ProjectPrompt(projectsOpts), &selectedProj, survey.WithValidator(survey.Required))
					if selectedProj != "" {
						openIndex := strings.Index(selectedProj, `(`)
						closeIndex := strings.Index(selectedProj, `)`)
						if openIndex != -1 && closeIndex != -1 && openIndex < closeIndex {
							cc.GcloudProjectName = selectedProj[openIndex+1 : closeIndex]
						} else {
							return errors.New("Invalid project name")
						}
					} else {
						return errors.New("Invalid project name")
					}
				} else {
					return cmd.stderr
				}
				return nil
			},
		},
		Command{
			name:    "create-storage-bucket-soucecode",
			rootCmd: "gsutil",
			args:    []string{"mb", "-c", "COLDLINE", "gs://" + cc.getStorageOpts().SourceCodeBucket},
			runFn: func(cmd *Command) error {
				return nil
			},
		},
		Command{
			name:    "create-storage-bucket-cloudbuild-logs",
			rootCmd: "gsutil",
			args:    []string{"mb", "-c", "COLDLINE", "gs://" + cc.getStorageOpts().CloudBuildBucket},
		},
		Command{
			name:    "create-kubernetes-cluster",
			rootCmd: "gcloud",
			generateArgs: func(cc *ClusterConfig) []string {
				return []string{
					"container", "clusters", "create", cc.Name,
					"--zone", cc.Zone,
					"--no-enable-basic-auth",
					"--cluster-version", "1.15.12-gke.2",
					"--machine-type", "n1-standard-1",
					"--image-type", "COS",
					"--disk-type", "pd-standard",
					"--disk-size=10",
					"--scopes",
					"https://www.googleapis.com/auth/devstorage.read_only,https://www.googleapis.com/auth/logging.write,https://www.googleapis.com/auth/monitoring,https://www.googleapis.com/auth/servicecontrol,https://www.googleapis.com/auth/service.management.readonly,https://www.googleapis.com/auth/trace.append,https://www.googleapis.com/auth/ndev.clouddns.readwrite",
					"--preemptible",
					"--num-nodes=2",
					"--network", fmt.Sprintf("projects/%s/global/networks/default", cc.GcloudProjectName),
					"--subnetwork", fmt.Sprintf("projects/%s/regions/%s/subnetworks/default", cc.GcloudProjectName, cc.Region),
					"--addons", "HorizontalPodAutoscaling,HttpLoadBalancing",
					"--enable-autoupgrade",
					"--enable-autorepair",
				}
			},
			internal: true,
		},
		Command{
			name:    "get-kubernetes-credentials",
			rootCmd: "gcloud",
			generateArgs: func(cc *ClusterConfig) []string {
				return []string{
					"container", "clusters", "get-credentials",
					cc.Name,
					"--zone", cc.Zone,
					"--project", cc.GcloudProjectName,
				}
			},
		},
		// Note: When running on GKE (Google Kubernetes Engine),
		// you may encounter a ‘permission denied’ error when creating some of these resources.
		// This is a nuance of the way GKE handles RBAC and IAM permissions,
		// and as such you should ‘elevate’ your own privileges to that of a ‘cluster-admin’ before running the above command.
		// If you have already run the above command, you should run them again after elevating your permissions:
		//
		//		kubectl create clusterrolebinding cluster-admin-binding \
		//    --clusterrole=cluster-admin \
		//    --user=$(gcloud config get-value core/account)
		//
		Command{
			name:    "gke-cluster-admin-role",
			rootCmd: "kubectl",
			generateArgs: func(cc *ClusterConfig) []string {
				return []string{
					"create", "clusterrolebinding", "cluster-admin-binding", "--clusterrole=cluster-admin",
					fmt.Sprintf("--user=%s", cc.Account),
				}
			},
		},
		Command{
			name:    "list-dns-server",
			rootCmd: "gcloud",
			generateArgs: func(cc *ClusterConfig) []string {
				return []string{
					"dns", "record-sets", "list",
					"--zone", cc.Name,
					"--format", "json",
				}
			},
			internal: true,
		},
		Command{
			name:    "create-dns-zone",
			rootCmd: "gcloud",
			generateArgs: func(cc *ClusterConfig) []string {
				return []string{
					"dns",
					"managed-zones", "create",
					cc.Name,
					"--dns-name", cc.DNSName,
					"--project", cc.GcloudProjectName,
					"--description", "kubepaas managed zone",
				}
			},
			runFn: func(cmd *Command) error {
				if cmd.succeed {
					printDNSServers(gcloudCmds, cc)
					return nil
				}

				if strings.Contains(cmd.stderr.Error(), "already exists") {
					printDNSServers(gcloudCmds, cc)
					return nil
				}

				return cmd.stderr
			},
		},
	}

	for _, cmd := range cmds {
		gcloudCmds.AddCmd(cmd)
	}

	return gcloudCmds, nil
}

func printDNSServers(gcs *cmdSet, cc *ClusterConfig) error {
	dnsCmd, err := gcs.getCommand("list-dns-server")
	if err != nil {
		return err
	}
	dnsCmd.execute(context.Background(), cc)
	if !dnsCmd.succeed {
		return dnsCmd.stderr
	}

	var dnsRecords DNSRecords
	var DNSServerAddrs string
	err = json.NewDecoder(strings.NewReader(dnsCmd.stdout)).Decode(&dnsRecords)
	if err != nil {
		return err
	}

	if len(dnsRecords) == 0 {
		return errors.New("the 'parameters.managedZone' resource named does not exists")
	}

	for _, rec := range dnsRecords {
		if rec.Type == "NS" {
			DNSServerAddrs = strings.Join(rec.Rrdatas, "\n")
		}
	}

	fmt.Println(DNSServerAddrs)
	return nil
}
