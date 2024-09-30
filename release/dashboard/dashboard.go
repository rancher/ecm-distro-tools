package dashboard

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/go-github/v39/github"
	ecmConfig "github.com/rancher/ecm-distro-tools/cmd/release/config"
	"github.com/rancher/ecm-distro-tools/release"
	"github.com/rancher/ecm-distro-tools/repository"
	"golang.org/x/mod/semver"
)

const (
	dashboardOrg           = "rancher"
	dashboardRepo          = "dashboard"
	dashboardImagesBaseURL = "https://github.com/" + dashboardOrg + "/" + dashboardRepo + "/releases"
)

func CreateRelease(ctx context.Context, client *github.Client, r *ecmConfig.DashboardRelease, opts *repository.CreateReleaseOpts, rc bool, releaseType string) error {
	fmt.Println("validating tag")
	if !semver.IsValid(opts.Tag) {
		return errors.New("tag isn't a valid semver: " + opts.Tag)
	}

	if r.DashboardRepoName != "" {
		opts.Repo = r.DashboardRepoName
	}

	latestRC, err := release.LatestRC(ctx, opts.Owner, opts.Repo, opts.Tag, opts.Tag, client)
	if err != nil {
		return err
	}

	if rc {
		latestRCNumber := 1
		if latestRC != nil {
			// v2.9.0-rcN
			_, trimmedRCNumber, found := strings.Cut(*latestRC, "-rc")
			if !found {
				return errors.New("failed to parse rc number from " + *latestRC)
			}
			currentRCNumber, err := strconv.Atoi(trimmedRCNumber)
			if err != nil {
				return err
			}
			latestRCNumber = currentRCNumber + 1
		} else {
			// this means it would be the first RC tag
			latestRC = new(string)
			*latestRC = opts.Tag + "-rc1"
		}
		opts.Tag = fmt.Sprintf("%s-%s%d", opts.Tag, releaseType, latestRCNumber)
	}

	opts.Name = opts.Tag
	opts.Prerelease = true
	opts.Draft = !rc
	opts.ReleaseNotes = ""

	if !rc {
		fmt.Printf("release.GenReleaseNotes(ctx, %s, %s, %s, %s, client)", opts.Owner, opts.Repo, opts.Branch, r.PreviousTag)
		buff, err := release.GenReleaseNotes(ctx, opts.Owner, opts.Repo, opts.Branch, r.PreviousTag, client)
		if err != nil {
			return err
		}
		opts.ReleaseNotes = buff.String()
	}

	fmt.Printf("create release options: %+v\n", *opts)

	if r.DryRun {
		fmt.Println("dry run, skipping creating release")
		return nil
	}

	createdRelease, err := repository.CreateRelease(ctx, client, opts)
	if err != nil {
		return err
	}

	fmt.Println("release created: " + *createdRelease.HTMLURL)
	return nil
}
