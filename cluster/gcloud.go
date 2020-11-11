package cluster

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
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

func (c *Cluster) InitGCloudCmdSet() (*CmdSet, error) {
	gcloudCmds := NewCmdSet(c, "gcloud")

	cmds := []Command{
		{
			Name:    "check-gcloud-login",
			RootCmd: "gcloud",
			Args:    []string{"config", "list", "--format", "json"},
			RunFn: func(cmd *Command) error {
				if cmd.Succeed {
					var ga GcloudAccount
					err := json.NewDecoder(strings.NewReader(cmd.Stdout)).Decode(&ga)
					if err != nil {
						return err
					}
					if ga.Core.Account == "" {
						loginCmd, err := gcloudCmds.GetCommand("gcloud-login")
						if err != nil {
							return err
						}
						loginCmd.Execute(context.Background(), c)
						if !loginCmd.Succeed {
							return loginCmd.Stderr
						}
					}
					if ga.Compute.Region != "" {
						c.Region = ga.Compute.Region
					}
					if ga.Compute.Zone != "" {
						c.Zone = ga.Compute.Zone
					}
					if ga.Core.Account != "" {
						c.Account = ga.Core.Account
					}
				} else {
					return cmd.Stderr
				}
				return nil
			},
		},
		{
			Name:     "gcloud-login",
			RootCmd:  "gcloud",
			Args:     []string{"auth", "login"},
			Internal: true,
		},
		{
			Name:    "list-gcloud-accounts",
			RootCmd: "gcloud",
			Args:    []string{"projects", "list", "--filter", "lifecycleState:ACTIVE", "--format", "json"},
			RunFn: func(cmd *Command) error {
				if cmd.Succeed {
					var pl ProjectList
					err := json.NewDecoder(strings.NewReader(cmd.Stdout)).Decode(&pl)
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
							c.GcloudProjectName = selectedProj[openIndex+1 : closeIndex]
						} else {
							return errors.New("Invalid project name")
						}
					} else {
						return errors.New("Invalid project name")
					}
				} else {
					return cmd.Stderr
				}
				return nil
			},
		},
		{
			Name:    "create-storage-bucket-soucecode",
			RootCmd: "gsutil",
			GenerateArgs: func(c *Cluster) []string {
				return []string{"mb", "-l", c.Region, "gs://" + c.GetStorageOpts().SourceCodeBucket}
			},
		},
		{
			Name:    "create-storage-bucket-cloudbuild-logs",
			RootCmd: "gsutil",
			GenerateArgs: func(c *Cluster) []string {
				return []string{"mb", "-l", c.Region, "gs://" + c.GetStorageOpts().CloudBuildBucket}
			},
		},
		{
			Name:    "list-dns-server",
			RootCmd: "gcloud",
			GenerateArgs: func(c *Cluster) []string {
				return []string{
					"dns", "record-sets", "list",
					"--zone", c.Name,
					"--format", "json",
				}
			},
			Internal: true,
		},
		{
			Name:    "create-dns-zone",
			RootCmd: "gcloud",
			GenerateArgs: func(c *Cluster) []string {
				return []string{
					"dns",
					"managed-zones", "create",
					c.Name,
					"--dns-name", c.DNSName,
					"--project", c.GcloudProjectName,
					"--description", "kubepaas managed zone",
				}
			},
			RunFn: func(cmd *Command) error {
				dnsListCmd, err := gcloudCmds.GetCommand("list-dns-server")
				if err != nil {
					return err
				}
				err = printDNSServers(dnsListCmd, c)
				if err != nil {
					return err
				}
				return nil
			},
		},
	}

	for _, cmd := range cmds {
		gcloudCmds.AddCmd(cmd)
	}

	return gcloudCmds, nil
}

func printDNSServers(dnsListCmd Command, c *Cluster) error {

	dnsListCmd.Execute(context.Background(), c)
	if !dnsListCmd.Succeed {
		return dnsListCmd.Stderr
	}

	var dnsRecords DNSRecords
	var DNSServerAddrs string
	err := json.NewDecoder(strings.NewReader(dnsListCmd.Stdout)).Decode(&dnsRecords)
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
	color.HiYellow("Add following nameserver to domain-name registar of your domain-name provider:")
	color.HiWhite(DNSServerAddrs)

	added := false
	for {
		err = survey.Ask(questions.DNSNameServerConfimation, &added)
		if err != nil {
			fmt.Println(err)
		}

		if added {
			return nil
		}
	}
}

func CreateServiceAccount(name string) error {
	cmd := Command{
		Name:    "create-service-account",
		RootCmd: "gcloud",
		Args: []string{
			"iam",
			"service-accounts",
			"create",
			name,
			"--display-name",
			name,
		},
	}

	cmd.Execute(context.Background(), nil)
	if !cmd.Succeed {
		return cmd.Stderr
	}

	return nil
}

func BindServiceAccToBucket(serviceAccount, bucket, permission string) error {
	cmd := Command{
		Name:    "bind-service-account-to-bucket",
		RootCmd: "gsutil",
		Args: []string{
			"iam", "ch", "serviceAccount:" + serviceAccount + ":" + permission,
			"gs://" + bucket,
		},
	}

	cmd.Execute(context.Background(), nil)
	if !cmd.Succeed {
		return cmd.Stderr
	}

	return nil
}

func BindServiceAccountToRole(gcloudProject, serviceAccount, role string) error {
	cmd := Command{
		Name:    "bind-service-account",
		RootCmd: "gcloud",
		Args: []string{
			"projects",
			"add-iam-policy-binding",
			gcloudProject,
			"--member",
			"serviceAccount:" + serviceAccount,
			"--role", role,
		},
	}

	cmd.Execute(context.Background(), nil)
	if !cmd.Succeed {
		return cmd.Stderr
	}

	return nil
}

func GenerateServiceAccountKey(serviceAccount, path string) error {
	cmd := Command{
		Name:    "generate-service-account-keys",
		RootCmd: "gcloud",
		Args: []string{
			"iam", "service-accounts", "keys", "create",
			"--iam-account", serviceAccount,
			path,
		},
	}

	cmd.Execute(context.Background(), nil)
	if !cmd.Succeed {
		return cmd.Stderr
	}
	return nil
}