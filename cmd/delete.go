package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/urvil38/kmanager/manager"
)

const (
	deleteUsageStr = "delete [cluster name]"
)

var (
	deleteUsageErrStr = fmt.Sprintf("expected '%s'.\ncluster name is a required argument for the delete command", deleteUsageStr)
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
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
			name := args[0]
			err := manager.DeleteCluster(name)
			if err != nil {
				cmd.PrintErrln(err)
				os.Exit(1)
			}
		}
	},
}

func validate(args []string, usage string) error {
	if len(args) == 0 {
		return errors.New(usage)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
