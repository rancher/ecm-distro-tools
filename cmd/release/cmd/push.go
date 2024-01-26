package cmd

import (
	"context"
	"errors"

	"github.com/rancher/ecm-distro-tools/release/k3s"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "",
}

var pushK3sCmd = &cobra.Command{
	Use:   "k3s [version]",
	Short: "",
}

var pushK3sTagsCmd = &cobra.Command{
	Use:   "tags [version]",
	Short: "push k8s tags to remote",
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
		return k3s.PushTags(ghClient, &k3sRelease, rootConfig.User, rootConfig.Auth.SSHKeyPath)
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)
	pushCmd.AddCommand(pushK3sCmd)
	pushK3sCmd.AddCommand(pushK3sTagsCmd)
}
