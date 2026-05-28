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
	"github.com/rancher/ecm-distro-tools/release/metrics"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

var (
	repo       *string
	startDate  *string
	endDate    *string
	format     *string
	webhookURL *string
	severity   *string
)

var repoToOwner = map[string]string{
	"rke2":    "rancher",
	"rancher": "rancher",
	"k3s":     "k3s-io",
}

// statsCmd represents the base stats parent command
var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Statistics commands",
	Long:  `Retrieve various statistics including releases and CVEs.`,
}

// releasesStatsCmd represents the release statistics command
var releasesStatsCmd = &cobra.Command{
	Use:   "releases",
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

		ctx := context.Background()
		client := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)

		s := spinner.New(spinner.CharSets[31], 100*time.Millisecond)
		s.HideCursor = true
		s.Writer = os.Stderr
		s.Start()

		sd, err := release.Stats(ctx, client, from, to, repoToOwner[*repo], *repo)
		if err != nil {
			return err
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

var cveStatsSubCmd = &cobra.Command{
	Use:   "cve",
	Short: "CVE statistics command",
	Long:  `Retrieve CVE statistics from current releases.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		ghClient := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)
		reports, err := metrics.CVEsMetrics(ctx, ghClient)
		if err != nil {
			return err
		}

		return reports.CVEsBySeverity(*severity, *webhookURL)
	},
}

func init() {
	rootCmd.AddCommand(statsCmd)

	statsCmd.AddCommand(releasesStatsCmd)
	statsCmd.AddCommand(cveStatsSubCmd)

	repo = releasesStatsCmd.Flags().StringP("repo", "r", "", "repository")
	startDate = releasesStatsCmd.Flags().StringP("start", "s", "", "start date")
	endDate = releasesStatsCmd.Flags().StringP("end", "e", "", "end date")
	format = releasesStatsCmd.Flags().StringP("format", "f", "json", "format (json|yaml)")
	webhookURL = cveStatsSubCmd.Flags().StringP("webhook-url", "u", "", "Slack webhook URL for sending messages")
	severity = cveStatsSubCmd.Flags().StringP("severity", "s", "critical", "severity (critical|high|medium|low)")

	if err := releasesStatsCmd.MarkFlagRequired("repo"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if err := releasesStatsCmd.MarkFlagRequired("start"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if err := releasesStatsCmd.MarkFlagRequired("end"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if err := cveStatsSubCmd.MarkFlagRequired("webhook-url"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
