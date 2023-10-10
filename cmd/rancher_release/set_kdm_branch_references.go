package main

import (
	"context"
	"errors"
	"os"

	"github.com/rancher/ecm-distro-tools/release/rancher"
	"github.com/urfave/cli/v2"
)

func setKDMBranchReferencesCommand() *cli.Command {
	return &cli.Command{
		Name:  "set-kdm-branch-refs",
		Usage: "set kdm branch references in files",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "fork-path",
				Usage:    "rancher repo fork directory path",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "base-branch",
				Usage:    "rancher branch to use as a base, e.g: release/v2.8",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "current-kdm-branch",
				Usage:    "current kdm branch set in the repo",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "new-kdm-branch",
				Usage:    "kdm branch to be replaced in the repo",
				Required: true,
			},
			&cli.BoolFlag{
				Name:  "create-pr",
				Usage: "if true, a PR will be created from your fork to the rancher repo base branch and a variable 'GITHUB_TOKEN' must be exported",
			},
			&cli.StringFlag{
				Name:     "fork-owner",
				Usage:    "github username of the owner of the fork, only required if 'create-pr' is true",
				Required: false,
			},
			&cli.BoolFlag{
				Name:     "dry-run",
				Usage:    "the newly created branch won't be pushed to remote and the PR won't be created",
				Required: false,
			},
		},
		Action: setKDMBranchReferences,
	}
}

func setKDMBranchReferences(c *cli.Context) error {
	forkPath := c.String("fork-path")
	baseBranch := c.String("base-branch")
	currentKDMBranch := c.String("current-kdm-branch")
	newKDMBranch := c.String("new-kdm-branch")
	createPR := c.Bool("create-pr")
	forkOwner := c.String("fork-owner")
	githubToken := os.Getenv("GITHUB_TOKEN")
	dryRun := c.Bool("dry-run")
	if createPR {
		if forkOwner == "" {
			return errors.New("'create-pr' requires 'fork-owner'")
		}
		if githubToken == "" {
			return errors.New("'create-pr' requires the 'GITHUB_TOKEN' env var")
		}
	}

	return rancher.SetKDMBranchReferences(context.Background(), forkPath, baseBranch, currentKDMBranch, newKDMBranch, forkOwner, githubToken, createPR, dryRun)
}
