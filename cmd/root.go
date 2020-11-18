package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "kmanager",
	Short: "A brief description of your application",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	kubectlCmd := exec.Command("kubectl", "version", "--client", "--short")

	err := kubectlCmd.Run()
	if err != nil {
		color.Red("Seems like you don't have \"kubectl\" installed on your computer!")
		color.HiYellow("=> Installation Guide: https://kubernetes.io/docs/tasks/tools/install-kubectl")
		os.Exit(1)
	}

	if kubectlCmd.Stderr != nil {
		color.Red("Seems like you don't have \"kubectl\" installed on your computer!")
		color.HiYellow("=> Installation Guide: https://kubernetes.io/docs/tasks/tools/install-kubectl")
		os.Exit(1)
	}

	gcloudCmd := exec.Command("gcloud", "--version")

	err = gcloudCmd.Run()
	if err != nil {
		color.Red("Seems like you don't have google cloud sdk (gcloud) installed on your computer!")
		color.HiYellow("=> Installation Guide: https://cloud.google.com/sdk/docs/install")
		os.Exit(1)
	}

	if gcloudCmd.Stderr != nil {
		color.Red("Seems like you don't have google cloud sdk (gcloud) installed on your computer!")
		color.HiYellow("=> Installation Guide: https://cloud.google.com/sdk/docs/install")
		os.Exit(1)
	}
}
