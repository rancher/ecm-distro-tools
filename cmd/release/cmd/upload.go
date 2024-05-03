package cmd

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rancher/ecm-distro-tools/release/rancher"
	"github.com/rancher/ecm-distro-tools/repository"

	"github.com/spf13/cobra"
)

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload files",
}

var uploadRancherCmd = &cobra.Command{
	Use:   "rancher",
	Short: "Upload rancher files",
}

var uploadRancherArtifactsCmd = &cobra.Command{
	Use:   "artifacts [version]",
	Short: "Upload rancher artifacts from a GitHub Release to an s3 bucket",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("expected at least one argument: [version]")
		}
		version := args[0]
		rancherRelease, found := rootConfig.Rancher.Versions[version]
		if !found {
			return errors.New("verify your config file, version not found: " + version)
		}
		ctx := context.Background()
		ghClient := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return errors.New("failed to load aws default config: " + err.Error())
		}
		s3Client := s3.NewFromConfig(cfg)
		s3Uploader := manager.NewUploader(s3Client)
		return rancher.UploadRancherArtifacts(ctx, ghClient, s3Uploader, &rancherRelease, version)
	},
}

func init() {
	rootCmd.AddCommand(uploadCmd)
	uploadCmd.AddCommand(uploadRancherCmd)
	uploadRancherCmd.AddCommand(uploadRancherArtifactsCmd)
}
