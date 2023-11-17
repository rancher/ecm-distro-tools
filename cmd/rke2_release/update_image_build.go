package main

import (
	"context"
	"errors"

	"github.com/rancher/ecm-distro-tools/release/rke2"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/urfave/cli/v2"
)

func updateImageBuildCommand() *cli.Command {
	return &cli.Command{
		Name:  "update-image-build",
		Usage: "checks if a new release is available at rancher/image-build-base and updates the references at the rancher/image-build repos",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "github-token",
				Aliases:  []string{"g"},
				EnvVars:  []string{"GITHUB_TOKEN"},
				Required: true,
			},
			&cli.StringFlag{
				Name:     "repo",
				Aliases:  []string{"r"},
				Required: true,
			},
			&cli.StringFlag{
				Name:     "owner",
				Aliases:  []string{"o"},
				Required: true,
			},
			&cli.StringFlag{
				Name:     "clone-dir",
				Aliases:  []string{"c"},
				Required: true,
			},
			&cli.BoolFlag{
				Name:     "dry-run",
				Aliases:  []string{"d"},
				Required: false,
			},
			&cli.BoolFlag{
				Name:     "create-pr",
				Aliases:  []string{"p"},
				Required: false,
			},
		},
		Action: updateImageBuild,
	}
}

func updateImageBuild(c *cli.Context) error {
	token := c.String("github-token")
	if token == "" {
		return errors.New("env var GITHUB_TOKEN is required")
	}
	repo := c.String("repo")
	owner := c.String("owner")
	cloneDir := c.String("clone-dir")
	dryRun := c.Bool("dry-run")
	createPR := c.Bool("create-pr")
	ctx := context.Background()
	ghClient := repository.NewGithub(ctx, token)
	return rke2.UpdateImageBuild(ctx, ghClient, repo, owner, cloneDir, dryRun, createPR)
}
