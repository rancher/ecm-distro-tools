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
	app.Name = "rancher-release"
	app.Usage = "Perform a Rancher release"
	app.UseShortOptionHandling = true
	app.Commands = []*cli.Command{
		listImagesRCCommand(),
		checkRancherImageCommand(),
		setKDMBranchReferencesCommand(),
		setChartsBranchReferencesCommand(),
		checkRancherRCDepsCommand(),
	}
	app.Flags = rootFlags

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}
