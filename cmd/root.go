package cmd

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const (
	banner = 
`
     __                                              
    / /______ ___  ____ _____  ____ _____ ____  _____
   / //_/ __ '__ \/ __ '/ __ \/ __ '/ __ '/ _ \/ ___/
  / ,< / / / / / / /_/ / / / / /_/ / /_/ /  __/ /    
 /_/|_/_/ /_/ /_/\__,_/_/ /_/\__,_/\__, /\___/_/     
                                  /____/             
					
`
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "kmanager",
	Short: "Cluster Manager of KubePAAS platform",
	Run: func(cmd *cobra.Command, args []string) {
		printBanner()
		cmd.Help()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func printBanner() {
	rand.Seed(time.Now().UnixNano())
	colorCounter := rand.Intn(7)
	fmt.Printf("\x1b[1;3%dm%v\x1b[0m", colorCounter+1, banner)
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
