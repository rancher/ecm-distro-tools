package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/briandowns/spinner"
	"github.com/google/go-github/v39/github"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/spf13/cobra"
)

var (
	repo      *string
	startDate *string
	endDate   *string
)

const layout = "2006-01-02"

var repoToOwner = map[string]string{
	"rke2":    "rancher",
	"rancher": "rancher",
	"k3s":     "k3s-io",
}

// statsCmd represents the stats command
var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Release statistics",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		from, err := time.Parse(layout, *startDate)
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
		to, err := time.Parse(layout, *endDate)
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
		githubToken := os.Getenv("GITHUB_TOKEN")

		ctx := context.Background()
		client := repository.NewGithub(ctx, githubToken)

		s := spinner.New(spinner.CharSets[31], 100*time.Millisecond)
		s.HideCursor = true
		s.Start()

		var allReleases []*github.RepositoryRelease
		monthly := make(map[int]int)
		captains := make(map[string]int)

		lo := github.ListOptions{
			PerPage: 100,
		}
		for {
			releases, resp, err := client.Repositories.ListReleases(ctx, repoToOwner[*repo], *repo, &lo)
			if err != nil {
				fmt.Fprint(os.Stderr, err)
				os.Exit(1)
			}

			for _, release := range releases {
				releaseDate := release.GetCreatedAt().Time
				if releaseDate.After(from) && releaseDate.Before(to) {
					allReleases = append(allReleases, release)

					if _, ok := monthly[int(releaseDate.Month())]; !ok {
						monthly[int(releaseDate.Month())]++
						continue
					}
					monthly[int(releaseDate.Month())]++

					if release.Author.Login != nil {
						if _, ok := captains[*release.Author.Login]; !ok {
							captains[*release.Author.Login]++
							continue
						}
						captains[*release.Author.Login]++
					}
				}
			}

			if resp.NextPage == 0 {
				break
			}
			lo.Page = resp.NextPage
		}

		months := make([]int, 0, 12)
		for k := range monthly {
			months = append(months, int(k))
		}
		sort.Ints(months)

		s.Stop()

		fmt.Printf("Total: %d\n", len(allReleases))
		for _, month := range months {
			fmt.Printf("  %-10s %2d\n", time.Month(month).String(), monthly[int(time.Month(month))])

		}

		fmt.Println("\nCaptains:")
		for captain, count := range captains {
			fmt.Printf("  %-15s %2d\n", captain, count)
		}
	},
}

func init() {
	rootCmd.AddCommand(statsCmd)

	repo = statsCmd.Flags().StringP("repo", "r", "", "repository")
	startDate = statsCmd.Flags().StringP("start", "s", "", "start date")
	endDate = statsCmd.Flags().StringP("end", "e", "", "end date")
}
