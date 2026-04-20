package main

import (
	"context"
	"errors"

	"github.com/rancher/ecm-distro-tools/release/rke2"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/urfave/cli/v3"
)

func imageBuildBaseReleaseCommand() *cli.Command {
	return &cli.Command{
		Name:  "image-build-base-release",
		Usage: "checks if new golang versions are available and creates new releases for rancher/image-build-base",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "github-token",
				Aliases:  []string{"g"},
				EnvVars:  []string{"GITHUB_TOKEN"},
				Required: true,
			},
			&cli.BoolFlag{
				Name:     "dry-run",
				Aliases:  []string{"r"},
				Required: false,
			},
		},
		Action: imageBuildBaseRelease,
	}
}

func imageBuildBaseRelease(c *cli.Context) error {
	token := c.String("github-token")
	if token == "" {
		return errors.New("env var GITHUB_TOKEN is required")
	}
	dryRun := c.Bool("dry-run")
	ctx := context.Background()
	ghClient := repository.NewGithub(ctx, token)
	return rke2.ImageBuildBaseRelease(ctx, ghClient, dryRun)
}
