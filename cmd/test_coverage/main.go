package main

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var (
	version   = "development"
	rootFlags = []cli.Flag{
		&cli.BoolFlag{
			Name:    "verbose",
			Aliases: []string{"v"},
			Usage:   "verbose output",
		},
		&cli.BoolFlag{
			Name:    "graph",
			Aliases: []string{"g"},
			Usage:   "display results as a graph",
		},
		&cli.BoolFlag{
			Name:    "table",
			Aliases: []string{"t"},
			Usage:   "display results as a markdown table",
		},
		&cli.BoolFlag{
			Name:    "list",
			Aliases: []string{"l"},
			Usage:   "display results as a list",
		},
		&cli.StringFlag{
			Name:     "path",
			Aliases:  []string{"p"},
			Usage:    "path to K3s/RKE2 repository",
			Required: true,
		},
	}
)

func main() {
	app := cli.NewApp()
	app.Name = "test-coverage"
	app.Usage = "Generate coverage report for E2E/Integration tests"
	app.UseShortOptionHandling = true
	app.Flags = rootFlags
	app.Action = coverage
	app.Version = version
	err := app.Run(os.Args)
	if err != nil {
		logrus.Fatal(err)
	}

}
