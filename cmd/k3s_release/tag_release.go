package main

import (
	"context"

	ecmGithub "github.com/rancher/ecm-distro-tools/github"
	"github.com/rancher/ecm-distro-tools/release/k3s"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func tagReleaseCommand() cli.Command {
	return cli.Command{
		Name:   "tag-release",
		Usage:  "tag release for k3s repo",
		Flags:  rootFlags,
		Action: createRelease,
	}
}

func createRelease(c *cli.Context) error {
	ctx := context.Background()

	configPath := c.String("config")

	release, err := k3s.NewRelease(configPath)
	if err != nil {
		logrus.Fatalf("failed to read config file: %v", err)
	}

	client, err := ecmGithub.NewGithubClient(ctx, release.Token)
	if err != nil {
		logrus.Fatalf("failed to initialize a new github client from token: %v", err)
	}

	return release.CreateRelease(ctx, client, false)
}
