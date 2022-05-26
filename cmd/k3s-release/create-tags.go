package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/rancher/ecm-distro-tools/release/k3s"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func CreateTagsCommand() cli.Command {
	return cli.Command{
		Name:   "create-tags",
		Usage:  "Create tags from kubernetes repo",
		Flags:  rootFlags,
		Action: CreateTags,
	}
}

func CreateTags(c *cli.Context) error {
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

	err = release.SetupK8sRemotes(ctx, client)
	if err != nil {
		logrus.Fatalf("failed to clone and setup remotes for k8s repo: %v", err)
	}
	// all business logic should exists in the package
	tagsCreated, err := release.TagsCreated(ctx)
	if err != nil {
		return err
	}
	if tagsCreated {
		logrus.Infof("Tag file already exists, skipping rebase and tag")
		return nil
	}
	tags, err := release.RebaseAndTag(ctx, client)
	if err != nil {
		logrus.Fatalf("failed to rebase and create tags: %v", err)
	}
	tagFile := filepath.Join(release.Workspace, "tags-"+release.NewK8SVersion)
	err = os.WriteFile(tagFile, []byte(strings.Join(tags, "\n")), 0644)
	if err != nil {
		logrus.Fatalf("failed to write tags file: %v", err)
	}
	return nil
}
