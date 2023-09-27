package main

import (
	"fmt"

	"github.com/rancher/ecm-distro-tools/release/rancher"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func listImagesRCCommand() cli.Command {
	return cli.Command{
		Name:  "list-nonmirrored-rc-images",
		Usage: "list all non-mirrored images which are in rc form given a tag in a MD format",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:     "tag",
				Usage:    "release tag to validate images",
				Required: true,
			},
		},
		Action: listImagesRC,
	}
}

func listImagesRC(c *cli.Context) error {
	tag := c.String("tag")
	logrus.Debug("tag: " + tag)

	imagesRC, err := rancher.ListRancherImagesRC(tag)
	if err != nil {
		return err
	}
	fmt.Println(imagesRC)
	return nil
}
