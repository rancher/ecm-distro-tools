package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/google/go-github/v39/github"
	"github.com/rancher/ecm-distro-tools/repository"
)

const layout = "2006-01-02"

var repoToOwner = map[string]string{
	"rke2":    "rancher",
	"rancher": "rancher",
	"k3s":     "k3s-io",
}

func main() {
	if len(os.Args) != 4 {
		fmt.Fprint(os.Stderr, "usage "+os.Args[0]+"<repo> <from> <to>\n")
		os.Exit(1)
	}

	repo := os.Args[1]
	from, err := time.Parse(layout, os.Args[2])
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
	to, err := time.Parse(layout, os.Args[3])
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	githubToken := os.Getenv("GITHUB_TOKEN")

	ctx := context.Background()
	client := repository.NewGithub(ctx, githubToken)

	var allReleases []*github.RepositoryRelease
	monthly := make(map[time.Month]int)
	captains := make(map[string]int)

	lo := github.ListOptions{
		PerPage: 100,
	}
	for {
		releases, resp, err := client.Repositories.ListReleases(ctx, repoToOwner[repo], repo, &lo)
		if err != nil {
			log.Fatal("Error fetching releases:", err)
		}

		for _, release := range releases {
			releaseDate := release.GetCreatedAt().Time
			if releaseDate.After(from) && releaseDate.Before(to) {
				allReleases = append(allReleases, release)

				if _, ok := monthly[releaseDate.Month()]; !ok {
					monthly[releaseDate.Month()]++
					continue
				}
				monthly[releaseDate.Month()]++

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

	fmt.Printf("%s: %d\n", repo, len(allReleases))
	for _, month := range months {
		fmt.Printf("  %-10s %2d\n", time.Month(month).String(), monthly[time.Month(month)])

	}

	fmt.Printf("\nCaptain Counts: %d\n", len(captains))
	for captain, count := range captains {
		fmt.Printf("  %-15s %2d\n", captain, count)
	}

	os.Exit(0)
}
