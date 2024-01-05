package main

import (
	"github.com/rancher/ecm-distro-tools/release/k3s"
	"github.com/urfave/cli/v2"
)

func generateConfigCommand() *cli.Command {
	return &cli.Command{
		Name:  "generate-config",
		Usage: "Generate a k3s release configuration file",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "version",
				Aliases:  []string{"v"},
				Usage:    "version to generate the config for",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "ssh-path",
				Aliases:  []string{"s"},
				Usage:    "ssh key path",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "generate-path",
				Aliases:  []string{"g"},
				Usage:    "path to generate the configuration in",
				Value:    ".",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "workspace",
				Aliases:  []string{"w"},
				Usage:    "directory to clone repositories and create files",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "handler",
				Aliases:  []string{"r"},
				Usage:    "your github handler",
				Required: true,
			},
		},
		Action: generateConfg,
	}
}

func generateConfg(c *cli.Context) error {
	version := c.String("version")
	sshPath := c.String("ssh-path")
	generatePath := c.String("generate-path")
	workspace := c.String("workspace")
	handler := c.String("handler")
	return k3s.GenerateConfig(version, sshPath, generatePath, workspace, handler)
}
