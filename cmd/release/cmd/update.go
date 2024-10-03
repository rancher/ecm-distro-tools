package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/rancher/ecm-distro-tools/release/charts"
	"github.com/rancher/ecm-distro-tools/release/k3s"
	"github.com/rancher/ecm-distro-tools/release/rancher"
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
	Use:     "charts [branch-line] [chart] [version]",
	Short:   "Update charts files locally, stage and commit the changes.",
	Example: "release update charts 2.9 rancher-istio 104.0.0+up1.21.1",
	Args: func(cmd *cobra.Command, args []string) error {
		if err := validateChartConfig(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if len(args) != 3 {
			return errors.New("expected 3 arguments: [branch-line] [chart] [version]")
		}

		var branch, chart, version string
		branch = args[0]
		chart = args[1]
		version = args[2]

		if found := charts.IsBranchAvailable(branch, rootConfig.Charts.BranchLines); !found {
			return errors.New("branch not available: " + branch)
		}

		found, err := charts.IsChartAvailable(context.Background(), rootConfig.Charts, chart)
		if err != nil {
			return err
		}
		if !found {
			return errors.New("chart not available: " + chart)
		}

		found, err = charts.IsVersionAvailable(context.Background(), rootConfig.Charts, chart, version)
		if err != nil {
			return err
		}
		if !found {
			return errors.New("version not available: " + version)
		}

		return nil
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if err := validateChartConfig(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if len(args) == 0 {
			return rootConfig.Charts.BranchLines, cobra.ShellCompDirectiveNoFileComp
		} else if len(args) == 1 {
			chArgs, err := charts.ChartArgs(context.Background(), rootConfig.Charts)
			if err != nil {
				fmt.Printf("failed to get available charts: %v\n", err)
				os.Exit(1)
			}

			return chArgs, cobra.ShellCompDirectiveNoFileComp
		} else if len(args) == 2 {
			vArgs, err := charts.VersionArgs(context.Background(), rootConfig.Charts, args[1])
			if err != nil {
				fmt.Printf("failed to get available versions: %v", err)
				os.Exit(1)
			}

			return vArgs, cobra.ShellCompDirectiveNoFileComp
		}

		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var branch, chart, version string
		branch = args[0]
		chart = args[1]
		version = args[2]

		output, err := charts.Update(context.Background(), rootConfig.Charts, branch, chart, version)
		if err != nil {
			return err
		}

		fmt.Println(output)
		return nil
	},
}

var updateRancherCmd = &cobra.Command{
	Use:   "rancher",
	Short: "Update rancher files",
}

var updateRancherDashboardCmd = &cobra.Command{
	Use:   "dashboard [version]",
	Short: "Update Rancher's Dashboard and UI references and create a PR",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("expected at least one argument: [version]")
		}

		version := args[0]

		// checking if the provided version is valid
		_, err := semver.NewVersion(version)
		if err != nil {
			return err
		}

		versionTrimmed, _, _ := strings.Cut(version, "-rc")
		versionTrimmed, _, _ = strings.Cut(versionTrimmed, "-alpha")

		dashboardRelease, found := rootConfig.Dashboard.Versions[versionTrimmed]
		if !found {
			return errors.New("verify your config file, version not found: " + version)
		}

		dashboardRelease.Tag = version

		ctx := context.Background()

		ghClient := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)

		return rancher.UpdateDashboardReferences(ctx, rootConfig.Dashboard, ghClient, &dashboardRelease, rootConfig.User)
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.AddCommand(updateChartsCmd)
	updateCmd.AddCommand(updateK3sCmd)
	updateK3sCmd.AddCommand(updateK3sReferencesCmd)
	updateCmd.AddCommand(updateRancherCmd)
	updateRancherCmd.AddCommand(updateRancherDashboardCmd)
}

func validateChartConfig() error {
	if rootConfig.Charts.Workspace == "" || rootConfig.Charts.ChartsForkURL == "" {
		return errors.New("verify your config file, chart configuration not implemented correctly, you must insert workspace path and your forked repo url")
	}
	return nil
}
