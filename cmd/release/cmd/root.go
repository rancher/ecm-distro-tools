package cmd

import (
	"fmt"
	"os"

	"github.com/rancher/ecm-distro-tools/cmd/release/config"
	"github.com/spf13/cobra"
)

var dryRun *bool
var rootConfig *config.Config

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "release",
	Short: "Central command to perform RKE2 and K3s Releases",
	Long:  ``,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println("error: " + err.Error())
		os.Exit(1)
	}
}

func SetVersion(version string) {
	rootCmd.Version = version
}

func init() {
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Debug")

	configPath, err := config.DefaultConfigPath()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if os.Args[1] == "config" && os.Args[2] == "gen" {
		fmt.Println("running release config gen, skipping config load")
		return
	}
	conf, err := config.Load(configPath)
	if err != nil {
		fmt.Println("error loading config, check if it exisits at " + configPath + " and if not, use: release config gen")
		fmt.Println(err)
		os.Exit(1)
	}

	rootConfig = conf
}
