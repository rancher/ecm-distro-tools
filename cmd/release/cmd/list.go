package cmd

import (
	"context"
	"errors"
	"fmt"

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

func init() {
	rancherListSubCmd.AddCommand(rancherListRCDepsSubCmd)
	listCmd.AddCommand(rancherListSubCmd)
	rootCmd.AddCommand(listCmd)
}
