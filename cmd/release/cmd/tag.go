package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rancher/ecm-distro-tools/cmd/release/config"
	"github.com/rancher/ecm-distro-tools/release/cli"
	"github.com/rancher/ecm-distro-tools/release/dashboard"
	"github.com/rancher/ecm-distro-tools/release/k3s"
	"github.com/rancher/ecm-distro-tools/release/rancher"
	"github.com/rancher/ecm-distro-tools/release/rke2"
	"github.com/rancher/ecm-distro-tools/release/ui"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/spf13/cobra"
)

type tagRKE2CmdFlags struct {
	AlpineVersion  *string
	ReleaseVersion *string
	RCVersion      *string
	RPMVersion     *int
}

var tagRKE2Flags tagRKE2CmdFlags

// tagCmd represents the tag command.
var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Tag releases",
}

var k3sTagSubCmd = &cobra.Command{
	Use:   "k3s [ga,rc] [version]",
	Short: "Tag k3s releases",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return errors.New("expected at least two arguments: [ga,rc] [version]")
		}

		rc, err := releaseTypePreRelease(args[0])
		if err != nil {
			return err
		}

		tag := args[1]
		k3sRelease, found := rootConfig.K3s.Versions[tag]
		if !found {
			return NewVersionNotFoundError(tag)
		}

		ctx := context.Background()
		ghClient := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)

		opts := repository.CreateReleaseOpts{
			Tag:    tag,
			Repo:   "k3s",
			Owner:  k3sRelease.K3sRepoOwner,
			Branch: k3sRelease.ReleaseBranch,
		}
		return k3s.CreateRelease(ctx, ghClient, &k3sRelease, &opts, rc)
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
			if err := rke2.ImageBuildBaseRelease(ctx, client, *tagRKE2Flags.AlpineVersion, dryRun); err != nil {
				return err
			}
		case "image-build-kubernetes":
			now := time.Now().UTC().Format("20060102")
			suffix := "-rke2" + *tagRKE2Flags.ReleaseVersion + "-build" + now

			if dryRun {
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

			if dryRun {
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
						Tag:        version + rpmTag,
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
	Use:   "rancher [ga, rc, alpha] [version]",
	Short: "Tag Rancher releases",
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return copyReleaseTypes(), cobra.ShellCompDirectiveNoFileComp
		}

		if len(args) == 1 {
			return copyRancherVersions(), cobra.ShellCompDirectiveNoFileComp
		}

		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return errors.New("expected at least two arguments: [ga,rc,alpha] [version]")
		}

		releaseType := args[0]
		preRelease, err := releaseTypePreRelease(releaseType)
		if err != nil {
			return err
		}

		tag := args[1]
		rancherRelease, found := rootConfig.Rancher.Versions[tag]
		if !found {
			return NewVersionNotFoundError(tag)
		}

		repo := config.ValueOrDefault(rootConfig.RancherRepositoryName, config.RancherRepositoryName)
		owner := config.ValueOrDefault(rootConfig.RancherGithubOrganization, config.RancherGithubOrganization)

		releaseBranch, err := rancher.ReleaseBranchFromTag(tag)
		if err != nil {
			return errors.New("failed to generate release branch from tag: " + err.Error())
		}

		branch := config.ValueOrDefault(rancherRelease.ReleaseBranch, releaseBranch)

		ctx := context.Background()
		ghClient := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)

		opts := &repository.CreateReleaseOpts{
			Tag:          tag,
			Repo:         repo,
			Owner:        owner,
			Branch:       branch,
			ReleaseNotes: "",
		}
		fmt.Printf("creating release options: %+v\n", opts)
		if dryRun {
			fmt.Println("dry run, skipping creating release")
			return nil
		}
		releaseURL, err := rancher.CreateRelease(ctx, ghClient, &rancherRelease, opts, preRelease, releaseType)
		if err != nil {
			return err
		}
		fmt.Println("created release: " + releaseURL)
		return nil
	},
}

var systemAgentInstallerK3sTagSubCmd = &cobra.Command{
	Use:   "system-agent-installer-k3s [ga,rc] [version]",
	Short: "Tag system-agent-installer-k3s releases",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return errors.New("expected at least two arguments: [ga,rc] [version]")
		}

		rc, err := releaseTypePreRelease(args[0])
		if err != nil {
			return err
		}

		tag := args[1]

		k3sRelease, found := rootConfig.K3s.Versions[tag]
		if !found {
			return NewVersionNotFoundError(tag)
		}

		ctx := context.Background()

		ghClient := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)
		opts := &repository.CreateReleaseOpts{
			Tag:    tag,
			Repo:   "system-agent-installer-k3s",
			Owner:  k3sRelease.SystemAgentInstallerRepoOwner,
			Branch: "main",
		}

		return k3s.CreateRelease(ctx, ghClient, &k3sRelease, opts, rc)
	},
}

var dashboardTagSubCmd = &cobra.Command{
	Use:   "dashboard [ga,rc,alpha] [version]",
	Short: "Tag dashboard releases",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return errors.New("expected at least two arguments: [ga,rc,alpha] [version]")
		}

		version := args[1]
		if _, found := rootConfig.Dashboard.Versions[version]; !found {
			return NewVersionNotFoundError(version)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		releaseType := args[0]

		rc, err := releaseTypePreRelease(releaseType)
		if err != nil {
			return err
		}

		tag := args[1]
		ctx := context.Background()
		ghClient := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)

		dashboardRelease, found := rootConfig.Dashboard.Versions[tag]
		if !found {
			return NewVersionNotFoundError(tag)
		}
		dashboardRelease.DryRun = dryRun

		uiOpts := &repository.CreateReleaseOpts{
			Tag:    tag,
			Repo:   rootConfig.Dashboard.UIRepoName,
			Owner:  rootConfig.Dashboard.UIRepoOwner,
			Branch: dashboardRelease.UIReleaseBranch,
		}

		if err := ui.CreateRelease(ctx, ghClient, &config.UIRelease{
			PreviousTag: dashboardRelease.PreviousTag,
			DryRun:      dryRun,
		}, uiOpts, rc, releaseType); err != nil {
			return err
		}

		dashboardOpts := &repository.CreateReleaseOpts{
			Tag:    tag,
			Repo:   rootConfig.Dashboard.RepoName,
			Owner:  rootConfig.Dashboard.RepoOwner,
			Branch: dashboardRelease.ReleaseBranch,
		}

		return dashboard.CreateRelease(ctx, ghClient, &dashboardRelease, dashboardOpts, rc, releaseType)
	},
}

var cliTagSubCmd = &cobra.Command{
	Use:   "cli [ga,rc] [version]",
	Short: "Tag dashboard releases",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return errors.New("expected at least two arguments: [ga,rc] [version]")
		}

		version := args[1]
		if _, found := rootConfig.CLI.Versions[version]; !found {
			return NewVersionNotFoundError(version)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		releaseType := args[0]

		rc, err := releaseTypePreRelease(releaseType)
		if err != nil {
			return err
		}

		tag := args[1]
		ctx := context.Background()
		ghClient := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)

		cliRelease, found := rootConfig.CLI.Versions[tag]
		if !found {
			return NewVersionNotFoundError(tag)
		}
		cliRelease.DryRun = dryRun

		cliOpts := &repository.CreateReleaseOpts{
			Tag:    tag,
			Repo:   rootConfig.CLI.RepoName,
			Owner:  rootConfig.CLI.RepoOwner,
			Branch: cliRelease.ReleaseBranch,
		}

		return cli.CreateRelease(ctx, ghClient, &cliRelease, cliOpts, rc, releaseType)
	},
}

func init() {
	rootCmd.AddCommand(tagCmd)

	tagCmd.AddCommand(k3sTagSubCmd)
	tagCmd.AddCommand(rke2TagSubCmd)
	tagCmd.AddCommand(rancherTagSubCmd)
	tagCmd.AddCommand(systemAgentInstallerK3sTagSubCmd)
	tagCmd.AddCommand(dashboardTagSubCmd)
	tagCmd.AddCommand(cliTagSubCmd)

	// rke2
	tagRKE2Flags.AlpineVersion = rke2TagSubCmd.Flags().StringP("alpine-version", "a", "", "Alpine version")
	tagRKE2Flags.ReleaseVersion = rke2TagSubCmd.Flags().StringP("release-version", "r", "r1", "Release version")
	tagRKE2Flags.RCVersion = rke2TagSubCmd.Flags().String("rc", "", "RC version")
	tagRKE2Flags.RPMVersion = rke2TagSubCmd.Flags().Int("rpm-version", 0, "RPM version")
}

func releaseTypePreRelease(releaseType string) (bool, error) {
	if releaseType == "rc" || releaseType == "alpha" {
		return true, nil
	}

	if releaseType == "ga" {
		return false, nil
	}

	return false, errors.New("release type must be either 'ga', 'alpha' or 'rc', instead got: " + releaseType)
}
