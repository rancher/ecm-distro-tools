package main

import (
	"context"

	"github.com/rancher/ecm-distro-tools/release/k3s"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

const (
	k3sRemote = "k3s-io"
)

func PushTagsCommand() cli.Command {
	return cli.Command{
		Name:   "push-tags",
		Usage:  "Push tags to speicifc remote",
		Flags:  rootFlags,
		Action: PushTags,
	}
}

func PushTags(c *cli.Context) error {
	ctx := context.Background()
	// pushing tags to k3s-io kubernetes fork
	release, err := k3s.NewReleaseFromConfig(c)
	if err != nil {
		logrus.Fatalf("failed to read config file: %v", err)
	}
	client, err := k3s.NewGithubClient(ctx, release.Token)
	if err != nil {
		logrus.Fatalf("failed to initialize a new github client from token: %v", err)
	}
	// this subcommand depends on tag file being created in workspace
	tags, err := release.TagsFromFile(ctx)
	if err != nil {
		logrus.Fatalf("failed to extract tags from file: %v", err)
	}
	if err := release.PushTags(ctx, tags, client, k3sRemote); err != nil {
		logrus.Fatalf("failed to push tags: %v", err)
	}
	return nil
}
