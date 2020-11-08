package manager

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/urvil38/kmanager/config"
)

const (
	serviceAccountFmt = `%s@%s.iam.gserviceaccount.com`
	storageBucketFmt  = `gs://%s`
)

type ClusterConfig struct {
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

type Cluster struct {
	Cc        *ClusterConfig
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (cc *ClusterConfig) getStorageOpts() Storage {
	s := Storage{
		CloudBuildBucket: fmt.Sprintf("%s-%s", cc.Name, "cloudbuild-logs"),
		SourceCodeBucket: fmt.Sprintf("%s-%s", cc.Name, "sourcecode"),
	}
	cc.Storage = s
	return s
}

func (cc *ClusterConfig) getServiceAccountOpts() ServiceAccount {
	cloudBuildSaName := fmt.Sprintf("%s-%s", cc.Name, "cloudbuild")
	storageSaName := fmt.Sprintf("%s-%s", cc.Name, "storage")
	clouddnsSaName := fmt.Sprintf("%s-%s", cc.Name, "cert-manager-clouddns")
	s := ServiceAccount{
		CloudBuildName: cloudBuildSaName,
		CloudBuild:     fmt.Sprintf(serviceAccountFmt, cloudBuildSaName, cc.GcloudProjectName),
		StorageName:    storageSaName,
		Storage:        fmt.Sprintf(serviceAccountFmt, storageSaName, cc.GcloudProjectName),
		DNSName:        clouddnsSaName,
		DNS:            fmt.Sprintf(serviceAccountFmt, clouddnsSaName, cc.GcloudProjectName),
	}
	cc.ServiceAccount = s
	return s
}

func getClusterConfig(name string) (ClusterConfig, error) {
	var cc ClusterConfig

	configFilePath, err := config.ClusterConfigPath(name)
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
