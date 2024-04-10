package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/rancher/ecm-distro-tools/release"
	"github.com/rancher/ecm-distro-tools/release/k3s"
	"github.com/rancher/ecm-distro-tools/release/rancher"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/spf13/cobra"
)

var (
	k3sPrevMilestone *string
	k3sMilestone     *string

	rke2PrevMilestone                   *string
	rke2Milestone                       *string
	artifactsIndexWriteToPath           *string
	concurrencyLimit                    *int
	rancherMissingImagesJSONOutput      *bool
	rancherArtifactsIndexWriteToPath    *string
	rancherArtifactsIndexIgnoreVersions *[]string
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

		notes, err := release.GenReleaseNotes(ctx, "k3s-io", "k3s", *k3sMilestone, *k3sPrevMilestone, client)
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

		notes, err := release.GenReleaseNotes(ctx, "rancher", "rke2", *rke2Milestone, *rke2PrevMilestone, client)
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
		return rancher.GeneratePrimeArtifactsIndex(*rancherArtifactsIndexWriteToPath, *rancherArtifactsIndexIgnoreVersions)
	},
}

var rancherGenerateMissingImagesListSubCmd = &cobra.Command{
	Use:   "missing-images-list [version]",
	Short: "Generate a missing images list",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("expected at least one argument: [version]")
		}
		version := args[0]
		rancherRelease, found := rootConfig.Rancher.Versions[version]
		if !found {
			return errors.New("verify your config file, version not found: " + version)
		}
		missingImages, err := rancher.GenerateMissingImagesList(version, *concurrencyLimit, rancherRelease.CheckImages)
		if err != nil {
			return err
		}
		// if there are missing images, return it as an error so CI also fails
		if len(missingImages) != 0 && !*rancherMissingImagesJSONOutput {
			return errors.New("found missing images: " + strings.Join(missingImages, ","))
		}
		if *rancherMissingImagesJSONOutput {
			b, err := json.MarshalIndent(missingImages, "", " ")
			if err != nil {
				return err
			}
			fmt.Println(string(b))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)

	k3sGenerateSubCmd.AddCommand(k3sGenerateReleaseNotesSubCmd)
	k3sGenerateSubCmd.AddCommand(k3sGenerateTagsSubCmd)
	rke2GenerateSubCmd.AddCommand(rke2GenerateReleaseNotesSubCmd)
	rancherGenerateSubCmd.AddCommand(rancherGenerateArtifactsIndexSubCmd)
	rancherGenerateSubCmd.AddCommand(rancherGenerateMissingImagesListSubCmd)

	generateCmd.AddCommand(k3sGenerateSubCmd)
	generateCmd.AddCommand(rke2GenerateSubCmd)
	generateCmd.AddCommand(rancherGenerateSubCmd)

	// k3s release notes
	k3sPrevMilestone = k3sGenerateReleaseNotesSubCmd.Flags().StringP("prev-milestone", "p", "", "Previous Milestone")
	k3sMilestone = k3sGenerateReleaseNotesSubCmd.Flags().StringP("milestone", "m", "", "Milestone")
	if err := k3sGenerateReleaseNotesSubCmd.MarkFlagRequired("prev-milestone"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if err := k3sGenerateReleaseNotesSubCmd.MarkFlagRequired("milestone"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// rke2 release notes
	rke2PrevMilestone = rke2GenerateReleaseNotesSubCmd.Flags().StringP("prev-milestone", "p", "", "Previous Milestone")
	rke2Milestone = rke2GenerateReleaseNotesSubCmd.Flags().StringP("milestone", "m", "", "Milestone")
	if err := rke2GenerateReleaseNotesSubCmd.MarkFlagRequired("prev-milestone"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if err := rke2GenerateReleaseNotesSubCmd.MarkFlagRequired("milestone"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// rancher artifacts-index
	rancherArtifactsIndexIgnoreVersions = rancherGenerateArtifactsIndexSubCmd.Flags().StringSliceP("ignore-versions", "i", []string{}, "Versions to ignore on the index")
	rancherArtifactsIndexWriteToPath = rancherGenerateArtifactsIndexSubCmd.Flags().StringP("write-path", "w", "", "Write To Path")
	if err := rancherGenerateArtifactsIndexSubCmd.MarkFlagRequired("write-path"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// rancher generate-missing-images-list
	concurrencyLimit = rancherGenerateMissingImagesListSubCmd.Flags().IntP("concurrency-limit", "c", 3, "Concurrency Limit")
	rancherMissingImagesJSONOutput = rancherGenerateMissingImagesListSubCmd.Flags().BoolP("json", "j", false, "JSON Output")
}
