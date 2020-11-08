package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/urvil38/kmanager/config"
)

const (
	describeUsageStr = "describe [cluster name]"
)

var (
	describeUsageErrStr = fmt.Sprintf("expected '%s'.\ncluster name is a required argument for the describe command", describeUsageStr)
)

// describeCmd represents the describe command
var describeCmd = &cobra.Command{
	Use:   describeUsageStr,
	Short: "describe print out configuration of given cluster",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		err := validate(args, describeUsageErrStr)
		if err != nil {
			cmd.PrintErrln(err)
			os.Exit(1)
		}

		name := args[0]
		path, err := config.ClusterConfigPath(name)
		if err != nil {
			cmd.Println(err)
			os.Exit(1)
		}

		if _, err := os.Stat(filepath.Join(path, "config.json")); errors.Is(err, os.ErrNotExist) {
			cmd.Println("Unable to print configuration, config.json file not exists")
			os.Exit(1)
		}

		b, err := ioutil.ReadFile(filepath.Join(path, "config.json"))
		if err != nil {
			cmd.Printf("Unable to print configuration, %v", err)
			os.Exit(1)
		}

		fmt.Println(string(b))
	},
}

func init() {
	rootCmd.AddCommand(describeCmd)
}
