package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/Masterminds/semver/v3"
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
			return NewVersionNotFoundError(tag, "k3s")
		}

		ctx := context.Background()
		ghClient := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)

		opts := repository.CreateRefOpts{
			Tag:    tag,
			Repo:   "k3s",
			Owner:  k3sRelease.K3sRepoOwner,
			Branch: k3sRelease.ReleaseBranch,
		}
		return k3s.CreateRef(ctx, ghClient, &k3sRelease, &opts, rc)
	},
}

var rke2TagSubCmd = &cobra.Command{
	Use:   "rke2 [ga,rc] [version]",
	Short: "Tag rke2 releases",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return errors.New("expected at least two arguments: [ga,rc] [version]")
		}

		rc, err := releaseTypePreRelease(args[0])
		if err != nil {
			return err
		}

		tag := args[1]
		rke2Release, found := rootConfig.RKE2.Versions[tag]
		if !found {
			return NewVersionNotFoundError(tag, "rke2")
		}

		ctx := context.Background()
		ghClient := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)

		opts := repository.CreateRefOpts{
			Tag:    tag,
			Repo:   rke2Release.RKE2RepoName,
			Owner:  rke2Release.RKE2RepoOwner,
			Branch: rke2Release.ReleaseBranch,
		}
		return rke2.CreateRef(ctx, ghClient, &rke2Release, &opts, rc)
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
			return NewVersionNotFoundError(tag, "rancher")
		}

		repo := config.ValueOrDefault(rootConfig.RancherRepositoryName, config.RancherRepositoryName)
		owner := config.ValueOrDefault(rootConfig.RancherGithubOrganization, config.RancherGithubOrganization)

		releaseBranch, err := rancher.ReleaseBranchFromTag(tag)
		if err != nil {
			return errors.New("failed to generate release branch from tag: " + err.Error())
		}

		releaseBranch = config.ValueOrDefault(rancherRelease.ReleaseBranch, releaseBranch)

		ctx := context.Background()
		ghClient := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)
		createdTag, tagCommit, err := rancher.CreateTag(ctx, ghClient, owner, repo, tag, "", releaseBranch, releaseType, preRelease, dryRun)
		if err != nil {
			return err
		}
		fmt.Println("created tag: " + createdTag + ": " + tagCommit)
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
			return NewVersionNotFoundError(tag, "k3s")
		}

		ctx := context.Background()

		ghClient := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)
		opts := &repository.CreateReleaseOpts{
			Tag:    tag,
			Repo:   "system-agent-installer-k3s",
			Owner:  k3sRelease.SystemAgentInstallerRepoOwner,
			Branch: "main",
		}

		return k3s.CreateRelease(ctx, ghClient, &k3sRelease, opts, releaseNotesAlert, rc)
	},
}

var rancherPrimeTagSubCmd = &cobra.Command{
	Use:   "rancher-prime [ga, rc, alpha] [version]",
	Short: "Tag Rancher Prime releases",
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
			return NewVersionNotFoundError(tag, "rancher")
		}

		repo := config.ValueOrDefault(rootConfig.RancherPrimeRepositoryName, config.RancherPrimeRepositoryName)
		owner := config.ValueOrDefault(rootConfig.RancherGithubOrganization, config.RancherGithubOrganization)

		releaseBranch, err := rancher.ReleaseBranchFromTag(tag)
		if err != nil {
			return errors.New("failed to generate release branch from tag: " + err.Error())
		}

		releaseBranch = config.ValueOrDefault(rancherRelease.ReleaseBranch, releaseBranch)

		ctx := context.Background()
		ghClient := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)

		createdTag, tagCommit, err := rancher.CreateTag(ctx, ghClient, owner, repo, tag, "", releaseBranch, releaseType, preRelease, dryRun)
		if err != nil {
			return err
		}
		fmt.Println("created tag: " + createdTag + ": " + tagCommit)
		return nil
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
			return NewVersionNotFoundError(version, "dashboard")
		}
		return nil
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return copyReleaseTypes(), cobra.ShellCompDirectiveNoFileComp
		}

		if len(args) == 1 {
			return copyDashboardVersions(), cobra.ShellCompDirectiveNoFileComp
		}

		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		releaseType := args[0]

		preRelease, err := releaseTypePreRelease(releaseType)
		if err != nil {
			return err
		}

		tag := args[1]
		ctx := context.Background()
		ghClient := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)

		dashboardRelease, found := rootConfig.Dashboard.Versions[tag]
		if !found {
			return NewVersionNotFoundError(tag, "dashboard")
		}

		uiRepo := config.ValueOrDefault(rootConfig.UIRepositoryName, config.UIRepositoryName)
		dashboardRepo := config.ValueOrDefault(rootConfig.DashboardRepositoryName, config.DashboardRepositoryName)
		repoOwner := config.ValueOrDefault(rootConfig.RancherGithubOrganization, config.RancherGithubOrganization)

		releaseBranch, err := dashboard.ReleaseBranchFromTag(tag)
		if err != nil {
			return errors.New("failed to generate release branch from tag: " + err.Error())
		}

		releaseBranch = config.ValueOrDefault(dashboardRelease.ReleaseBranch, releaseBranch)

		previousTag := dashboardRelease.PreviousTag

		if previousTag == "" {
			previousTag, err = previousPatch(tag)
			if err != nil {
				return err
			}
		}

		uiOpts := &repository.CreateReleaseOpts{
			Tag:    tag,
			Repo:   uiRepo,
			Owner:  repoOwner,
			Branch: releaseBranch,
			Draft:  false,
		}

		if err := ui.CreateRelease(ctx, ghClient, uiOpts, preRelease, dryRun, releaseType, previousTag, releaseNotesAlert); err != nil {
			return err
		}

		dashboardOpts := &repository.CreateReleaseOpts{
			Tag:    tag,
			Repo:   dashboardRepo,
			Owner:  repoOwner,
			Branch: releaseBranch,
			Draft:  false,
		}

		return dashboard.CreateRelease(ctx, ghClient, dashboardOpts, preRelease, dryRun, releaseType, previousTag, releaseNotesAlert)
	},
}

var cliTagSubCmd = &cobra.Command{
	Use:   "cli [ga,rc] [version]",
	Short: "Tag CLI releases",
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return copyReleaseTypes(), cobra.ShellCompDirectiveNoFileComp
		}

		if len(args) == 1 {
			return copyCLIVersions(), cobra.ShellCompDirectiveNoFileComp
		}

		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return errors.New("expected at least two arguments: [ga,rc] [version]")
		}

		version := args[1]
		if _, found := rootConfig.CLI.Versions[version]; !found {
			return NewVersionNotFoundError(version, "cli")
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
			return NewVersionNotFoundError(tag, "cli")
		}

		repo := config.ValueOrDefault(rootConfig.CLIRepositoryName, config.CLIRepositoryName)
		owner := config.ValueOrDefault(rootConfig.RancherGithubOrganization, config.RancherGithubOrganization)

		releaseBranch, err := cli.ReleaseBranchFromTag(tag)
		if err != nil {
			return errors.New("failed to generate release branch from tag: " + err.Error())
		}

		releaseBranch = config.ValueOrDefault(cliRelease.ReleaseBranch, releaseBranch)

		previousTag := cliRelease.PreviousTag

		if previousTag == "" {
			previousTag, err = previousPatch(tag)
			if err != nil {
				return err
			}
		}

		cliOpts := &repository.CreateReleaseOpts{
			Tag:    tag,
			Repo:   repo,
			Owner:  owner,
			Branch: releaseBranch,
			Draft:  false,
		}

		return cli.CreateRelease(ctx, ghClient, cliOpts, rc, releaseType, previousTag, releaseNotesAlert, dryRun)
	},
}

func previousPatch(tag string) (string, error) {
	version, err := semver.NewVersion(tag)
	if err != nil {
		return "", err
	}
	patch := version.Patch()
	if patch == 0 {
		return "", errors.New("can't find previous tag for a new minor: " + tag)
	}

	patch -= 1

	previousPatch := fmt.Sprintf("v%d.%d.%d", version.Major(), version.Minor(), patch)
	return previousPatch, nil
}

func copyDashboardVersions() []string {
	versions := make([]string, len(rootConfig.Dashboard.Versions))

	var i int

	for version := range rootConfig.Dashboard.Versions {
		versions[i] = version
		i++
	}

	return versions
}

func init() {
	rootCmd.AddCommand(tagCmd)

	tagCmd.AddCommand(k3sTagSubCmd)
	tagCmd.AddCommand(rke2TagSubCmd)
	tagCmd.AddCommand(rancherTagSubCmd)
	tagCmd.AddCommand(rancherPrimeTagSubCmd)
	tagCmd.AddCommand(systemAgentInstallerK3sTagSubCmd)
	tagCmd.AddCommand(dashboardTagSubCmd)
	tagCmd.AddCommand(cliTagSubCmd)

	// rke2
	tagRKE2Flags.ReleaseVersion = rke2TagSubCmd.Flags().StringP("release-version", "r", "r1", "Release version")
	tagRKE2Flags.RCVersion = rke2TagSubCmd.Flags().String("rc", "", "RC version")
	tagRKE2Flags.RPMVersion = rke2TagSubCmd.Flags().Int("rpm-version", 0, "RPM version")
}

func releaseTypePreRelease(releaseType string) (bool, error) {
	rt, ok := rancher.ReleaseTypes[releaseType]
	if !ok {
		return false, errors.New("invalid release type: " + releaseType)
	}
	if rt == rancher.ReleaseTypePreRelease {
		return true, nil
	}
	return false, nil
}
