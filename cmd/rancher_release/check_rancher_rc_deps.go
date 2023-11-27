package main

import (
	"context"

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
				Name:     "org",
				Aliases:  []string{"o"},
				Usage:    "organization name",
				Required: false,
				Value:    "rancher",
			},
			&cli.StringFlag{
				Name:     "repo",
				Aliases:  []string{"r"},
				Usage:    "rancher repository",
				Required: false,
				Value:    "rancher",
			},
			&cli.StringFlag{
				Name:     "files",
				Aliases:  []string{"f"},
				Usage:    "files to be checked if remotely",
				Required: false,
			},
			&cli.BoolFlag{
				Name:     "for-ci",
				Aliases:  []string{"p"},
				Usage:    "export a md template also check raising an error if contains rc tags and dev deps",
				Required: false,
			},
		},
		Action: checkRancherRCDeps,
	}
}

func checkRancherRCDeps(c *cli.Context) error {
	const files = "/bin/rancher-images.txt,/bin/rancher-windows-images.txt,Dockerfile.dapper,go.mod,/package/Dockerfile,/pkg/apis/go.mod,/pkg/settings/setting.go,/scripts/package-env"

	var local bool

	rcCommit := c.String("commit")
	rcOrg := c.String("org")
	rcRepo := c.String("repo")
	rcFiles := c.String("files")
	forCi := c.Bool("for-ci")

	if rcFiles == "" {
		rcFiles = files
	}
	if rcCommit == "" {
		local = true
	}

	logrus.Debugf("organization: %s, repository: %s, commit: %s, files: %s",
		rcOrg, rcRepo, rcCommit, rcFiles)

	err := rancher.CheckRancherRCDeps(context.Background(), local, forCi, rcOrg, rcRepo, rcCommit, rcFiles)
	if err != nil {
		return err
	}
	return nil
}
