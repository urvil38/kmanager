package cluster

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"text/template"

	"github.com/urvil38/kmanager/config"
	kh "github.com/urvil38/kmanager/http"
	"gopkg.in/yaml.v2"
)

func (c *Cluster) ConfigKubernetes() error {
	client := kh.NewHTTPClient(nil)

	kmangerIndexReq, err := http.NewRequest(http.MethodGet, "https://storage.googleapis.com/kmanager/index.yaml", nil)
	if err != nil {
		return err
	}

	res, err := client.Do(kmangerIndexReq)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	index, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(index, &c.KubeAppConfig)
	if err != nil {
		return err
	}

	kConfigDir, err := config.CreateConfigDir(c.Name)
	if err != nil {
		return err
	}

	for _, app := range c.KubeAppConfig.Apps {
		if app.Deprecated {
			continue
		}

		if c.KubeAppMap == nil {
			c.KubeAppMap = make(map[string]App)
		}

		_, isthere := c.KubeAppMap[app.Name]
		if !isthere {
			c.KubeAppMap[app.Name] = app
		}

		req, err := http.NewRequest(http.MethodGet, app.Path, nil)
		if err != nil {
			return err
		}

		res, err := client.Do(req)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}

		cData, err := c.valueFromTemplate(app, string(b))
		if err != nil {
			fmt.Print("err:", err)
			return err
		}

		configFilePath := filepath.Join(kConfigDir, fmt.Sprintf("%s.yaml", app.Name))
		err = ioutil.WriteFile(configFilePath, []byte(cData), 0777)
		if err != nil {
			return err
		}

		err = c.kubectlRunAndWait(configFilePath, app.Name)
		if err != nil {
			continue
		}
	}

	return nil
}

type externalDNSCfg struct {
	IngressControllerService string
	DomainName               string
	ProjectName              string
	Email                    string
}

type clusterIssuerCfg struct {
	Email                string
	ProjectName          string
	ServiceAccountSecret string
	SecretFileKey        string
}

type wildCardCertCfg struct {
	DNSName string
}

type generatorCfg struct {
	ClusterIssuer string
	DNSName       string
	Envs          []env
}

type env struct {
	Name  string
	Value string
}

func (c *Cluster) valueFromTemplate(app App, templateData string) (string, error) {
	switch app.Name {
	case "externalDNS":
		ec := externalDNSCfg{
			IngressControllerService: "ingress-controller-nginx-ingress",
			DomainName:               c.DNSName,
			ProjectName:              c.GcloudProjectName,
			Email:                    c.Account,
		}
		cnf, err := generateKubeAppConfigFromTemplate(ec, templateData)
		if err != nil {
			return "", err
		}
		return cnf, nil
	case "wildcard-cert":
		wc := wildCardCertCfg{
			DNSName: "*." + c.DNSName,
		}
		cnf, err := generateKubeAppConfigFromTemplate(wc, templateData)
		if err != nil {
			return "", err
		}
		return cnf, nil
	case "cluster-issuer":
		ci := clusterIssuerCfg{
			Email:                c.Account,
			ProjectName:          c.GcloudProjectName,
			ServiceAccountSecret: c.GetServiceAccountOpts().DNSName,
			SecretFileKey:        c.GetServiceAccountOpts().DNSName,
		}
		_ = c.createSecret(
			c.GetServiceAccountOpts().DNSName,
			"cert-manager",
			filepath.Join(c.ConfPath, c.GetServiceAccountOpts().DNSName+".json"),
		)
		cnf, err := generateKubeAppConfigFromTemplate(ci, templateData)
		if err != nil {
			return "", err
		}
		return cnf, nil
	case "generator":
		gn := generatorCfg{
			ClusterIssuer: "letsencrypt-prod",
			DNSName:       "generator." + c.DNSName,
			Envs: []env{
				{
					Name:  "FLASK_ENV",
					Value: "production",
				},
				{
					Name:  "GCP_PROJECT",
					Value: c.GcloudProjectName,
				},
				{
					Name:  "SOURCE_BUCKET",
					Value: c.Storage.SourceCodeBucket,
				},
				{
					Name:  "CLOUDBUILD_BUCKET",
					Value: c.Storage.CloudBuildBucket,
				},
				{
					Name:  "ISSUER_NAME",
					Value: "letsencrypt-prod",
				},
				{
					Name:  "CLUSTER_NAME",
					Value: c.Name,
				},
				{
					Name:  "COMPUTE_ZONE",
					Value: c.Zone,
				},
				{
					Name:  "DNS_NAME",
					Value: c.DNSName,
				},
			},
		}
		err := c.createNamespace("generator")
		if err != nil {
			fmt.Println("err:", err)
		}

		err = c.createSecret(
			"cloudbuild-secret",
			"generator",
			filepath.Join(c.ConfPath, c.GetServiceAccountOpts().CloudBuildName+".json"),
		)
		if err != nil {
			fmt.Println("err:", err)
		}

		err = c.createSecret(
			"cloudstorage-secret",
			"generator",
			filepath.Join(c.ConfPath, c.GetServiceAccountOpts().StorageName+".json"),
		)
		if err != nil {
			fmt.Println("err:", err)
		}

		cnf, err := generateKubeAppConfigFromTemplate(gn, templateData)
		if err != nil {
			return "", err
		}
		return cnf, nil
	}
	return templateData, nil
}

func generateKubeAppConfigFromTemplate(cf interface{}, tmpl string) (string, error) {
	t, err := template.New("tmpl").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, cf)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
