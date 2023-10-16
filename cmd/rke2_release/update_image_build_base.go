package main

import (
	"context"
	"errors"

	"github.com/rancher/ecm-distro-tools/release/rke2"
	"github.com/urfave/cli/v2"
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
		},
		Action: imageBuildBaseRelease,
	}
}

func imageBuildBaseRelease(c *cli.Context) error {
	token := c.String("github-token")
	if token == "" {
		return errors.New("env var GITHUB_TOKEN is required")
	}
	return rke2.ImageBuildBaseRelease(context.Background(), token)
}
