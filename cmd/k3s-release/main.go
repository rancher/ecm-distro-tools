package main

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	rootFlags = []cli.Flag{
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
)

func main() {
	app := cli.NewApp()
	app.Name = "k3s-release"
	app.Usage = "Start a k3s release"
	app.Commands = []cli.Command{
		CreateTagsCommand(),
		PushTagsCommand(),
		ModifyK3SCommand(),
	}
	app.Flags = rootFlags
	err := app.Run(os.Args)
	if err != nil {
		logrus.Fatal(err)
	}
}
