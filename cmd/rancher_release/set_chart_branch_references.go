package main

import (
	"context"
	"errors"
	"os"
	"strconv"

	"github.com/rancher/ecm-distro-tools/release/rancher"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func setChartsBranchReferencesCommand() *cli.Command {
	return &cli.Command{
		Name:  "set-charts-branch-refs",
		Usage: "set charts branch references in files",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "fork-path",
				Aliases:  []string{"f"},
				Usage:    "rancher repo fork directory path",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "base-branch",
				Aliases:  []string{"b"},
				Usage:    "rancher branch to use as a base, e.g: release/v2.8",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "new-charts-branch",
				Aliases:  []string{"n"},
				Usage:    "new branch to replace the current",
				Required: true,
			},
			&cli.BoolFlag{
				Name:    "create-pr",
				Aliases: []string{"p"},
				Usage:   "if true, a PR will be created from your fork to the rancher repo base branch and a variable 'GITHUB_TOKEN' must be exported",
			},
			&cli.StringFlag{
				Name:     "github-user",
				Aliases:  []string{"u"},
				Usage:    "github username of the owner of the fork, only required if 'create-pr' is true",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "github-token",
				Aliases:  []string{"g"},
				Usage:    "github token",
				EnvVars:  []string{"GITHUB_TOKEN"},
				Required: false,
			},
			&cli.BoolFlag{
				Name:     "dry-run",
				Aliases:  []string{"r"},
				Usage:    "the newly created branch won't be pushed to remote and the PR won't be created",
				Required: false,
			},
		},
		Action: setChartBranchReferences,
	}
}

func setChartBranchReferences(c *cli.Context) error {
	forkPath := c.String("fork-path")
	if forkPath == "" {
		fp, err := os.Getwd()
		if err != nil {
			return err
		}
		logrus.Info("fork path: " + fp)
		if err := isGitRepo(forkPath); err != nil {
			return err
		}
		forkPath = fp
	}
	baseBranch := c.String("base-branch")
	if baseBranch == "" {
		bb, err := currentBranch(forkPath)
		if err != nil {
			return err
		}
		baseBranch = bb
		logrus.Info("base branch: " + baseBranch)
	}
	newBranch := c.String("new-charts-branch")
	logrus.Info("new Branch: " + newBranch)
	githubUser := c.String("github-user")
	githubToken := c.String("github-token")
	dryRun := c.Bool("dry-run")
	logrus.Info("dry run: " + strconv.FormatBool(dryRun))
	createPR := c.Bool("create-pr")
	logrus.Info("create PR: " + strconv.FormatBool(createPR))
	if createPR {
		if githubUser == "" {
			gu, err := gitRepoOwner(forkPath)
			if err != nil {
				return err
			}
			githubUser = gu
			logrus.Info("github username: ", githubUser)
		}
		if githubToken == "" {
			return errors.New("'create-pr' requires the 'GITHUB_TOKEN' env var")
		}
	}

	return rancher.SetChartBranchReferences(
		context.Background(),
		forkPath,
		baseBranch,
		newBranch,
		githubUser,
		githubToken,
		createPR,
		dryRun,
	)
}
