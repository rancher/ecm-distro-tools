package main

import (
	"context"

	"github.com/rancher/ecm-distro-tools/release/rke2"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func imageBuildCommand() *cli.Command {
	return &cli.Command{
		Name: "image-build",
		Subcommands: []*cli.Command{
			{
				Name:   "list-repos",
				Usage:  "List all repos supported by this command",
				Action: listRepos,
			},
			{
				Name:  "update",
				Usage: "updates hardened-build-base references in the rancher/image-build-* repos",
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
						Usage:    "What image-build repo to update, use `rke2_release image-build list-repos` for a full list",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "owner",
						Aliases:  []string{"o"},
						Usage:    "Owner of the repo, default is 'rancher' only used for testing purposes",
						Value:    "rancher",
						Required: false,
					},
					&cli.StringFlag{
						Name:     "repo-path",
						Aliases:  []string{"p"},
						Usage:    "Local copy of the image-build repo that is being updated",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "working-dir",
						Aliases:  []string{"w"},
						Usage:    "Directory in which the temporary scripts will be created, default is /tmp",
						Value:    "/tmp",
						Required: false,
					},
					&cli.StringFlag{
						Name:     "build-base-tag",
						Aliases:  []string{"t"},
						Usage:    "hardened-build-base Docker image tag to update the references in the repo to",
						Required: true,
					},
					&cli.BoolFlag{
						Name:     "dry-run",
						Aliases:  []string{"d"},
						Usage:    "Don't push changes to remote and don't create the PR, just log the information",
						Required: false,
					},
					&cli.BoolFlag{
						Name:     "create-pr",
						Aliases:  []string{"c"},
						Usage:    "If not set, the images will be pushed to a new branch, but a PR won't be created",
						Required: false,
					},
				},
				Action: updateImageBuild,
			},
		},
	}
}

func updateImageBuild(c *cli.Context) error {
	token := c.String("github-token")
	repo := c.String("repo")
	owner := c.String("owner")
	repoPath := c.String("repo-path")
	workingDir := c.String("working-dir")
	buildBaseTag := c.String("build-base-tag")
	dryRun := c.Bool("dry-run")
	createPR := c.Bool("create-pr")
	ctx := context.Background()
	ghClient := repository.NewGithub(ctx, token)
	return rke2.UpdateImageBuild(ctx, ghClient, repo, owner, repoPath, workingDir, buildBaseTag, dryRun, createPR)
}

func listRepos(c *cli.Context) error {
	repos := ""
	for k := range rke2.ImageBuildRepos {
		repos += "  " + k + "\n"
	}
	logrus.Info("Supported repos: \n" + repos)
	return nil
}
