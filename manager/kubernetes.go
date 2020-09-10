package manager

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"text/template"

	"gopkg.in/yaml.v2"

	"github.com/urvil38/kmanager/config"
	kh "github.com/urvil38/kmanager/http"
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

	cmds := []Command{
		{
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
		},
		{
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
		{
			name:    "gke-cluster-admin-role",
			rootCmd: "kubectl",
			generateArgs: func(cc *ClusterConfig) []string {
				return []string{
					"create", "clusterrolebinding", "cluster-admin-binding", "--clusterrole=cluster-admin",
					fmt.Sprintf("--user=%s", cc.Account),
				}
			},
		},
	}

	for _, cmd := range cmds {
		kubernetesCmds.AddCmd(cmd)
	}

	return kubernetesCmds, nil
}

func (cc *ClusterConfig) generateCertManagerSecret() error {
	createCmd := Command{
		name:    "cert-manager-clouddns-secert",
		rootCmd: "kubectl",
		generateArgs: func(cc *ClusterConfig) []string {
			return []string{
				"create", "secret", "generic", cc.getServiceAccountOpts().DNSName, fmt.Sprintf("--namespace=%s", "cert-manager"), fmt.Sprintf("--from-file=%s=%s", cc.getServiceAccountOpts().DNSName, filepath.Join(cc.ConfPath, cc.getServiceAccountOpts().DNSName+".json")),
			}
		},
	}

	createCmd.execute(context.Background(), cc)
	if !createCmd.succeed {
		return createCmd.stderr
	}
	return nil
}

func (cc *ClusterConfig) configKubernetes() error {
	c := kh.NewHTTPClient(nil)

	kmangerIndexReq, err := http.NewRequest(http.MethodGet, "https://storage.googleapis.com/kmanager/index.yaml", nil)
	if err != nil {
		return err
	}

	res, err := c.Do(kmangerIndexReq)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	index, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(index, &cc.KubeAppConfig)
	if err != nil {
		return err
	}

	kConfigDir, err := config.GetConfigPath(cc.Name)
	if err != nil {
		return err
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
			return err
		}

		res, err := c.Do(req)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}

		cData := cc.valueFromTemplate(app, string(b))

		configFilePath := filepath.Join(kConfigDir, fmt.Sprintf("%s.yaml", app.Name))
		err = ioutil.WriteFile(configFilePath, []byte(cData), 0777)
		if err != nil {
			return err
		}

		err = cc.kubectlRunAndWait(configFilePath)
		if err != nil {
			continue
		}
	}

	return nil
}

func (cc *ClusterConfig) kubectlRunAndWait(filePath string) error {
	waitCmd := Command{
		name:    "wait-for-kubernetes-resources",
		rootCmd: "kubectl",
		args: []string{
			"wait",
			"--for=condition=Ready",
			"--timeout=20s",
			"pods",
			"--all",
			"--all-namespaces=true",
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
	DomainName  string
	ProjectName string
	Email       string
}

type clusterIssuerCfg struct {
	Email                string
	ProjectName          string
	ServiceAccountSecret string
	SecretFileKey        string
}

func (cc *ClusterConfig) valueFromTemplate(app App, templateData string) string {
	switch app.Name {
	case "externalDNS":
		ec := externalDNSCfg{
			DomainName:  cc.DNSName,
			ProjectName: cc.GcloudProjectName,
			Email:       cc.Account,
		}
		cnf, err := generateExternalDNSKubeConfig(ec, templateData)
		if err != nil {
			fmt.Println("err:", err)
			return ""
		}
		return cnf
	case "cluster-issuer":
		ci := clusterIssuerCfg{
			Email:                cc.Account,
			ProjectName:          cc.GcloudProjectName,
			ServiceAccountSecret: cc.getServiceAccountOpts().DNSName,
			SecretFileKey:        cc.getServiceAccountOpts().DNSName,
		}
		_ = cc.generateCertManagerSecret()
		cnf, err := generateClusterIssuerConfig(ci, templateData)
		if err != nil {
			fmt.Println("err:", err)
			return ""
		}
		return cnf
	}
	return templateData
}

func generateExternalDNSKubeConfig(ec externalDNSCfg, tmpl string) (string, error) {
	t, err := template.New("tmpl").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, ec)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func generateClusterIssuerConfig(ci clusterIssuerCfg, tmpl string) (string, error) {
	t, err := template.New("tmpl").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, ci)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
