package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/urvil38/kmanager/manager"

	"github.com/spf13/cobra"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new kubepaas cluster",
	Run: func(cmd *cobra.Command, args []string) {
		m := manager.Cluster{
			Cc:        new(manager.ClusterConfig),
			CreatedAt: time.Now(),
		}
		err := m.Create()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
}
