package main

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var rootFlags = []cli.Flag{
	cli.StringFlag{
		Name:   "config",
		Usage:  "Specify release config file",
		EnvVar: "RELEASE_CONFIG",
	},
	cli.BoolFlag{
		Name:  "debug",
		Usage: "Debug mode",
	},
}

func main() {
	app := cli.NewApp()
	app.Name = "k3s-release"
	app.Usage = "Perform a k3s release"
	app.Commands = []cli.Command{
		createTagsCommand(),
		pushTagsCommand(),
		modifyK3SCommand(),
		tagRCReleaseCommand(),
		tagReleaseCommand(),
	}
	app.Flags = rootFlags

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}
