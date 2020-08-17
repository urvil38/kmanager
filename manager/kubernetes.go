package manager

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"

	kh "github.com/urvil38/kmanager/http"
	"github.com/urvil38/kmanager/utils"
)

type KubeApp struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Apps       []App    `yaml:"apps"`
}

type App struct {
	Deprecated bool   `yaml:"deprecated"`
	Path       string `yaml:"path"`
	Name       string `yaml:"name"`
}

type Metadata struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

func (cc *ClusterConfig) initKubeCmdSet() (*cmdSet, error) {
	kubernetesCmds := newCmdSet(cc, "kubernetes")

	c := kh.NewHTTPClient(nil)

	kmangerIndexReq, err := http.NewRequest(http.MethodGet, "https://storage.googleapis.com/kmanager/index.yaml", nil)
	if err != nil {
		return nil, err
	}

	res, err := c.Do(kmangerIndexReq)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	index, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(index, &cc.KubeAppConfig)
	if err != nil {
		return nil, err
	}

	kConfigDir, err := utils.GetConfigPath(cc.Name)
	if err != nil {
		return nil, err
	}

	for _, app := range cc.KubeAppConfig.Apps {
		if app.Deprecated {
			continue
		}

		if cc.KubeAppMap == nil {
			cc.KubeAppMap = make(map[string]App)
		}

		_, isthere := cc.KubeAppMap[app.Name]
		if !isthere {
			cc.KubeAppMap[app.Name] = app
		}

		req, err := http.NewRequest(http.MethodGet, app.Path, nil)
		if err != nil {
			return nil, err
		}

		res, err := c.Do(req)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()

		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}

		cData := cc.valueFromTemplate(app, string(b))

		configFilePath := filepath.Join(kConfigDir, fmt.Sprintf("%s.yaml", app.Name))
		err = ioutil.WriteFile(configFilePath, []byte(cData), 0777)
		if err != nil {
			return nil, err
		}

		err = cc.kubectlRunAndWait(configFilePath)
		if err != nil {
			continue
		}

	}

	return kubernetesCmds, nil
}

func (cc *ClusterConfig) kubectlRunAndWait(filePath string) error {
	waitCmd := Command{
		name:    "wait-for-kubernetes-resources",
		rootCmd: "kubectl",
		args: []string{
			"wait",
			"--for=condition=Ready",
			"-f",
			filePath,
		},
	}
	applyCmd := Command{
		name:    "create-kubernetes-resources",
		rootCmd: "kubectl",
		args:    []string{"create", "-f", filePath},
		runFn: func(cmd *Command) error {
			waitCmd.execute(context.Background(), cc)
			if !cmd.succeed {
				return cmd.stderr
			}
			return nil
		},
	}

	applyCmd.execute(context.Background(), cc)
	if !applyCmd.succeed {
		//fmt.Println(applyCmd.stderr)
		return applyCmd.stderr
	}
	return nil
}

type externalDNSCfg struct {
	Domain  string
	Project string
	owner   string
}

func (cc *ClusterConfig) valueFromTemplate(app App, templateData string) string {
	switch app.Name {
	case "externalDNS":
		ec := externalDNSCfg{
			Domain:  cc.DNSName,
			Project: cc.GcloudProjectName,
			owner:   cc.Account,
		}
		return generateExternalDNSKubeConfig(ec, templateData)
	case "cluster-issuer":
		return generateClusterIssuerConfig(cc.Account, templateData)
	}
	return templateData
}

func generateExternalDNSKubeConfig(ec externalDNSCfg, templateData string) string {
	templateData = strings.Replace(templateData, "{{domain_name}}", ec.Domain, -1)
	templateData = strings.Replace(templateData, "{{project_name}}", ec.Project, -1)
	templateData = strings.Replace(templateData, "{{owner_id}}", ec.owner, -1)
	return templateData
}

func generateClusterIssuerConfig(email string, templateData string) string {
	templateData = strings.Replace(templateData, "{{email}}", email, -1)
	return templateData
}
