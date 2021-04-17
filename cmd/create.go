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

	err = cluster.CreateServiceAccount(c.GetServiceAccountOpts().DNSName)
	if err != nil {
		fmt.Println(err)
	}

	err = cluster.BindServiceAccountToRole(c.GcloudProjectName, c.GetServiceAccountOpts().DNS, "roles/dns.admin")
	if err != nil {
		fmt.Println(err)
	}

	err = cluster.CreateServiceAccount(c.GetServiceAccountOpts().StorageName)
	if err != nil {
		fmt.Println("error:", err)
	}

	err = cluster.BindServiceAccToBucket(c.GetServiceAccountOpts().Storage, c.GetStorageOpts().SourceCodeBucket, "objectCreator")
	if err != nil {
		fmt.Println("error:", err)
	}

	err = cluster.BindServiceAccToBucket(c.GetServiceAccountOpts().Storage, c.GetStorageOpts().CloudBuildBucket, "objectViewer")
	if err != nil {
		fmt.Println("error:", err)
	}

	err = cluster.CreateServiceAccount(c.GetServiceAccountOpts().CloudBuildName)
	if err != nil {
		fmt.Println("error:", err)
	}

	err = cluster.BindServiceAccountToRole(c.GcloudProjectName, c.GetServiceAccountOpts().CloudBuild, "roles/cloudbuild.builds.editor")
	if err != nil {
		fmt.Println("error:", err)
	}

	err = cluster.GenerateServiceAccountKey(c.GetServiceAccountOpts().CloudBuild, filepath.Join(c.ConfPath, c.GetServiceAccountOpts().CloudBuildName+".json"))
	if err != nil {
		fmt.Println("error:", err)
	}

	err = cluster.GenerateServiceAccountKey(c.GetServiceAccountOpts().Storage, filepath.Join(c.ConfPath, c.GetServiceAccountOpts().StorageName+".json"))
	if err != nil {
		fmt.Println("error:", err)
	}

	err = cluster.GenerateServiceAccountKey(c.GetServiceAccountOpts().DNS, filepath.Join(c.ConfPath, c.GetServiceAccountOpts().DNSName+".json"))
	if err != nil {
		fmt.Println("error:", err)
	}

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
