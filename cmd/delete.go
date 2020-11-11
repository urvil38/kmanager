package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/urvil38/kmanager/cluster"
)

const (
	deleteUsageStr = "delete [cluster name]"
)

var (
	deleteUsageErrStr = fmt.Sprintf("expected '%s'.\ncluster name is a required argument for the delete command", deleteUsageStr)
)

type DeleteOptions struct {
	ClusterName  string
	LeaveDNSZone bool
}

func newDeleteOptions() *DeleteOptions {
	return &DeleteOptions{}
}

// deleteCmd represents the delete command
func newDeleteCmd() *cobra.Command {
	o := newDeleteOptions()

	cmd := &cobra.Command{
		Use:   deleteUsageStr,
		Short: "delete will delete the cluster of given name",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			err := validate(args, deleteUsageErrStr)
			if err != nil {
				cmd.PrintErrln(err)
				os.Exit(1)
			}

			if len(args) > 0 {
				o.ClusterName = args[0]
				err := deleteCluster(*o)
				if err != nil {
					cmd.PrintErrln(err)
					os.Exit(1)
				}
			}
		},
	}

	o.addFlags(cmd)
	return cmd
}

func (o *DeleteOptions) addFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&o.LeaveDNSZone, "leave-dns-zone", false, "not delete the dns zone attached to this cluster")
}

func validate(args []string, usage string) error {
	if len(args) == 0 {
		return errors.New(usage)
	}

	return nil
}

func deleteCluster(o DeleteOptions) error {
	cc, err := cluster.Get(o.ClusterName)
	if err != nil {
		return err
	}

	err = DeleteKubernetesCluster(cc)
	if err != nil {
		fmt.Println(err)
	}

	if !o.LeaveDNSZone {
		err = DeleteDNSZone(cc)
		if err != nil {
			fmt.Println(err)
		}
	}

	err = DeleteStorageBuckets(cc)
	if err != nil {
		fmt.Println(err)
	}

	err = DeleteServiceAccounts(cc)
	if err != nil {
		fmt.Println(err)
	}

	err = os.RemoveAll(cc.ConfPath)
	if err != nil {
		return err
	}

	return nil
}

func DeleteKubernetesCluster(c cluster.Cluster) error {
	deleteKubernetesClusterCmd := cluster.Command{
		Name:    "delete-kubernetes-cluster",
		RootCmd: "gcloud",
		Args: []string{
			"container", "clusters", "delete", c.Name, "--quiet", "--zone", c.Zone, "--project", c.GcloudProjectName,
		},
	}

	deleteKubernetesClusterCmd.Execute(context.Background(), nil)
	if !deleteKubernetesClusterCmd.Succeed {
		return deleteKubernetesClusterCmd.Stderr
	}
	return nil
}

func DeleteDNSZone(c cluster.Cluster) error {
	deleteDNSZoneCmd := cluster.Command{
		Name:    "delete-dns-zone",
		RootCmd: "gcloud",
		Args: []string{
			"dns", "managed-zones", "delete", c.Name,
		},
	}

	deleteDNSZoneCmd.Execute(context.Background(), nil)
	if !deleteDNSZoneCmd.Succeed {
		return deleteDNSZoneCmd.Stderr
	}
	return nil
}

func DeleteStorageBuckets(c cluster.Cluster) error {

	if c.Storage.CloudBuildBucket != "" {
		err := deleteBucket(c.Storage.CloudBuildBucket)
		if err != nil {
			return err
		}
	}

	if c.Storage.SourceCodeBucket != "" {
		err := deleteBucket(c.Storage.SourceCodeBucket)
		if err != nil {
			return err
		}
	}

	return nil
}

func DeleteServiceAccounts(c cluster.Cluster) error {
	if c.ServiceAccount.CloudBuild != "" {
		err := deleteServiceAccount(c.ServiceAccount.CloudBuild)
		if err != nil {
			return err
		}
	}

	if c.ServiceAccount.Storage != "" {
		err := deleteServiceAccount(c.ServiceAccount.Storage)
		if err != nil {
			return err
		}
	}

	if c.ServiceAccount.DNS != "" {
		err := deleteServiceAccount(c.ServiceAccount.DNS)
		if err != nil {
			return err
		}
	}

	return nil
}

func deleteBucket(name string) error {
	deleteBucketCmd := cluster.Command{
		Name:    "delete-storage-bucket",
		RootCmd: "gsutil",
		Args: []string{
			"-m", "rm", "-r", fmt.Sprintf(cluster.StorageBucketFmt, name),
		},
	}

	deleteBucketCmd.Execute(context.Background(), nil)
	if !deleteBucketCmd.Succeed {
		return deleteBucketCmd.Stderr
	}

	return nil
}

func deleteServiceAccount(name string) error {
	deleteServiceAccountCmd := cluster.Command{
		Name:    "delete-service-account",
		RootCmd: "gcloud",
		Args: []string{
			"iam", "service-accounts",
			"delete", name,
		},
	}

	deleteServiceAccountCmd.Execute(context.Background(), nil)
	if !deleteServiceAccountCmd.Succeed {
		return deleteServiceAccountCmd.Stderr
	}

	return nil
}

func init() {
	rootCmd.AddCommand(newDeleteCmd())
}
