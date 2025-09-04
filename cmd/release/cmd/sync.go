package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/rancher/ecm-distro-tools/release/imagebuild"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/spf13/cobra"
)

var (
	imageBuildOwner   string
	imageBuildRepo    string
	upstreamOwner     string
	upstreamRepo      string
	upstreamTagPrefix string
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync images and other utilities",
}

var syncImageBuildCmd = &cobra.Command{
	Use:       "image-build",
	Short:     "Sync image-build repo with upstream",
	ValidArgs: []string{},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		ghToken := os.Getenv("GITHUB_TOKEN")
		if ghToken == "" {
			return errors.New("GITHUB_TOKEN env is empty")
		}
		ghClient := repository.NewGithub(ctx, ghToken)

		return imagebuild.Sync(ctx, ghClient, imageBuildOwner, imageBuildRepo, upstreamOwner, upstreamRepo, upstreamTagPrefix, dryRun)
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)

	syncCmd.AddCommand(syncImageBuildCmd)

	syncImageBuildCmd.Flags().StringVar(&upstreamTagPrefix, "tag-prefix", "", "Upstream tag Prefix")
	syncImageBuildCmd.Flags().StringVar(&upstreamRepo, "upstream-repo", "", "Upstream repository name")
	if err := syncImageBuildCmd.MarkFlagRequired("upstream-repo"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	syncImageBuildCmd.Flags().StringVar(&upstreamOwner, "upstream-owner", "", "Upstream repository owner")
	if err := syncImageBuildCmd.MarkFlagRequired("upstream-owner"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	syncImageBuildCmd.Flags().StringVar(&imageBuildRepo, "image-build-repo", "", "Image build repository name")
	if err := syncImageBuildCmd.MarkFlagRequired("image-build-repo"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	syncImageBuildCmd.Flags().StringVar(&imageBuildOwner, "image-build-owner", "rancher", "Image build repository owner")
	if err := syncImageBuildCmd.MarkFlagRequired("image-build-owner"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
