package main

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/rancher/ecm-distro-tools/exec"
	"github.com/rancher/ecm-distro-tools/release/rancher"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func setKDMBranchReferencesCommand() *cli.Command {
	return &cli.Command{
		Name:  "set-kdm-branch-refs",
		Usage: "set kdm branch references in files",
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
				Name:     "new-kdm-branch",
				Aliases:  []string{"n"},
				Usage:    "kdm branch to be replaced in the repo",
				Required: true,
			},
			&cli.BoolFlag{
				Name:    "create-pr",
				Aliases: []string{"p"},
				Usage:   "if true, a PR will be created from your fork to the rancher repo base branch and a variable 'GITHUB_TOKEN' must be exported",
			},
			&cli.StringFlag{
				Name:     "fork-owner",
				Aliases:  []string{"o"},
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
		Action: setKDMBranchReferences,
	}
}

func setKDMBranchReferences(c *cli.Context) error {
	var err error
	forkPath := c.String("fork-path")
	if forkPath == "" {
		forkPath, err = cmdPath()
		if err != nil {
			return err
		}
		logrus.Info("fork path: " + forkPath)
		if err := isGitRepo(forkPath); err != nil {
			return err
		}
	}
	baseBranch := c.String("base-branch")
	if baseBranch == "" {
		baseBranch, err = currentBranch(forkPath)
		if err != nil {
			return err
		}
		logrus.Info("base branch: " + baseBranch)
	}
	newKDMBranch := c.String("new-kdm-branch")
	logrus.Info("new KDM Branch: " + newKDMBranch)
	forkOwner := c.String("fork-owner")
	githubToken := c.String("github-token")
	dryRun := c.Bool("dry-run")
	logrus.Info("dry run: " + strconv.FormatBool(dryRun))
	createPR := c.Bool("create-pr")
	logrus.Info("create PR: " + strconv.FormatBool(createPR))
	if createPR {
		if forkOwner == "" {
			if forkOwner, err = gitRepoOwner(forkPath); err != nil {
				return err
			}
			logrus.Info("fork owner: ", forkOwner)
		}
		if githubToken == "" {
			return errors.New("'create-pr' requires the 'GITHUB_TOKEN' env var")
		}
	}
	return rancher.SetKDMBranchReferences(context.Background(), forkPath, baseBranch, newKDMBranch, forkOwner, githubToken, createPR, dryRun)
}

func cmdPath() (string, error) {
	path, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return path, nil
}

func gitRepoOwner(path string) (string, error) {
	originURL, err := gitOriginURL(path)
	if err != nil {
		return "", err
	}
	if strings.Contains(originURL, "https") {
		return repoOwnerHTTPS(originURL), nil
	}
	return repoOwnerSSH(originURL), nil
}

func currentBranch(path string) (string, error) {
	result, err := exec.RunCommand(path, "git", "branch", "--show-current")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result), nil
}

func gitOriginURL(path string) (string, error) {
	result, err := exec.RunCommand(path, "git", "ls-remote", "--get-url", "origin")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result), nil
}

func repoOwnerHTTPS(URL string) string {
	// https://github.com/rancher/rancher.git
	ownerAndRepo := strings.Split(URL, ".com/")[1]
	owner := strings.Split(ownerAndRepo, "/")[0]
	return owner
}

func repoOwnerSSH(URL string) string {
	// git@github.com:rancher/rancher.git
	ownerAndRepo := strings.Split(URL, ":")[1]
	owner := strings.Split(ownerAndRepo, "/")[0]
	return owner
}

func isGitRepo(path string) error {
	result, err := exec.RunCommand(path, "git", "rev-parse", "--is-inside-work-tree")
	if err != nil {
		return err
	}
	if strings.TrimSpace(result) != "true" {
		return errors.New("path is not a git directory")
	}
	return nil
}
