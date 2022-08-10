package main

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	rootFlags = []cli.Flag{
		&cli.BoolFlag{
			Name:  "debug,d ",
			Usage: "debug mode",
		},
		&cli.StringFlag{
			Name:     "program, p",
			Usage:    "program name [k3s|rke2]",
			Required: true,
		},
		&cli.StringFlag{
			Name:  "commit, c",
			Usage: "commit hash to generate coverage report for",
		},
	}
)

func main() {
	app := cli.NewApp()
	app.Name = "coverage"
	app.Usage = "Generate coverage report for E2E/integration tests"
	app.Flags = rootFlags
	app.Action = coverage
	err := app.Run(os.Args)
	if err != nil {
		logrus.Fatal(err)
	}

}
