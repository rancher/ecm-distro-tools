package main

import (
	"errors"

	"github.com/rancher/ecm-distro-tools/release/rancher"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func checkRancherRCDepsCommand() *cli.Command {
	return &cli.Command{
		Name:  "check-rancher-rc-deps",
		Usage: "check if rancher commit contains the necessary values for a final rc",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "commit",
				Aliases:  []string{"c"},
				Usage:    "last commit for a final rc",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "release-title",
				Aliases:  []string{"rt"},
				Usage:    "release title from a given release process",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "org",
				Aliases:  []string{"o"},
				Usage:    "organization name, e.g rancher,suse",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "repo",
				Aliases:  []string{"r"},
				Usage:    "repository from process",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "files",
				Aliases:  []string{"f"},
				Usage:    "files to be checked",
				Required: false,
			},
		},
		Action: checkRancherRCDeps,
	}
}

func checkRancherRCDeps(c *cli.Context) error {
	rcReleaseTitle := c.String("release-title")
	rcCommit := c.String("commit")
	rcOrg := c.String("org")
	rcRepo := c.String("repo")
	rcFiles := c.String("files")

	if rcFiles == "" {
		return errors.New("'files' is required, e.g  --files Dockerfile.dapper,go.mod ")
	}
	if rcCommit == "" && rcReleaseTitle == "" {
		return errors.New("'commit' or 'release-title' are required")
	}
	if rcOrg == "" {
		return errors.New("'org' is required")
	}
	logrus.Debug("commit: " + rcCommit)

	return rancher.CheckRancherFinalRCDeps(rcOrg, rcRepo, rcCommit, rcReleaseTitle, rcFiles)
}
