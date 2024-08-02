package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/rancher/ecm-distro-tools/cmd/release/config"
	"github.com/spf13/cobra"
)

var (
	debug        bool
	dryRun       bool
	rootConfig   *config.Config
	configFile   string
	stringConfig string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:          "release",
	Short:        "Central command to perform RKE2, K3s and Rancher Releases",
	SilenceUsage: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.OnInitialize(initConfig)
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}

func SetVersion(version string) {
	rootCmd.Version = version
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Debug")
	rootCmd.PersistentFlags().BoolVarP(&dryRun, "dry-run", "r", false, "Drun Run")
	rootCmd.PersistentFlags().StringVarP(&configFile, "config-file", "c", "$HOME/.ecm-distro-tools/config.json", "Path for the config.json file")
	rootCmd.PersistentFlags().StringVarP(&stringConfig, "config", "C", "", "JSON config string")
}

func initConfig() {
	if len(os.Args) >= 2 {
		if os.Args[1] == "config" && os.Args[2] == "gen" {
			return
		}
	}
	var conf *config.Config
	var err error
	if stringConfig != "" {
		log.Println("loading string config")
		conf, err = config.Read(strings.NewReader(stringConfig))
		if err != nil {
			fmt.Println("failed to load string config")
			panic(err)
		}
	} else {
		configFile = os.ExpandEnv(configFile)
		conf, err = config.Load(configFile)
		if err != nil {
			fmt.Println("failed to load config, use 'release config gen' to create a new one at: " + configFile)
			panic(err)
		}
	}

	rootConfig = conf

	if err := rootConfig.Validate(); err != nil {
		panic(err)
	}
}
