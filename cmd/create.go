package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/urvil38/kmanager/cluster"
	"github.com/urvil38/kmanager/config"
	"github.com/urvil38/kmanager/questions"

	"github.com/spf13/cobra"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new kubepaas cluster",
	Run: func(cmd *cobra.Command, args []string) {
		err := CreateCluster()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func CreateCluster() error {
	c := new(cluster.Cluster)

	if err := survey.Ask(questions.ClusterName, &c.Name); err != nil {
		return err
	}

	if err := survey.Ask(questions.DomainName, &c.DNSName); err != nil {
		return err
	}

	confPath, err := config.CreateConfigDir(c.Name)
	if err != nil {
		return err
	}

	if confPath != "" {
		c.ConfPath = confPath
	}

	gCmds, err := c.InitGCloudCmdSet()
	if err != nil {
		log.Fatal(err)
	}

	for _, cmd := range gCmds.Cmds {
		if !cmd.Internal {
			cmd.Execute(context.Background(), c)
			if !cmd.Succeed {
				// fmt.Println(cmd.stderr)
				continue
			}
		}
	}

	kCmds, err := c.InitKubeCmdSet()
	if err != nil {
		log.Fatal(err)
	}

	for _, cmd := range kCmds.Cmds {
		if !cmd.Internal {
			cmd.Execute(context.Background(), c)
			if !cmd.Succeed {
				fmt.Println(cmd.Stderr)
				continue
			}
		}
	}

	_ = cluster.CreateServiceAccount(c.GetServiceAccountOpts().DNSName)

	_ = cluster.BindServiceAccountToRole(c.GcloudProjectName, c.GetServiceAccountOpts().DNS, "roles/dns.admin")

	_ = cluster.CreateServiceAccount(c.GetServiceAccountOpts().StorageName)

	_ = cluster.BindServiceAccToBucket(c.GetServiceAccountOpts().Storage, c.GetStorageOpts().SourceCodeBucket, "objectCreator")

	_ = cluster.BindServiceAccToBucket(c.GetServiceAccountOpts().Storage, c.GetStorageOpts().CloudBuildBucket, "objectViewer")

	_ = cluster.CreateServiceAccount(c.GetServiceAccountOpts().CloudBuildName)

	_ = cluster.BindServiceAccountToRole(c.GcloudProjectName, c.GetServiceAccountOpts().CloudBuild, "roles/cloudbuild.builds.editor")

	_ = cluster.GenerateServiceAccountKey(c.GetServiceAccountOpts().CloudBuild, filepath.Join(c.ConfPath, c.GetServiceAccountOpts().CloudBuildName+".json"))

	_ = cluster.GenerateServiceAccountKey(c.GetServiceAccountOpts().Storage, filepath.Join(c.ConfPath, c.GetServiceAccountOpts().StorageName+".json"))

	_ = cluster.GenerateServiceAccountKey(c.GetServiceAccountOpts().DNS, filepath.Join(c.ConfPath, c.GetServiceAccountOpts().DNSName+".json"))

	err = c.ConfigKubernetes()
	if err != nil {
		fmt.Println(err)
	}

	err = c.GenerateConfig()
	if err != nil {
		return err
	}

	return nil
}

func init() {
	rootCmd.AddCommand(createCmd)
}
