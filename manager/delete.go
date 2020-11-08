package manager

import (
	"context"
	"fmt"
	"os"
)

func DeleteCluster(name string) error {
	cc, err := getClusterConfig(name)
	if err != nil {
		return err
	}

	err = cc.deleteKubernetesCluster()
	if err != nil {
		fmt.Println(err)
	}

	err = cc.deleteStorageBuckets()
	if err != nil {
		fmt.Println(err)
	}

	err = cc.deleteServiceAccounts()
	if err != nil {
		fmt.Println(err)
	}

	err = os.RemoveAll(cc.ConfPath)
	if err != nil {
		return err
	}

	return nil
}

func (cc ClusterConfig) deleteKubernetesCluster() error {
	deleteKubernetesClusterCmd := Command{
		name:    "delete-kubernetes-cluster",
		rootCmd: "gcloud",
		args: []string{
			"container", "clusters", "delete", cc.Name, "--quiet", "--zone", cc.Zone, "--project", cc.GcloudProjectName,
		},
	}

	deleteKubernetesClusterCmd.execute(context.Background(), nil)
	if !deleteKubernetesClusterCmd.succeed {
		return deleteKubernetesClusterCmd.stderr
	}
	return nil
}

func (cc ClusterConfig) deleteStorageBuckets() error {

	if cc.Storage.CloudBuildBucket != "" {
		err := deleteBucket(cc.Storage.CloudBuildBucket)
		if err != nil {
			return err
		}
	}

	if cc.Storage.SourceCodeBucket != "" {
		err := deleteBucket(cc.Storage.SourceCodeBucket)
		if err != nil {
			return err
		}
	}

	return nil
}

func (cc ClusterConfig) deleteServiceAccounts() error {
	if cc.ServiceAccount.CloudBuild != "" {
		err := deleteServiceAccount(cc.ServiceAccount.CloudBuild)
		if err != nil {
			return err
		}
	}

	if cc.ServiceAccount.Storage != "" {
		err := deleteServiceAccount(cc.ServiceAccount.Storage)
		if err != nil {
			return err
		}
	}

	if cc.ServiceAccount.DNS != "" {
		err := deleteServiceAccount(cc.ServiceAccount.DNS)
		if err != nil {
			return err
		}
	}

	return nil
}

func deleteBucket(name string) error {
	deleteBucketCmd := Command{
		name:    "delete-storage-bucket",
		rootCmd: "gsutil",
		args: []string{
			"-m", "rm", "-r", fmt.Sprintf(storageBucketFmt, name),
		},
	}

	deleteBucketCmd.execute(context.Background(), nil)
	if !deleteBucketCmd.succeed {
		return deleteBucketCmd.stderr
	}

	return nil
}

func deleteServiceAccount(name string) error {
	deleteServiceAccountCmd := Command{
		name:    "delete-service-account",
		rootCmd: "gcloud",
		args: []string{
			"iam", "service-accounts",
			"delete", name,
		},
	}

	deleteServiceAccountCmd.execute(context.Background(), nil)
	if !deleteServiceAccountCmd.succeed {
		return deleteServiceAccountCmd.stderr
	}

	return nil
}
