package main

import (
	"context"

	"github.com/rancher/ecm-distro-tools/release/k3s"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func pushTagsCommand() *cli.Command {
	return &cli.Command{
		Name:   "push-tags",
		Usage:  "Push tags to speicifc remote",
		Flags:  rootFlags,
		Action: pushTags,
	}
}

func pushTags(c *cli.Context) error {
	ctx := context.Background()

	// pushing tags to k3s-io kubernetes fork
	configPath := c.String("config")

	release, err := k3s.NewRelease(configPath)
	if err != nil {
		logrus.Fatalf("failed to read config file: %v", err)
	}

	client, err := k3s.NewGithubClient(ctx, release.GithubToken)
	if err != nil {
		logrus.Fatalf("failed to initialize a new github client from token: %v", err)
	}

	// this subcommand depends on tag file being created in workspace
	tags, err := release.TagsFromFile(ctx)
	if err != nil {
		logrus.Fatalf("failed to extract tags from file: %v", err)
	}

	logrus.Infof("pushing tags to github")
	if err := release.PushTags(ctx, tags, client); err != nil {
		logrus.Fatalf("failed to push tags: %v", err)
	}

	return nil
}
