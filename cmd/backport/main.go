package main

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()
	app.Name = "backport"
	app.Usage = "generate backport issues and cherry pick commits to branches"
	app.UseShortOptionHandling = true
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     "repo",
			Aliases:  []string{"r"},
			Usage:    "name of the repository to perform the backport. e.g: k3s, rke2",
			Required: true,
		},
		&cli.UintFlag{
			Name:     "issue",
			Aliases:  []string{"i"},
			Usage:    "ID of the original issue",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "commits",
			Aliases:  []string{"c"},
			Usage:    "commits to be backported, if none is provided, only the issues will be created, when passing this flag, it assumes you're running from the repository this operation is related to (comma separated)",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "branches",
			Aliases:  []string{"b"},
			Usage:    "branches the issue is being backported to, one or more (comma separated)",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "user",
			Aliases:  []string{"u"},
			Usage:    "user to assign new issues to (default: user assigned to the original issue)",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "owner",
			Aliases:  []string{"o"},
			Usage:    "owner of the repository, defaults to either k3s-io or rancher depending on the repository",
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "dry-run",
			Aliases:  []string{"n"},
			Usage:    "skip creating issues and pushing changes to remote",
			Value:    false,
			Required: false,
		},
		&cli.StringFlag{
			Name:     "github-token",
			Aliases:  []string{"g"},
			EnvVars:  []string{"GITHUB_TOKEN"},
			Usage:    "github token",
			Required: true,
		},
	}

	app.Action = backport

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}

func backport(c *cli.Context) error {
	repo := c.String("repo")
	issue := c.Uint("issue")
	rawCommits := c.String("commits")
	rawBranches := c.String("branches")
	user := c.String("user")
	owner := c.String("owner")
	githubToken := c.String("github-token")
	dryRun := c.Bool("dry-run")

	branches := strings.Split(rawBranches, ",")
	if len(branches) < 1 || branches[0] == "" {
		return errors.New("no branches specified")
	}
	commits := strings.Split(rawCommits, ",")
	if len(commits) < 1 || commits[0] == "" {
		return errors.New("no commits specified")
	}

	ctx := context.Background()
	githubClient := repository.NewGithub(ctx, githubToken)

	if owner == "" {
		defaultOwner, err := repository.OrgFromRepo(repo)
		if err != nil {
			return err
		}
		owner = defaultOwner
	}
	pbo := &repository.PerformBackportOpts{
		Owner:    owner,
		Repo:     repo,
		Commits:  commits,
		IssueID:  issue,
		Branches: branches,
		User:     user,
		DryRun:   dryRun,
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
