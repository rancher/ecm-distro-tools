package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
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

var repoToOwner = map[string]string{
	"rke2":    "rancher",
	"rancher": "rancher",
	"k3s":     "k3s-io",
}

type monthly struct {
	count    int
	captains []string
	tags     []string
}

type relStats struct {
	count   int
	monthly map[time.Month]monthly
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
		s.Start()

		var total int
		data := make(map[int]relStats)
		captains := make(map[string]int)

		lo := github.ListOptions{
			PerPage: 100,
		}
		for {
			releases, resp, err := client.Repositories.ListReleases(ctx, repoToOwner[*repo], *repo, &lo)
			if err != nil {
				return err
			}

			for _, release := range releases {
				releaseDate := release.GetCreatedAt().Time
				if releaseDate.After(from) && (releaseDate.Before(to) || releaseDate.Equal(to)) {
					total++

					if _, ok := data[int(release.CreatedAt.Year())]; !ok {
						data[int(release.CreatedAt.Year())] = relStats{
							count: 1,
							monthly: map[time.Month]monthly{
								release.CreatedAt.Month(): {
									count: 1,
									captains: []string{
										*release.Author.Login,
									},
									tags: []string{
										*release.Name,
									},
								},
							},
						}
						continue
					}

					rs := data[int(release.CreatedAt.Year())]
					rs.count++

					mon := rs.monthly[release.CreatedAt.Month()]
					mon.count++
					mon.captains = append(mon.captains, *release.Author.Login)
					mon.tags = append(mon.tags, *release.Name)

					rs.monthly[release.CreatedAt.Month()] = mon

					data[int(release.CreatedAt.Year())] = rs

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

		s.Stop()

		for year := range data {
			fmt.Printf("\n%d:\n", year)

			months := make([]int, 0, len(data[year].monthly))
			for k := range data[year].monthly {
				months = append(months, int(k))
			}
			sort.Ints(months)

			for _, m := range months {
				mon := time.Month(m)
				tmp := data[year].monthly[mon]
				tmp.captains = dedup(tmp.captains)
				data[year].monthly[mon] = tmp
				captains := strings.Join(data[year].monthly[mon].captains, ", ")
				tags := strings.Join(data[year].monthly[mon].tags, ", ")
				fmt.Printf("  %-9s\n    Count: %3d\n    Captains: %s\n    Tags: %s\n",
					mon, data[year].monthly[mon].count, captains, tags)
			}
		}

		fmt.Printf("\nTotal: %d\n", total)

		return nil
	},
}

func dedup(slice []string) []string {
	seen := make(map[string]struct{})
	result := []string{}

	for _, val := range slice {
		if _, ok := seen[val]; !ok {
			seen[val] = struct{}{}
			result = append(result, val)
		}
	}

	return result
}

func init() {
	rootCmd.AddCommand(statsCmd)

	repo = statsCmd.Flags().StringP("repo", "r", "", "repository")
	startDate = statsCmd.Flags().StringP("start", "s", "", "start date")
	endDate = statsCmd.Flags().StringP("end", "e", "", "end date")

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
