package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rancher/ecm-distro-tools/release"
	"github.com/rancher/ecm-distro-tools/release/k3s"
	"github.com/rancher/ecm-distro-tools/release/rancher"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/spf13/cobra"
)

var (
	k3sPrevMilestone string
	k3sMilestone     string

	concurrencyLimit                    int
	imagesListURL                       string
	ignoreImages                        []string
	checkImages                         []string
	registry                            string
	rancherMissingImagesJSONOutput      bool
	rke2PrevMilestone                   string
	rke2Milestone                       string
	rancherArtifactsIndexWriteToPath    string
	rancherArtifactsIndexIgnoreVersions []string
	rancherImagesDigestsOutputFile      string
	rancherImagesDigestsRegistry        string
	rancherImagesDigestsImagesURL       string
	rancherSyncImages                   []string
	rancherSourceRegistry               string
	rancherTargetRegistry               string
	rancherSyncConfigOutputPath         string
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Various utilities to generate release artifacts",
}

var k3sGenerateSubCmd = &cobra.Command{
	Use:   "k3s",
	Short: "Generate k3s related artifacts",
}

var k3sGenerateReleaseNotesSubCmd = &cobra.Command{
	Use:   "release-notes",
	Short: "Generate k3s release notes",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)

		notes, err := release.GenReleaseNotes(ctx, "k3s-io", "k3s", k3sMilestone, k3sPrevMilestone, client)
		if err != nil {
			return err
		}

		fmt.Print(notes.String())

		return nil
	},
}

var k3sGenerateTagsSubCmd = &cobra.Command{
	Use:   "tags [version]",
	Short: "Generate k8s tags for a given version",
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
		return k3s.GenerateTags(ctx, ghClient, &k3sRelease, rootConfig.User, rootConfig.Auth.SSHKeyPath)
	},
}

var rke2GenerateSubCmd = &cobra.Command{
	Use:   "rke2",
	Short: "Generate rke2 related artifacts",
}

var rke2GenerateReleaseNotesSubCmd = &cobra.Command{
	Use:   "release-notes",
	Short: "Generate rke2 release notes",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)

		notes, err := release.GenReleaseNotes(ctx, "rancher", "rke2", rke2Milestone, rke2PrevMilestone, client)
		if err != nil {
			return err
		}

		fmt.Print(notes.String())

		return nil
	},
}

var rancherGenerateSubCmd = &cobra.Command{
	Use:   "rancher",
	Short: "Generate rancher related artifacts",
}

var rancherGenerateArtifactsIndexSubCmd = &cobra.Command{
	Use:   "artifacts-index",
	Short: "Generate artifacts index page",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		cfg, err := config.LoadDefaultConfig(ctx, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(rootConfig.Auth.AWSAccessKeyID, rootConfig.Auth.AWSSecretAccessKey, rootConfig.Auth.AWSSessionToken)), config.WithDefaultRegion(rootConfig.Auth.AWSDefaultRegion))
		if err != nil {
			return err
		}
		client := s3.NewFromConfig(cfg)
		return rancher.GeneratePrimeArtifactsIndex(ctx, rancherArtifactsIndexWriteToPath, rancherArtifactsIndexIgnoreVersions, client)
	},
}

var rancherGenerateMissingImagesListSubCmd = &cobra.Command{
	Use:   "missing-images-list",
	Short: "Generate a missing images list",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(checkImages) == 0 && imagesListURL == "" {
			return errors.New("either --images-list-url or --check-images must be provided")
		}
		missingImages, err := rancher.GenerateMissingImagesList(imagesListURL, registry, concurrencyLimit, checkImages, ignoreImages, verbose)
		if err != nil {
			return err
		}
		// if there are missing images, return it as an error so CI also fails
		if len(missingImages) != 0 && !rancherMissingImagesJSONOutput {
			return errors.New("found missing images: " + strings.Join(missingImages, ","))
		}
		if rancherMissingImagesJSONOutput {
			b, err := json.MarshalIndent(missingImages, "", " ")
			if err != nil {
				return err
			}
			fmt.Println(string(b))
		}
		return nil
	},
}

var rancherGenerateDockerImagesDigestsSubCmd = &cobra.Command{
	Use:   "docker-images-digests",
	Short: "Generate a file with images digests from an images list",
	RunE: func(cmd *cobra.Command, args []string) error {
		return rancher.GenerateDockerImageDigests(rancherImagesDigestsOutputFile, rancherImagesDigestsImagesURL, rancherImagesDigestsRegistry, verbose)
	},
}

var rancherGenerateImagesSyncConfigSubCmd = &cobra.Command{
	Use:   "images-sync-config",
	Short: "Generate a regsync config file for images sync",
	RunE: func(cmd *cobra.Command, args []string) error {
		return rancher.GenerateImagesSyncConfig(rancherSyncImages, rancherSourceRegistry, rancherTargetRegistry, rancherSyncConfigOutputPath)
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)

	k3sGenerateSubCmd.AddCommand(k3sGenerateReleaseNotesSubCmd)
	k3sGenerateSubCmd.AddCommand(k3sGenerateTagsSubCmd)
	rke2GenerateSubCmd.AddCommand(rke2GenerateReleaseNotesSubCmd)
	rancherGenerateSubCmd.AddCommand(rancherGenerateArtifactsIndexSubCmd)
	rancherGenerateSubCmd.AddCommand(rancherGenerateMissingImagesListSubCmd)
	rancherGenerateSubCmd.AddCommand(rancherGenerateDockerImagesDigestsSubCmd)
	rancherGenerateSubCmd.AddCommand(rancherGenerateImagesSyncConfigSubCmd)

	generateCmd.AddCommand(k3sGenerateSubCmd)
	generateCmd.AddCommand(rke2GenerateSubCmd)
	generateCmd.AddCommand(rancherGenerateSubCmd)

	// k3s release notes
	k3sGenerateReleaseNotesSubCmd.Flags().StringVarP(&k3sPrevMilestone, "prev-milestone", "p", "", "Previous Milestone")
	k3sGenerateReleaseNotesSubCmd.Flags().StringVarP(&k3sMilestone, "milestone", "m", "", "Milestone")
	if err := k3sGenerateReleaseNotesSubCmd.MarkFlagRequired("prev-milestone"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if err := k3sGenerateReleaseNotesSubCmd.MarkFlagRequired("milestone"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// rke2 release notes
	rke2GenerateReleaseNotesSubCmd.Flags().StringVarP(&rke2PrevMilestone, "prev-milestone", "p", "", "Previous Milestone")
	rke2GenerateReleaseNotesSubCmd.Flags().StringVarP(&rke2Milestone, "milestone", "m", "", "Milestone")
	if err := rke2GenerateReleaseNotesSubCmd.MarkFlagRequired("prev-milestone"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if err := rke2GenerateReleaseNotesSubCmd.MarkFlagRequired("milestone"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// rancher artifacts-index
	rancherGenerateArtifactsIndexSubCmd.Flags().StringSliceVarP(&rancherArtifactsIndexIgnoreVersions, "ignore-versions", "i", []string{}, "Versions to ignore on the index")
	rancherGenerateArtifactsIndexSubCmd.Flags().StringVarP(&rancherArtifactsIndexWriteToPath, "write-path", "w", "", "Write To Path")
	if err := rancherGenerateArtifactsIndexSubCmd.MarkFlagRequired("write-path"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// rancher generate missing-images-list
	rancherGenerateMissingImagesListSubCmd.Flags().IntVarP(&concurrencyLimit, "concurrency-limit", "l", 3, "Concurrency Limit")
	rancherGenerateMissingImagesListSubCmd.Flags().BoolVarP(&rancherMissingImagesJSONOutput, "json", "j", false, "JSON Output")
	rancherGenerateMissingImagesListSubCmd.Flags().StringVarP(&imagesListURL, "images-list-url", "i", "", "URL of the artifact containing all images for a given version 'rancher-images.txt' (required)")
	rancherGenerateMissingImagesListSubCmd.Flags().StringSliceVarP(&ignoreImages, "ignore-images", "g", make([]string, 0), "Images to ignore when checking for missing images without the version. e.g: rancher/rancher")
	rancherGenerateMissingImagesListSubCmd.Flags().StringSliceVarP(&checkImages, "check-images", "k", make([]string, 0), "Images to check for when checking for missing images with the version. e.g: rancher/rancher-agent:v2.9.0")
	rancherGenerateMissingImagesListSubCmd.Flags().StringVarP(&registry, "registry", "r", "registry.rancher.com", "Registry where the images should be located at")

	// rancher generate docker-images-digests
	rancherGenerateDockerImagesDigestsSubCmd.Flags().StringVarP(&rancherImagesDigestsOutputFile, "output-file", "o", "", "Output file with images digests")
	if err := rancherGenerateDockerImagesDigestsSubCmd.MarkFlagRequired("output-file"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	rancherGenerateDockerImagesDigestsSubCmd.Flags().StringVarP(&rancherImagesDigestsImagesURL, "images-url", "i", "", "Images list artifact URL")
	if err := rancherGenerateDockerImagesDigestsSubCmd.MarkFlagRequired("images-url"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	rancherGenerateDockerImagesDigestsSubCmd.Flags().StringVarP(&rancherImagesDigestsRegistry, "registry", "r", "", "Docker Registry e.g: docker.io")
	if err := rancherGenerateDockerImagesDigestsSubCmd.MarkFlagRequired("registry"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	// rancher generate images-sync-config
	rancherGenerateImagesSyncConfigSubCmd.Flags().StringSliceVarP(&rancherSyncImages, "images", "k", make([]string, 0), "List of images to sync to a registry")
	rancherGenerateImagesSyncConfigSubCmd.Flags().StringVarP(&rancherSourceRegistry, "source-registry", "s", "", "Source registry, where the images are located")
	rancherGenerateImagesSyncConfigSubCmd.Flags().StringVarP(&rancherTargetRegistry, "target-registry", "t", "", "Target registry, where the images should be synced to")
	rancherGenerateImagesSyncConfigSubCmd.Flags().StringVarP(&rancherSyncConfigOutputPath, "output", "o", "./config.yaml", "Output path of the generated config file")
	if err := rancherGenerateImagesSyncConfigSubCmd.MarkFlagRequired("images"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if err := rancherGenerateImagesSyncConfigSubCmd.MarkFlagRequired("source-registry"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if err := rancherGenerateImagesSyncConfigSubCmd.MarkFlagRequired("target-registry"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
