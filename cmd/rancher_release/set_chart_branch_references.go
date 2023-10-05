package main

import (
	"context"
	"errors"
	"os"

	"github.com/rancher/ecm-distro-tools/release/rancher"
	"github.com/urfave/cli"
)

func setChartsBranchReferencesCommand() cli.Command {
	return cli.Command{
		Name:  "set-charts-branch-refs",
		Usage: "set charts branch references in files",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:     "fork-path",
				Usage:    "rancher repo fork directory path",
				Required: true,
			},
			cli.StringFlag{
				Name:     "base-branch",
				Usage:    "rancher branch to use as a base, e.g: release/v2.8",
				Required: true,
			},
			cli.StringFlag{
				Name:     "current-charts-branch",
				Usage:    "current branch set for charts in the repo",
				Required: true,
			},
			cli.StringFlag{
				Name:     "new-charts-branch",
				Usage:    "branch to be replaced in charts in the repo",
				Required: true,
			},
			cli.BoolFlag{
				Name:  "create-pr",
				Usage: "if true, a PR will be created from your fork to the rancher repo base branch and a variable 'GITHUB_TOKEN' must be exported",
			},
			cli.StringFlag{
				Name:     "fork-owner",
				Usage:    "github username of the owner of the fork, only required if 'create-pr' is true",
				Required: false,
			},
			cli.BoolFlag{
				Name:     "dry-run",
				Usage:    "the newly created branch won't be pushed to remote and the PR won't be created",
				Required: false,
			},
		},
		Action: setChartBranchReferences,
	}
}

func setChartBranchReferences(c *cli.Context) error {
	forkPath := c.String("fork-path")
	baseBranch := c.String("base-branch")
	currentBranch := c.String("current-charts-branch")
	newBranch := c.String("new-charts-branch")
	createPR := c.BoolT("create-pr")
	forkOwner := c.String("fork-owner")
	githubToken := os.Getenv("GITHUB_TOKEN")
	dryRun := c.BoolT("dry-run")
	if createPR {
		if forkOwner == "" {
			return errors.New("'create-pr' requires 'fork-owner'")
		}
		if githubToken == "" {
			return errors.New("'create-pr' requires the 'GITHUB_TOKEN' env var")
		}
	}

	return rancher.SetChartBranchReferences(context.Background(), forkPath, baseBranch, currentBranch, newBranch, forkOwner, githubToken, createPR, dryRun)
}
