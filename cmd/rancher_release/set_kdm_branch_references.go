package main

import (
	"context"
	"errors"

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
				Aliases:  []string{"f"},
				Usage:    "rancher repo fork directory path",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "base-branch",
				Aliases:  []string{"b"},
				Usage:    "rancher branch to use as a base, e.g: release/v2.8",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "current-kdm-branch",
				Aliases:  []string{"c"},
				Usage:    "current kdm branch set in the repo",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "new-kdm-branch",
				Aliases:  []string{"n"},
				Usage:    "kdm branch to be replaced in the repo",
				Required: true,
			},
			&cli.BoolFlag{
				Name:    "create-pr",
				Aliases: []string{"p"},
				Usage:   "if true, a PR will be created from your fork to the rancher repo base branch and a variable 'GITHUB_TOKEN' must be exported",
			},
			&cli.StringFlag{
				Name:     "fork-owner",
				Aliases:  []string{"o"},
				Usage:    "github username of the owner of the fork, only required if 'create-pr' is true",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "github-token",
				Aliases:  []string{"g"},
				Usage:    "github token",
				EnvVars:  []string{"GITHUB_TOKEN"},
				Required: false,
			},
			&cli.BoolFlag{
				Name:     "dry-run",
				Aliases:  []string{"r"},
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
	githubToken := c.String("github-token")
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
