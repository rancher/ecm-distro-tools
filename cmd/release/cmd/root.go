package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/rancher/ecm-distro-tools/cmd/release/config"
	"github.com/spf13/cobra"
)

var (
	debug        bool
	dryRun       bool
	rootConfig   *config.Config
	verbose      bool
	configFile   string
	stringConfig string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:           "release",
	Short:         "Central command to perform RKE2, K3s, Rancher and Chart Releases",
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.OnInitialize(initConfig)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println("error: ", err)
		os.Exit(1)
	}
}

func SetVersion(version string) {
	rootCmd.Version = version
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "D", false, "Debug")
	rootCmd.PersistentFlags().BoolVarP(&dryRun, "dry-run", "R", false, "Dry Run")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "V", false, "Verbose output")
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
		conf, err = config.Read(strings.NewReader(stringConfig))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	} else {
		configFile = os.ExpandEnv(configFile)
		conf, err = config.Load(configFile)
		if err != nil {
			fmt.Println("failed to load config, use 'release config gen' to create a new one at: " + configFile)
			fmt.Println(err)
			os.Exit(1)
		}
	}

	rootConfig = conf
}
