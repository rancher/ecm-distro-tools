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
		if err := config.Generate(configPath); err != nil {
			return err
		}

		fmt.Println("config generated")
		fmt.Println("to view it, run: release config view")
		fmt.Println("to edit it, run: release config edit")

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
		return config.OpenOnEditor(configPath)
	},
}

func init() {
	rootCmd.AddCommand(configCmd)

	configCmd.AddCommand(genConfigSubCmd)
	configCmd.AddCommand(viewConfigSubCmd)
	configCmd.AddCommand(editConfigSubCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// configCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// configCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
