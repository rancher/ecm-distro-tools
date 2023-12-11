package main

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var rootFlags = []cli.Flag{
	&cli.StringFlag{
		Name:    "config",
		Aliases: []string{"c"},
		Usage:   "Specify release config file",
		EnvVars: []string{"RELEASE_CONFIG"},
	},
	&cli.BoolFlag{
		Name:    "debug",
		Aliases: []string{"d"},
		Usage:   "Debug mode",
	},
}

func main() {
	app := cli.NewApp()
	app.Name = "k3s-release"
	app.Usage = "Perform a k3s release"
	app.UseShortOptionHandling = true
	app.Commands = []*cli.Command{
		createTagsCommand(),
		pushTagsCommand(),
		modifyK3SCommand(),
		tagRCReleaseCommand(),
		tagReleaseCommand(),
		releaseCommand(),
	}
	app.Flags = rootFlags

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}
