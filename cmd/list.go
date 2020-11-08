package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"
	"github.com/urvil38/kmanager/config"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List cluster managed by current kmanager",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		clusters, err := listClusters()
		if err != nil {
			cmd.PrintErrln("Unable to list clusters, ", err)
			os.Exit(1)
		}

		if len(clusters) == 0 {
			fmt.Println("No cluster found!")
		} else {
			fmt.Println("clusters:")
			fmt.Println("---------")
			for _, c := range clusters {
				fmt.Println(c)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func listClusters() ([]string, error) {
	kmanagerConfPath, err := config.KmanagerConfigPath()
	if err != nil {
		return nil, err
	}

	fis, err := ioutil.ReadDir(kmanagerConfPath)
	if err != nil {
		return nil, err
	}

	var clusters []string

	for _, fi := range fis {
		if fi.IsDir() {
			clusters = append(clusters, fi.Name())
		}
	}

	return clusters, nil
}
