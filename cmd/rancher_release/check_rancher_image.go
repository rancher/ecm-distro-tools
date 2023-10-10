package main

import (
	"context"

	"github.com/rancher/ecm-distro-tools/release/rancher"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

const (
	rancherOrg  = "rancher"
	rancherRepo = rancherOrg
)

func checkRancherImageCommand() *cli.Command {
	return &cli.Command{
		Name:  "check-rancher-image",
		Usage: "check if rancher helm charts and docker images exist for a given image tag",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "tag",
				Aliases:  []string{"t"},
				Usage:    "release tag to validate image",
				Required: true,
			},
		},
		Action: checkRancherImage,
	}
}

func checkRancherImage(c *cli.Context) error {
	tag := c.String("tag")
	logrus.Debug("tag: " + tag)
	rancherArchs := []string{"amd64", "arm64", "s390x"}
	if err := rancher.CheckRancherDockerImage(context.Background(), rancherOrg, rancherRepo, tag, rancherArchs); err != nil {
		return err
	}
	return rancher.CheckHelmChartVersion(tag)
}
