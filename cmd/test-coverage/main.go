package main

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	rootFlags = []cli.Flag{
		&cli.BoolFlag{
			Name:  "verbose,v ",
			Usage: "verbose output",
		},
		&cli.BoolFlag{
			Name:  "graph,g ",
			Usage: "display results as a graph",
		},
		&cli.StringFlag{
			Name:     "path, p",
			Usage:    "path to K3s/RKE2 repository",
			Required: true,
		},
	}
)

func main() {
	app := cli.NewApp()
	app.Name = "test-coverage"
	app.Usage = "Generate coverage report for E2E/Integration tests"
	app.Flags = rootFlags
	app.Action = coverage
	err := app.Run(os.Args)
	if err != nil {
		logrus.Fatal(err)
	}

}
