package manager

import (
	"fmt"
)

type ClusterConfig struct {
	Name              string `json:"cluster_name" survey:"clusterName"`
	GcloudProjectName string `json:"project_name" survey:"project"`
	Account           string `json:"account"`
	Region            string `json:"region"`
	Zone              string `json:"zone"`
	DNSName           string `json:"dns_name" survey:"dnsName"`
	Storage           Storage
	KubeAppConfig     *KubeApp
	KubeAppMap        map[string]App
}

type Storage struct {
	CloudBuildBucket string `json:"cloudbuild_bucket_name"`
	SourceCodeBucket string `json:"sourcecode_bucket_name"`
}

type Manager struct {
	Cc   *ClusterConfig
	Name string
}

func (cc *ClusterConfig) getStorageOpts() Storage {
	s := Storage{
		CloudBuildBucket: fmt.Sprintf("%s-%s", cc.Name, "cloudbuild-logs"),
		SourceCodeBucket: fmt.Sprintf("%s-%s", cc.Name, "sourcecode"),
	}
	cc.Storage = s
	return s
}
