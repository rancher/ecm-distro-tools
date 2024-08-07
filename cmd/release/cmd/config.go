package cmd

import (
	"fmt"

	"github.com/rancher/ecm-distro-tools/cmd/release/config"
	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage the release cli config file",
}

var genConfigSubCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generates a config file in the default location if it doesn't exists",
	RunE: func(cmd *cobra.Command, args []string) error {
		conf, err := config.ExampleConfig()
		if err != nil {
			return err
		}
		fmt.Println(conf)
		return nil
	},
}

var viewConfigSubCmd = &cobra.Command{
	Use:   "view",
	Short: "Print the parsed config to stdout",
	RunE: func(cmd *cobra.Command, args []string) error {
		return config.View(rootConfig)
	},
}

var editConfigSubCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open the config file in your default editor",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		return config.OpenOnEditor(configFile)
	},
}

func init() {
	rootCmd.AddCommand(configCmd)

	configCmd.AddCommand(genConfigSubCmd)
	configCmd.AddCommand(viewConfigSubCmd)
	configCmd.AddCommand(editConfigSubCmd)
}
