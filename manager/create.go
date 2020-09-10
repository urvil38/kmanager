package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/urvil38/kmanager/config"
	"github.com/urvil38/kmanager/questions"
)

func (c *Cluster) Create() error {

	if err := survey.Ask(questions.ClusterName, &c.Cc.Name); err != nil {
		return err
	}

	if err := survey.Ask(questions.DomainName, &c.Cc.DNSName); err != nil {
		return err
	}

	confPath, err := config.GetConfigPath(c.Cc.Name)
	if err != nil {
		return err
	}

	if confPath != "" {
		c.Cc.ConfPath = confPath
	}

	gCmds, err := c.Cc.initGCloudCmdSet()
	if err != nil {
		log.Fatal(err)
	}

	for _, cmd := range gCmds.cmds {
		if !cmd.internal {
			cmd.execute(context.Background(), c.Cc)
			if !cmd.succeed {
				// fmt.Println(cmd.stderr)
				continue
			}
		}
	}

	kCmds, err := c.Cc.initKubeCmdSet()
	if err != nil {
		log.Fatal(err)
	}

	for _, cmd := range kCmds.cmds {
		if !cmd.internal {
			cmd.execute(context.Background(), c.Cc)
			if !cmd.succeed {
				// fmt.Println(cmd.stderr)
				continue
			}
		}
	}

	_ = CreateServiceAccount(c.Cc.getServiceAccountOpts().DNSName)

	_ = BindServiceAccountToRole(c.Cc.GcloudProjectName, c.Cc.getServiceAccountOpts().DNS, "roles/dns.admin")

	_ = CreateServiceAccount(c.Cc.getServiceAccountOpts().StorageName)

	_ = BindServiceAccToBucket(c.Cc.getServiceAccountOpts().Storage, c.Cc.getStorageOpts().SourceCodeBucket, "objectCreator")

	_ = BindServiceAccToBucket(c.Cc.getServiceAccountOpts().Storage, c.Cc.getStorageOpts().CloudBuildBucket, "objectViewer")

	_ = CreateServiceAccount(c.Cc.getServiceAccountOpts().CloudBuildName)

	_ = BindServiceAccountToRole(c.Cc.GcloudProjectName, c.Cc.getServiceAccountOpts().CloudBuild, "roles/cloudbuild.builds.editor")

	_ = GenerateServiceAccountKey(c.Cc.getServiceAccountOpts().CloudBuild, filepath.Join(c.Cc.ConfPath, c.Cc.getServiceAccountOpts().CloudBuildName+".json"))

	_ = GenerateServiceAccountKey(c.Cc.getServiceAccountOpts().Storage, filepath.Join(c.Cc.ConfPath, c.Cc.getServiceAccountOpts().StorageName+".json"))

	_ = GenerateServiceAccountKey(c.Cc.getServiceAccountOpts().DNS, filepath.Join(c.Cc.ConfPath, c.Cc.getServiceAccountOpts().DNSName+".json"))

	err = c.Cc.configKubernetes()
	if err != nil {
		fmt.Println(err)
	}

	err = c.generateConfig()
	if err != nil {
		return err
	}

	return nil
}

func (m *Cluster) generateConfig() error {
	b, err := json.MarshalIndent(m.Cc, "", "    ")
	if err != nil {
		return err
	}
	fmt.Print(string(b))
	return nil
}
