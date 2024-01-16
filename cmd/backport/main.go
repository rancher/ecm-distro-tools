package main

import (
	"context"
	"errors"
	"os"

	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var version = "development"

type BackportCmdOpts struct {
	Repo     string
	Issue    uint
	Commits  []string
	Branches []string
	User     string
	Owner    string
	DryRun   bool
}

var backportCmdOpts BackportCmdOpts

func main() {
	cmd := &cobra.Command{
		Use:     "backport",
		Short:   "Generate backport issues and cherry pick commits to branches",
		Long:    "The backport utility needs to be executed inside the repository you want to perform the actions",
		RunE:    backport,
		Version: version,
	}

	cmd.Flags().StringVarP(&backportCmdOpts.Repo, "repo", "r", "", "name of the repository to perform the backport (k3s, rke2)")
	cmd.Flags().UintVarP(&backportCmdOpts.Issue, "issue", "i", 0, "ID of the original issue")
	cmd.Flags().StringSliceVarP(&backportCmdOpts.Commits, "commits", "c", []string{}, "commits to be backported, if none is provided, only the issues will be created, when passing this flag, it assumes you're running from the repository this operation is related to (comma separated)")
	cmd.Flags().StringSliceVarP(&backportCmdOpts.Branches, "branches", "b", []string{}, "branches the issue is being backported to, one or more (comma separated)")
	cmd.Flags().StringVarP(&backportCmdOpts.User, "user", "u", "", "user to assign new issues to (default: user assigned to the original issue)")
	cmd.Flags().StringVarP(&backportCmdOpts.Owner, "owner", "o", "", "owner of the repository, e.g: k3s-io, rancher")
	cmd.Flags().BoolVarP(&backportCmdOpts.DryRun, "dry-run", "n", false, "skip creating issues and pushing changes to remote")

	if err := cmd.MarkFlagRequired("repo"); err != nil {
		logrus.Fatal(err)
	}
	if err := cmd.MarkFlagRequired("owner"); err != nil {
		logrus.Fatal(err)
	}
	if err := cmd.MarkFlagRequired("issue"); err != nil {
		logrus.Fatal(err)
	}
	if err := cmd.MarkFlagRequired("branches"); err != nil {
		logrus.Fatal(err)
	}

	if err := cmd.Execute(); err != nil {
		logrus.Fatal(err)
	}
}

func backport(cmd *cobra.Command, args []string) error {
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		return errors.New("env variable GITHUB_TOKEN is required")
	}
	ctx := context.Background()
	githubClient := repository.NewGithub(ctx, githubToken)

	pbo := &repository.PerformBackportOpts{
		Owner:    backportCmdOpts.Owner,
		Repo:     backportCmdOpts.Repo,
		Branches: backportCmdOpts.Branches,
		Commits:  backportCmdOpts.Commits,
		IssueID:  backportCmdOpts.Issue,
		User:     backportCmdOpts.User,
		DryRun:   backportCmdOpts.DryRun,
	}
	issues, err := repository.PerformBackport(ctx, githubClient, pbo)
	if err != nil {
		return err
	}

	for _, issue := range issues {
		logrus.Info("Backport issue created: " + issue.GetHTMLURL())
	}

	return nil
}
