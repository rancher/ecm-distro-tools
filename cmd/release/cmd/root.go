package cmd

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/rancher/ecm-distro-tools/cmd/release/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	v          *viper.Viper
	debug      bool
	dryRun     bool
	rootConfig *config.Config
	configPath string
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
	rootCmd.PersistentFlags().StringVarP(&configPath, "config-path", "c", "$HOME/.ecm-distro-tools", "path for the config.json file")

	v = viper.NewWithOptions(viper.KeyDelimiter("::"))
	v.SetConfigName("config")
	v.SetConfigType("json")
	v.AddConfigPath(configPath)
	err := v.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
	if err = v.Unmarshal(&rootConfig); err != nil {
		fmt.Println("failed to load config, use 'release config gen' to create a new one at: " + configPath)
		panic(err)
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err = validate.Struct(rootConfig); err != nil {
		panic(err)
	}
}
