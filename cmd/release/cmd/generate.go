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
	k3sPrevMilestone string
	k3sMilestone     string

	concurrencyLimit                    int
	rancherMissingImagesJSONOutput      bool
	rke2PrevMilestone                   string
	rke2Milestone                       string
	rancherArtifactsIndexWriteToPath    string
	rancherArtifactsIndexIgnoreVersions []string
	rancherImagesDigestsOutputFile      string
	rancherImagesDigestsRegistry        string
	rancherImagesDigestsImagesURL       string
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
		return rancher.GeneratePrimeArtifactsIndex(rancherArtifactsIndexWriteToPath, rancherArtifactsIndexIgnoreVersions)
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
		missingImages, err := rancher.GenerateMissingImagesList(version, concurrencyLimit, rancherRelease.CheckImages)
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
		return rancher.GenerateDockerImageDigests(rancherImagesDigestsOutputFile, rancherImagesDigestsImagesURL, rancherImagesDigestsRegistry)
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
	rancherGenerateMissingImagesListSubCmd.Flags().IntVarP(&concurrencyLimit, "concurrency-limit", "c", 3, "Concurrency Limit")
	rancherGenerateMissingImagesListSubCmd.Flags().BoolVarP(&rancherMissingImagesJSONOutput, "json", "j", false, "JSON Output")

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
}
