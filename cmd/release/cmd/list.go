package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/rancher/ecm-distro-tools/release/charts"
	"github.com/rancher/ecm-distro-tools/release/rancher"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List resources",
}

var rancherListSubCmd = &cobra.Command{
	Use:   "rancher",
	Short: "List Rancher Utilities",
}

var rancherListRCDepsSubCmd = &cobra.Command{
	Use:   "rc-deps [git-ref]",
	Short: "List Rancher RC Deps",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("expected at least one argument: [git-ref]")
		}
		rancherRCDeps, err := rancher.CheckRancherRCDeps(context.Background(), "rancher", args[0])
		if err != nil {
			return err
		}
		deps, err := rancherRCDeps.ToString()
		if err != nil {
			return err
		}
		fmt.Println(deps)
		return nil
	},
}

var chartsListSubCmd = &cobra.Command{
	Use:   "charts [branch] [charts](optional)",
	Short: "List Charts assets versions state for release process",
	RunE: func(cmd *cobra.Command, args []string) error {
		var branch, chart string

		if len(args) < 1 {
			return errors.New("expected at least one argument: [branch]")
		}
		branch = args[0]

		if len(args) > 1 {
			chart = args[1]
		}

		config := rootConfig.Charts
		if config.Workspace == "" || config.ChartsForkURL == "" {
			return errors.New("verify your config file, chart configuration not implemented correctly, you must insert workspace path and your forked repo url")
		}

		resp, err := charts.List(context.Background(), config, branch, chart)
		if err != nil {
			return err
		}

		fmt.Println(resp)
		return nil
	},
}

func init() {
	rancherListSubCmd.AddCommand(rancherListRCDepsSubCmd)
	listCmd.AddCommand(rancherListSubCmd)
	listCmd.AddCommand(chartsListSubCmd)
	rootCmd.AddCommand(listCmd)
}
