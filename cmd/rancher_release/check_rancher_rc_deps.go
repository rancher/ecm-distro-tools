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
		Usage: "check if the Rancher version specified by the commit does not contain development dependencies and rc tags",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "commit",
				Aliases:  []string{"c"},
				Usage:    "last commit for a final rc",
				Required: false,
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
				Usage:    "organization name",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "repo",
				Aliases:  []string{"r"},
				Usage:    "rancher repository from process",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "files",
				Aliases:  []string{"f"},
				Usage:    "files to be checked",
				Required: true,
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

	if rcCommit == "" && rcReleaseTitle == "" {
		return errors.New("'commit' or 'release-title' are required")
	}
	if rcFiles == "" {
		return errors.New("'files' is required, e.g, --files Dockerfile.dapper,go.mod")
	}
	if rcOrg == "" {
		return errors.New("'org' is required")
	}
	logrus.Debugf("organization: %s, repository: %s, commit: %s, release title: %s, files: %s",
		rcOrg, rcRepo, rcCommit, rcReleaseTitle, rcFiles)

	return rancher.CheckRancherFinalRCDeps(rcOrg, rcRepo, rcCommit, rcReleaseTitle, rcFiles)
}
