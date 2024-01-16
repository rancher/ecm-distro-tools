package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rancher/ecm-distro-tools/cmd/release/config"
	"github.com/spf13/cobra"
)

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
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func SetVersion(version string) {
	rootCmd.Version = version
}

func init() {
	rootCmd.Flags().BoolP("debug", "d", false, "Debug")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	const (
		ecmDistroDir   = ".ecm-distro-tools"
		configFileName = "config.json"
	)
	configFile := filepath.Join(homeDir, ecmDistroDir, configFileName)

	conf, err := config.Load(configFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	rootConfig = conf
}
