package cluster

import (
	"context"
	"fmt"
	"time"
)

func (c *Cluster) InitKubeCmdSet() (*CmdSet, error) {
	kubernetesCmds := NewCmdSet(c, "kubernetes")

	cmds := []Command{
		{
			Name:    "create-kubernetes-cluster",
			RootCmd: "gcloud",
			GenerateArgs: func(c *Cluster) []string {
				return []string{
					"container", "clusters", "create", c.Name,
					"--project", c.GcloudProjectName,
					"--zone", c.Zone,
					"--no-enable-basic-auth",
					"--cluster-version", "1.16.13-gke.401",
					"--machine-type", "n1-standard-1",
					"--image-type", "COS",
					"--disk-type", "pd-standard",
					"--disk-size=10",
					"--scopes",
					"https://www.googleapis.com/auth/devstorage.read_only,https://www.googleapis.com/auth/logging.write,https://www.googleapis.com/auth/monitoring,https://www.googleapis.com/auth/servicecontrol,https://www.googleapis.com/auth/service.management.readonly,https://www.googleapis.com/auth/trace.append,https://www.googleapis.com/auth/ndev.clouddns.readwrite",
					"--preemptible",
					"--num-nodes=2",
					"--network", fmt.Sprintf("projects/%s/global/networks/default", c.GcloudProjectName),
					"--subnetwork", fmt.Sprintf("projects/%s/regions/%s/subnetworks/default", c.GcloudProjectName, c.Region),
					"--addons", "HttpLoadBalancing",
				}
			},
		},
		{
			Name:    "get-kubernetes-credentials",
			RootCmd: "gcloud",
			GenerateArgs: func(c *Cluster) []string {
				return []string{
					"container", "clusters", "get-credentials",
					c.Name,
					"--zone", c.Zone,
					"--project", c.GcloudProjectName,
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
		{
			Name:    "gke-cluster-admin-role",
			RootCmd: "kubectl",
			GenerateArgs: func(c *Cluster) []string {
				return []string{
					"create", "clusterrolebinding", "cluster-admin-binding", "--clusterrole=cluster-admin",
					fmt.Sprintf("--user=%s", c.Account),
				}
			},
		},
	}

	for _, cmd := range cmds {
		kubernetesCmds.AddCmd(cmd)
	}

	return kubernetesCmds, nil
}

func (c *Cluster) createSecret(name, namespace, filepath string) error {
	createCmd := Command{
		Name:    name,
		RootCmd: "kubectl",
		GenerateArgs: func(c *Cluster) []string {
			if filepath != "" {
				return []string{
					"create", "secret", "generic", name, fmt.Sprintf("--namespace=%s", namespace), fmt.Sprintf("--from-file=%s=%s", name, filepath),
				}
			} else {
				return []string{
					"create", "secret", "generic", name, fmt.Sprintf("--namespace=%s", namespace),
				}
			}
		},
	}

	createCmd.Execute(context.Background(), c)
	if !createCmd.Succeed {
		return createCmd.Stderr
	}
	return nil
}

func (c *Cluster) kubectlRunAndWait(filePath string, appName string) error {
	waitCmd := Command{
		Name:    "wait-for-kubernetes-resources",
		RootCmd: "kubectl",
		Args: []string{
			"wait",
			"--for=condition=Ready",
			"--timeout=20s",
			"pods",
			"--all",
			"--all-namespaces=true",
		},
	}

	applyCmd := Command{
		Name:    "create-kubernetes-resources",
		RootCmd: "kubectl",
		Args:    []string{"create", "-f", filePath},
		RunFn: func(cmd *Command) error {
			if !cmd.Succeed {
				return cmd.Stderr
			}

			if appName == "wildcard-cert" {
				checkCmd := Command{
					Name:    "check-kubernetes-secret",
					RootCmd: "kubectl",
					Args: []string{
						"get",
						"secret",
						"wildcard-cert-secret",
					},
				}
				timer := time.NewTimer(5 * time.Minute)
			outer:
				for {
					select {
					case <-timer.C:
						timer.Stop()
						fmt.Println("timeout: kubectl get secret wildcard-cert-secret")
						_ = c.CreateFakeSecret()
						break outer
					default:
						checkCmd.Execute(context.Background(), c)
						if checkCmd.Succeed {
							break outer
						}
						time.Sleep(5 * time.Second)
					}
				}
			} else {
				waitCmd.Execute(context.Background(), c)
			}
			return nil
		},
	}

	applyCmd.Execute(context.Background(), c)
	if !applyCmd.Succeed {
		fmt.Println(applyCmd.Stderr)
	}
	return nil
}

func (c *Cluster) createNamespace(name string) error {
	nsCmd := Command{
		Name:    "create-kubernetes-namespace",
		RootCmd: "kubectl",
		Args:    []string{"create", "ns", name},
	}

	nsCmd.Execute(context.Background(), c)
	if !nsCmd.Succeed {
		return nsCmd.Stderr
	}
	return nil
}
