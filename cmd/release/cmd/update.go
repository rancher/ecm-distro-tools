package cmd

import (
	"context"
	"errors"

	"github.com/rancher/ecm-distro-tools/release/k3s"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "",
}

var updateK3sCmd = &cobra.Command{
	Use:   "k3s",
	Short: "",
}

var updateK3sReferencesCmd = &cobra.Command{
	Use:   "references [version]",
	Short: "update k8s and go references in a k3s repo and create a PR",
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

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.AddCommand(updateK3sCmd)
	updateK3sCmd.AddCommand(updateK3sReferencesCmd)
}
