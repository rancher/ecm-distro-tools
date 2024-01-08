package main

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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
	}
	app.Flags = rootFlags

	// if err := app.Run(os.Args); err != nil {
	// 	logrus.Fatal(err)
	// }
	var rootCmd = &cobra.Command{
		Use:   "k3s-release",
		Short: "Perform a k3s release",
	}
	rootCmd.AddCommand(configCommand())

	if err := rootCmd.Execute(); err != nil {
		logrus.Fatal(err)
	}
}
