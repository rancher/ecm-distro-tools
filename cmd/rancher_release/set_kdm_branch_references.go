package main

import (
	"github.com/rancher/ecm-distro-tools/release/rancher"
	"github.com/urfave/cli"
)

func setKDMBranchReferencesCommand() cli.Command {
	return cli.Command{
		Name:  "set-kdm-branch-references",
		Usage: "set kdm branch references in files",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:     "rancher-repo-dir",
				Usage:    "rancher repo directory",
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
		},
		Action: setKDMBranchReferences,
	}
}

func setKDMBranchReferences(c *cli.Context) error {
	rancherRepoDir := c.String("rancher-repo-dir")
	baseBranch := c.String("rancher-base-branch")
	currentKDMBranch := c.String("current-kdm-branch")
	newKDMBranch := c.String("new-kdm-branch")

	return rancher.SetKDMBranchReferences(rancherRepoDir, baseBranch, currentKDMBranch, newKDMBranch)
}
