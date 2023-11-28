package main

import (
	"context"

	"github.com/rancher/ecm-distro-tools/release/rancher"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/urfave/cli/v2"
)

func tagReleaseCommand() *cli.Command {
	return &cli.Command{
		Name:  "tag-release",
		Usage: "tag a rancher release",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "github-token",
				Aliases:  []string{"g"},
				EnvVars:  []string{"GITHUB_TOKEN"},
				Required: true,
			},
			&cli.StringFlag{
				Name:     "tag",
				Aliases:  []string{"t"},
				Usage:    "release tag",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "remote-branch",
				Aliases:  []string{"r"},
				Usage:    "rancher remote branch",
				Required: true,
			},
			&cli.BoolFlag{
				Name:     "general-availability",
				Aliases:  []string{"a"},
				Usage:    "by default, the release will be created as a pre-release, make sure it absolutely needs to be a GA release before setting this",
				Required: false,
			},
			&cli.BoolFlag{
				Name:     "ignore-draft",
				Aliases:  []string{"d"},
				Usage:    "by default, the release will be created as a draft, so you can verify all information is correct before unmarking it",
				Required: false,
			},
		},
		Action: tagRelease,
	}
}

func tagRelease(c *cli.Context) error {
	token := c.String("github-token")
	tag := c.String("tag")
	remoteBranch := c.String("remote-branch")
	generalAvailability := c.Bool("general-availability")
	ignoreDraft := c.Bool("ignore-draft")
	ctx := context.Background()
	ghClient := repository.NewGithub(ctx, token)
	return rancher.TagRancherRelease(ctx, ghClient, tag, remoteBranch, generalAvailability, ignoreDraft)
}
