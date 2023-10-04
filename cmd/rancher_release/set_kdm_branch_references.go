package main

import (
	"context"
	"errors"
	"os"

	"github.com/rancher/ecm-distro-tools/release/rancher"
	"github.com/urfave/cli"
)

func setKDMBranchReferencesCommand() cli.Command {
	return cli.Command{
		Name:  "set-kdm-branch-references",
		Usage: "set kdm branch references in files",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:     "rancher-fork-dir",
				Usage:    "rancher repo fork directory",
				Required: true,
			},
			cli.StringFlag{
				Name:     "rancher-base-branch",
				Usage:    "rancher branch to use as a base, e.g: release/v2.8",
				Required: true,
			},
			cli.StringFlag{
				Name:     "current-kdm-branch",
				Usage:    "current kdm branch set in the repo",
				Required: true,
			},
			cli.StringFlag{
				Name:     "new-kdm-branch",
				Usage:    "kdm branch to be replaced in the repo",
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
		},
		Action: setKDMBranchReferences,
	}
}

func setKDMBranchReferences(c *cli.Context) error {
	rancherForkDir := c.String("rancher-fork-dir")
	baseBranch := c.String("rancher-base-branch")
	currentKDMBranch := c.String("current-kdm-branch")
	newKDMBranch := c.String("new-kdm-branch")
	createPR := c.BoolT("create-pr")
	forkOwner := c.String("fork-owner")
	githubToken := os.Getenv("GITHUB_TOKEN")
	if createPR {
		if forkOwner == "" {
			return errors.New("if 'create-pr' is true, fork-owner is required")
		}
		if githubToken == "" {
			return errors.New("if 'create-pr' is true, the 'GITHUB_TOKEN' env variable is required")
		}
	}

	return rancher.SetKDMBranchReferences(context.Background(), rancherForkDir, baseBranch, currentKDMBranch, newKDMBranch, forkOwner, githubToken, createPR)
}
