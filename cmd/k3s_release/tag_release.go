package main

import (
	"context"

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

	client, err := k3s.NewGithubClient(ctx, release.Token)
	if err != nil {
		logrus.Fatalf("failed to initialize a new github client from token: %v", err)
	}

	err = release.CreateRelease(ctx, client, false)
	if err != nil {
		return err
	}
	return nil
}
