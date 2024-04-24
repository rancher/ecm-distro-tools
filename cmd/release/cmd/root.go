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
	Use:          "release",
	Short:        "Central command to perform RKE2, K3s and Rancher Releases",
	SilenceUsage: true,
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

	if len(os.Args) >= 2 {
		if os.Args[1] == "config" && os.Args[2] == "gen" {
			fmt.Println("running release config gen, skipping config load")
			return
		}
	}
	conf, err := config.Load(configPath)
	if err != nil {
		fmt.Println("failed to load config, use 'release config gen' to create a new one at: " + configPath)
		fmt.Println(err)
		os.Exit(1)
	}

	rootConfig = conf
}
