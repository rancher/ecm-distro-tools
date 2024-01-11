package cmd

import (
	"fmt"
	"os"

	"github.com/rancher/ecm-distro-tools/cmd/release/config"
	"github.com/sirupsen/logrus"
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
		logrus.Fatal(err)
	}
	conf, err := config.Load(configPath)
	if err != nil {
		logrus.Fatal(err)
	}

	rootConfig = conf
}
