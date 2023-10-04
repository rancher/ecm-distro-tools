package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	ecmGithub "github.com/rancher/ecm-distro-tools/github"
	"github.com/rancher/ecm-distro-tools/release/k3s"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func createTagsCommand() cli.Command {
	return cli.Command{
		Name:   "create-tags",
		Usage:  "Create tags from kubernetes repo",
		Flags:  rootFlags,
		Action: createTags,
	}
}

func createTags(c *cli.Context) error {
	ctx := context.Background()

	configPath := c.String("config")

	release, err := k3s.NewRelease(configPath)
	if err != nil {
		logrus.Fatalf("config file: %v", err)
	}

	client, err := ecmGithub.NewGithubClient(ctx, release.Token)
	if err != nil {
		logrus.Fatalf("failed to initialize a new github client from token: %v", err)
	}

	if err := release.SetupK8sRemotes(ctx, client); err != nil {
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

	tags, rebaseOut, err := release.RebaseAndTag(ctx, client)
	if err != nil {
		logrus.Fatalf("failed to rebase and create tags: %v", err)
	}
	logrus.Info("successfully rebased and tagged")
	logrus.Info(rebaseOut)

	tagFile := filepath.Join(release.Workspace, "tags-"+release.NewK8SVersion)
	if err := os.WriteFile(tagFile, []byte(strings.Join(tags, "\n")), 0644); err != nil {
		logrus.Fatalf("failed to write tags file: %v", err)
	}
	return nil
}
