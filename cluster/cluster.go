package cluster

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/urvil38/kmanager/config"
)

const (
	ServiceAccountFmt = `%s@%s.iam.gserviceaccount.com`
	StorageBucketFmt  = `gs://%s`
)

type Cluster struct {
	Name              string         `json:"cluster_name" survey:"clusterName"`
	GcloudProjectName string         `json:"project_name" survey:"project"`
	Account           string         `json:"account"`
	Region            string         `json:"region"`
	Zone              string         `json:"zone"`
	DNSName           string         `json:"dns_name" survey:"dnsName"`
	Storage           Storage        `json:"storage"`
	ServiceAccount    ServiceAccount `json:"service_account"`
	KubeAppConfig     *KubeApp       `json:"kubeapp"`
	KubeAppMap        map[string]App `json:"-"`
	ConfPath          string         `json:"config_path"`
}

type Storage struct {
	CloudBuildBucket string `json:"cloudbuild_bucket_name"`
	SourceCodeBucket string `json:"sourcecode_bucket_name"`
}

type ServiceAccount struct {
	CloudBuildName string `json:"cloudbuild_serviceaccount_name"`
	CloudBuild     string `json:"cloudbuild_serviceaccount"`
	StorageName    string `json:"storage_serviceaccount_name"`
	Storage        string `json:"storage_serviceaccount"`
	DNSName        string `json:"clouddns_serviceaccount_name"`
	DNS            string `json:"clouddns_serviceaccount"`
}

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

func (c *Cluster) GetStorageOpts() Storage {
	s := Storage{
		CloudBuildBucket: fmt.Sprintf("%s-%s", c.Name, "cloudbuild-logs"),
		SourceCodeBucket: fmt.Sprintf("%s-%s", c.Name, "sourcecode"),
	}
	c.Storage = s
	return s
}

func (c *Cluster) GetServiceAccountOpts() ServiceAccount {
	cloudBuildSaName := fmt.Sprintf("%s-%s", c.Name, "cloudbuild")
	storageSaName := fmt.Sprintf("%s-%s", c.Name, "storage")
	clouddnsSaName := fmt.Sprintf("%s-%s", c.Name, "cert-clouddns")
	s := ServiceAccount{
		CloudBuildName: cloudBuildSaName,
		CloudBuild:     fmt.Sprintf(ServiceAccountFmt, cloudBuildSaName, c.GcloudProjectName),
		StorageName:    storageSaName,
		Storage:        fmt.Sprintf(ServiceAccountFmt, storageSaName, c.GcloudProjectName),
		DNSName:        clouddnsSaName,
		DNS:            fmt.Sprintf(ServiceAccountFmt, clouddnsSaName, c.GcloudProjectName),
	}
	c.ServiceAccount = s
	return s
}

func Get(name string) (Cluster, error) {
	var cc Cluster

	configFilePath, err := config.ClusterPath(name)
	if err != nil {
		return cc, err
	}

	if _, err := os.Stat(filepath.Join(configFilePath, "config.json")); errors.Is(err, os.ErrNotExist) {
		return cc, err
	}

	b, err := ioutil.ReadFile(filepath.Join(configFilePath, "config.json"))
	if err != nil {
		return cc, err
	}

	err = json.Unmarshal(b, &cc)
	if err != nil {
		return cc, err
	}

	return cc, nil
}

func (c Cluster) GenerateConfig() error {
	b, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(c.ConfPath, "config.json"), b, 0600)
	if err != nil {
		return err
	}
	return nil
}
