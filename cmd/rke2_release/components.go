package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/urfave/cli/v2"
)

func componentsCommand() *cli.Command {
	return &cli.Command{
		Name:  "components",
		Usage: "interact with the RKE2 components",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "github-token",
				Aliases:  []string{"g"},
				EnvVars:  []string{"GITHUB_TOKEN"},
				Required: true,
			},
			&cli.StringFlag{
				Name:     "list",
				Aliases:  []string{"l"},
				Required: false,
			},
		},
		Action: components,
	}
}

func components(c *cli.Context) error {
	token := c.String("github-token")
	if token == "" {
		return errors.New("env var GITHUB_TOKEN is required")
	}

	listContent := c.String("list")

	ctx := context.Background()
	ghClient := repository.NewGithub(ctx, token)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 0, ' ', tabwriter.Debug)

	if listContent == "all" {
		for _, repo := range repository.RKE2HardenedImages {
			owner, repo, err := repository.SplitOwnerRepo(repo)
			if err != nil {
				return err
			}

			tag, err := repository.LatestTag(ctx, ghClient, owner, repo)
			if err != nil {
				return err
			}

			fmt.Fprintf(w, "%s\t%+v\n", repo, *tag.Name)
		}
	} else {
		owner, repo, err := repository.SplitOwnerRepo(listContent)
		if err != nil {
			return err
		}

		tag, err := repository.LatestTag(ctx, ghClient, owner, repo)
		if err != nil {
			return err
		}

		fmt.Fprintf(w, "%s\t%+v\n", repo, *tag.Name)
	}

	w.Flush()

	return nil
}
