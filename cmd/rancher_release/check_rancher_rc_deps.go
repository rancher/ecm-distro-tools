package main

import (
	"errors"
	"fmt"

	"github.com/rancher/ecm-distro-tools/release/rancher"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func checkRancherRCDepsCommand() *cli.Command {
	return &cli.Command{
		Name:  "check-rancher-rc-deps",
		Usage: "check if the Rancher version specified by the commit or pre-release title does not contain development dependencies and rc tags",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "commit",
				Aliases:  []string{"c"},
				Usage:    "last commit for a final rc",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "release-title",
				Aliases:  []string{"t"},
				Usage:    "release title from a given release process",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "org",
				Aliases:  []string{"o"},
				Usage:    "organization name",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "repo",
				Aliases:  []string{"r"},
				Usage:    "rancher repository",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "files",
				Aliases:  []string{"f"},
				Usage:    "files to be checked",
				Required: true,
			},
			&cli.BoolFlag{
				Name:     "for-ci",
				Aliases:  []string{"p"},
				Usage:    "export a md template also check raising a error if contains rc tags and dev deps",
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
	forCi := c.Bool("for-ci")

	if rcCommit == "" && rcReleaseTitle == "" {
		return errors.New("'commit' or 'release-title' are required")
	}
	if rcFiles == "" {
		return errors.New("'files' is required, e.g, --files Dockerfile.dapper,go.mod")
	}
	logrus.Debugf("organization: %s, repository: %s, commit: %s, release title: %s, files: %s",
		rcOrg, rcRepo, rcCommit, rcReleaseTitle, rcFiles)

	output, err := rancher.CheckRancherRCDeps(forCi, rcOrg, rcRepo, rcCommit, rcReleaseTitle, rcFiles)
	if err != nil {
		return err
	}

	fmt.Println(output)

	return nil
}
