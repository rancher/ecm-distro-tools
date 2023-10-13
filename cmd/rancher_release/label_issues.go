package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/Masterminds/semver/v3"
	"github.com/google/go-github/v39/github"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/urfave/cli/v2"
)

func labelIssuesCommand() *cli.Command {
	return &cli.Command{
		Name:  "label-issues",
		Usage: "relabels 'Waiting for RC' issues with 'To Test' and adds a comment",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "github-token",
				Aliases:  []string{"g"},
				Usage:    "github token",
				EnvVars:  []string{"GITHUB_TOKEN"},
				Required: false,
			},
			&cli.StringFlag{
				Name:     "tag",
				Aliases:  []string{"t"},
				Usage:    "release candidate tag",
				Required: true,
			},
			&cli.BoolFlag{
				Name:     "dry-run",
				Aliases:  []string{"r"},
				Usage:    "list matching issues and new labels without updating",
				Required: false,
			},
		},
		Action: labelIssuesWaitingForRC,
	}
}

// labelIssuesWaitingForRC updates issues with the "Waiting for RC" label
// to "To Test" if the issue's milestone matches the tag, and creates a comment
func labelIssuesWaitingForRC(c *cli.Context) error {
	ctx := context.Background()
	tag := c.String("tag")
	dryRun := c.Bool("dry-run")
	githubToken := c.String("github-token")
	repo := "rancher"
	org := "rancher"
	oldTag := "[zube]: Waiting for RC"
	newTag := "[zube]: To Test"

	if tag == "" {
		return errors.New("'tag' must be set")
	}
	version, err := semver.NewVersion(tag)
	if err != nil {
		return err
	}
	if version.Prerelease() == "" {
		return errors.New("'tag' must be a prerelease")
	}

	if githubToken == "" {
		return errors.New("'GITHUB_TOKEN' environment variable must be set")
	}
	client := repository.NewGithub(ctx, githubToken)
	opts := &github.IssueListByRepoOptions{
		State:  "open",
		Labels: []string{oldTag},
	}
	issues := make([]*github.Issue, 0)
	for {

		ghIssues, resp, err := client.Issues.ListByRepo(ctx, org, repo, opts)
		if err != nil {
			return err
		}

		for _, issue := range ghIssues {
			if issue.Milestone == nil {
				continue
			}
			pattern, err := semver.NewConstraint(*issue.Milestone.Title)
			if err != nil {
				return err
			}
			if pattern.Check(version) {
				issues = append(issues, issue)
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.ListOptions.Page = resp.NextPage
	}

	if dryRun {
		if len(issues) == 1 {
			fmt.Printf("Updating %d issue\n", len(issues))
		} else {
			fmt.Printf("Updating %d issues\n", len(issues))
		}
	}

	for _, issue := range issues {
		if dryRun {
			fmt.Printf("#%d %s (%s)\n  [%s] -> [%s] \n", *issue.Number, *issue.Title, *issue.Milestone.Title, oldTag, newTag)
			continue
		}
		labels := make([]string, 0, len(issue.Labels))
		for _, label := range issue.Labels {
			if label.GetName() != oldTag {
				labels = append(labels, label.GetName())
			}
		}
		labels = append(labels, newTag)
		_, _, err = client.Issues.Edit(ctx, org, repo, *issue.Number, &github.IssueRequest{
			Labels: &labels,
		})
		if err != nil {
			return err
		}
		_, _, err = client.Issues.CreateComment(ctx, org, repo, *issue.Number, &github.IssueComment{
			Body: github.String("Available to test on " + tag),
		})
		if err != nil {
			return err
		}
	}

	return nil
}
