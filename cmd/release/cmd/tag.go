package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/rancher/ecm-distro-tools/release/k3s"
	"github.com/rancher/ecm-distro-tools/release/rancher"
	"github.com/rancher/ecm-distro-tools/release/rke2"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/spf13/cobra"
)

type tagRKE2CmdFlags struct {
	AlpineVersion  *string
	ReleaseVersion *string
	RCVersion      *string
	RPMVersion     *int
}

type tagRancherCmdFlags struct {
	Tag       *string
	Branch    *string
	RepoOwner *string
	DryRun    *bool
}

var (
	tagRKE2Flags    tagRKE2CmdFlags
	tagRancherFlags tagRancherCmdFlags
)

// tagCmd represents the tag command.
var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Tag releases",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			rootCmd.Help()
			os.Exit(0)
		}
	},
}

var k3sTagSubCmd = &cobra.Command{
	Use:   "k3s [ga,rc] [version]",
	Short: "Tag k3s releases",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return errors.New("expected at least two arguments: [ga,rc] [version]")
		}
		rc, err := releaseTypeRC(args[0])
		if err != nil {
			return err
		}
		tag := args[1]
		k3sRelease, found := rootConfig.K3s.Versions[tag]
		if !found {
			return errors.New("verify your config file, version not found: " + tag)
		}
		ctx := context.Background()
		ghClient := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)
		return k3s.CreateRelease(ctx, ghClient, &k3sRelease, tag, rc)
	},
}

var rke2TagSubCmd = &cobra.Command{
	Use:   "rke2",
	Short: "Tag rke2 releases",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(*tagRKE2Flags.AlpineVersion) != 2 {
			return errors.New("invalid release version")
		}

		ctx := context.Background()
		client := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)

		switch args[0] {
		case "image-build-base":
			if err := rke2.ImageBuildBaseRelease(ctx, client, *tagRKE2Flags.AlpineVersion, *dryRun); err != nil {
				return err
			}
		case "image-build-kubernetes":
			now := time.Now().UTC().Format("20060102")
			suffix := "-rke2" + *tagRKE2Flags.ReleaseVersion + "-build" + now

			if *dryRun {
				fmt.Println("dry-run:")
				for _, version := range rootConfig.RKE2.Versions {
					fmt.Println("\t" + version + suffix)
				}
			} else {
				for _, version := range rootConfig.RKE2.Versions {
					cro := repository.CreateReleaseOpts{
						Owner:      "rancher",
						Repo:       "image-build-kubernetes",
						Branch:     "master",
						Name:       version + suffix,
						Prerelease: false,
					}
					if _, err := repository.CreateRelease(ctx, client, &cro); err != nil {
						return err
					}

					fmt.Println("tag " + version + suffix + " created successfully")
				}
			}
		case "rke2":
			//
		case "rpm":
			if len(args) == 1 {
				return errors.New("invalid rpm tag. expected {testinglatest|stable}")
			}

			rpmTag := fmt.Sprintf("+rke2%s.%s.%d", *tagRKE2Flags.ReleaseVersion, args[1], *tagRKE2Flags.RPMVersion)
			if *tagRKE2Flags.RCVersion != "" {
				rpmTag = fmt.Sprintf("+rke2%s-rc%s.%s.%d", *tagRKE2Flags.ReleaseVersion, *tagRKE2Flags.RCVersion, args[1], *tagRKE2Flags.RPMVersion)
			}

			if *dryRun {
				fmt.Print("(dry-run)\n\nTagging github.com/rancher/rke2-packaging:\n\n")
				for _, version := range rootConfig.RKE2.Versions {
					fmt.Println("\t" + version + rpmTag)
				}
			} else {
				for _, version := range rootConfig.RKE2.Versions {
					cro := repository.CreateReleaseOpts{
						Owner:      "rancher",
						Repo:       "rke2-packaging",
						Branch:     "master",
						Name:       version + rpmTag,
						Prerelease: false,
					}
					if _, err := repository.CreateRelease(ctx, client, &cro); err != nil {
						return err
					}
				}
			}
		default:
			return errors.New("unrecognized resource")
		}

		return nil
	},
}

var rancherTagSubCmd = &cobra.Command{
	Use:   "rancher",
	Short: "Tag Rancher releases",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		ghClient := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)
		return rancher.TagRelease(ctx, ghClient, *tagRancherFlags.Tag, *tagRancherFlags.Branch, *tagRancherFlags.RepoOwner, *tagRancherFlags.DryRun)
	},
}

func init() {
	rootCmd.AddCommand(tagCmd)

	tagCmd.AddCommand(k3sTagSubCmd)
	tagCmd.AddCommand(rke2TagSubCmd)
	tagCmd.AddCommand(rancherTagSubCmd)

	dryRun = tagCmd.PersistentFlags().BoolP("dry-run", "r", false, "Dry run")

	// rke2
	tagRKE2Flags.AlpineVersion = rke2TagSubCmd.Flags().StringP("alpine-version", "a", "", "Alpine version")
	tagRKE2Flags.ReleaseVersion = rke2TagSubCmd.Flags().StringP("release-version", "r", "r1", "Release version")
	tagRKE2Flags.RCVersion = rke2TagSubCmd.Flags().String("rc", "", "RC version")
	tagRKE2Flags.RPMVersion = rke2TagSubCmd.Flags().Int("rpm-version", 0, "RPM version")

	// rancher
	tagRancherFlags.Tag = rancherTagSubCmd.Flags().StringP("tag", "t", "", "tag to be created. e.g: v2.8.1-rc4")
	tagRancherFlags.Branch = rancherTagSubCmd.Flags().StringP("branch", "b", "", "branch to be used as the base to create the tag. e.g: release/v2.8")
	tagRancherFlags.RepoOwner = rancherTagSubCmd.Flags().StringP("repo-owner", "o", "rancher", "repository owner to create the tag in, optional")
	tagRancherFlags.DryRun = dryRun
	if err := rancherTagSubCmd.MarkFlagRequired("tag"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if err := rancherTagSubCmd.MarkFlagRequired("branch"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func releaseTypeRC(releaseType string) (bool, error) {
	if releaseType == "rc" {
		return true, nil
	}
	if releaseType == "ga" {
		return false, nil
	}
	return false, errors.New("release type must be either 'ga' or 'rc', instead got: " + releaseType)
}
