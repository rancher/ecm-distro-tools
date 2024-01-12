package cmd

import (
	"fmt"
	"os"

	"github.com/rancher/ecm-distro-tools/cmd/release/config"
	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			rootCmd.Help()
			os.Exit(0)
		}
	},
}

var genConfigSubCmd = &cobra.Command{
	Use:   "gen",
	Short: "generate config",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Here we are!")
	},
}

var viewConfigSubCmd = &cobra.Command{
	Use:   "view",
	Short: "view config",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		conf, err := rootConfig.String()
		if err != nil {
			return err
		}
		fmt.Println(conf)
		return nil
	},
}

var editConfigSubCmd = &cobra.Command{
	Use:   "edit",
	Short: "edit config",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		return config.OpenOnEditor()
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

const configViewTemplate = `RKE2 Version
------------
{{- range .RKE2.Versions }}
{{ . -}}+rke2r1
{{- end}}

`
