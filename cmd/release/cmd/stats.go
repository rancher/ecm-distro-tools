package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/rancher/ecm-distro-tools/release"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var (
	repo      *string
	startDate *string
	endDate   *string
	format    *string
)

var repoToOwner = map[string]string{
	"rke2":    "rancher",
	"rancher": "rancher",
	"k3s":     "k3s-io",
}

// statsCmd represents the stats command
var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Release statistics",
	Long:  `Retrieve release statistics for a time period.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		from, err := time.Parse(time.DateOnly, *startDate)
		if err != nil {
			return err
		}

		to, err := time.Parse(time.DateOnly, *endDate)
		if err != nil {
			return err
		}

		if to.Before(from) {
			return errors.New("end date before start date")
		}

		githubToken := os.Getenv("GITHUB_TOKEN")

		ctx := context.Background()
		client := repository.NewGithub(ctx, githubToken)

		s := spinner.New(spinner.CharSets[31], 100*time.Millisecond)
		s.HideCursor = true
		s.Writer = os.Stderr
		s.Start()

		sd, err := release.Stats(ctx, client, from, to, repoToOwner[*repo], *repo)
		if err != nil {
			return err
		}

		if format == nil || *format == "" {
			*format = "json"
		}

		var b []byte

		switch *format {
		case "json":
			b, err = json.Marshal(sd)
		case "yaml":
			b, err = yaml.Marshal(sd)
		default:
			return errors.New("unrecognized format")
		}
		if err != nil {
			return err
		}

		s.Stop()

		fmt.Println(string(b))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statsCmd)

	repo = statsCmd.Flags().StringP("repo", "r", "", "repository")
	startDate = statsCmd.Flags().StringP("start", "s", "", "start date")
	endDate = statsCmd.Flags().StringP("end", "e", "", "end date")
	format = statsCmd.Flags().StringP("format", "f", "", "format (json|yaml). default: json")

	if err := statsCmd.MarkFlagRequired("repo"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if err := statsCmd.MarkFlagRequired("start"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if err := statsCmd.MarkFlagRequired("end"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
