package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/rancher/ecm-distro-tools/release/charts"
	"github.com/rancher/ecm-distro-tools/release/k3s"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update files and other utilities",
}

var updateK3sCmd = &cobra.Command{
	Use:   "k3s",
	Short: "Update k3s files",
}

var updateK3sReferencesCmd = &cobra.Command{
	Use:   "references [version]",
	Short: "Update k8s and Go references in a k3s repo and create a PR",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("expected at least one argument: [version]")
		}

		version := args[0]

		k3sRelease, found := rootConfig.K3s.Versions[version]
		if !found {
			return errors.New("verify your config file, version not found: " + version)
		}

		ctx := context.Background()

		ghClient := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)

		return k3s.UpdateK3sReferences(ctx, ghClient, &k3sRelease, rootConfig.User)
	},
}

var updateChartsCmd = &cobra.Command{
	Use:   "charts [branch] [chart] [version]",
	Short: "Update charts files locally, stage and commit the changes.",
	RunE: func(cmd *cobra.Command, args []string) error {
		var branch, chart, version string

		if len(args) < 3 {
			return errors.New("expected at least three arguments: [branch] [chart] [version]")
		}

		branch = args[0]
		chart = args[1]
		version = args[2]

		config := rootConfig.Charts
		if config.Workspace == "" || config.ChartsForkURL == "" {
			return errors.New("verify your config file, chart configuration not implemented correctly, you must insert workspace path and your forked repo url")
		}

		ctx := context.Background()
		gh := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)

		output, err := charts.Update(ctx, gh, config, branch, chart, version)
		if err != nil {
			fmt.Println(output)
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.AddCommand(updateChartsCmd)
	updateCmd.AddCommand(updateK3sCmd)
	updateK3sCmd.AddCommand(updateK3sReferencesCmd)
}
