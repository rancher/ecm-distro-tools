package main

import (
	"context"

	"github.com/rancher/ecm-distro-tools/release/k3s"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func modifyK3SCommand() cli.Command {
	return cli.Command{
		Name:   "modify-k3s",
		Usage:  "Modify k3s go.mod with the updated tags and create a new PR",
		Flags:  rootFlags,
		Action: modifyK3S,
	}
}

func modifyK3S(c *cli.Context) error {
	ctx := context.Background()
	configPath := c.String("config")
	release, err := k3s.NewReleaseFromConfig(configPath)
	if err != nil {
		logrus.Fatalf("failed to read config file: %v", err)
	}
	client, err := k3s.NewGithubClient(ctx, release.Token)
	if err != nil {
		logrus.Fatalf("failed to initialize a new github client from token: %v", err)
	}

	if err := release.ModifyAndPush(ctx); err != nil {
		logrus.Fatalf("failed to modify k3s go.mod: %v", err)
	}

	if err := release.CreatePRFromK3S(ctx, client); err != nil {
		logrus.Fatalf("failed to create a new PR: %v", err)
	}

	return nil
}
