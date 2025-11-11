package ui

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/go-github/v77/github"
	"github.com/rancher/ecm-distro-tools/release"
	"github.com/rancher/ecm-distro-tools/repository"
	"golang.org/x/mod/semver"
)

const (
	uiOrg           = "rancher"
	uiRepo          = "ui"
	uiImagesBaseURL = "https://github.com/" + uiOrg + "/" + uiRepo + "/releases"
)

// CreateRelease will create a new tag and a new release with given params.
func CreateRelease(ctx context.Context, client *github.Client, opts *repository.CreateReleaseOpts, preRelease, dryRun bool, releaseType, previousTag string) error {
	if !semver.IsValid(opts.Tag) {
		return errors.New("tag isn't a valid semver: " + opts.Tag)
	}

	latestPreRelease, err := release.LatestPreRelease(ctx, client, opts.Owner, opts.Repo, opts.Tag, releaseType)
	if err != nil {
		return err
	}

	if preRelease {
		latestRCNumber := 1
		if latestPreRelease != nil {
			// v2.9.0-rcN / -alphaN
			_, trimmedRCNumber, found := strings.Cut(*latestPreRelease, "-"+releaseType)
			if !found {
				return errors.New("failed to parse rc number from " + *latestPreRelease)
			}
			currentRCNumber, err := strconv.Atoi(trimmedRCNumber)
			if err != nil {
				return err
			}
			latestRCNumber = currentRCNumber + 1
		}
		opts.Tag = fmt.Sprintf("%s-%s%d", opts.Tag, releaseType, latestRCNumber)
	}

	opts.Name = opts.Tag
	opts.Prerelease = true
	opts.ReleaseNotes = ""

	if !preRelease {
		fmt.Printf("release.GenReleaseNotes(ctx, %s, %s, %s, %s, client)", opts.Owner, opts.Repo, opts.Branch, previousTag)
		buff, err := release.GenReleaseNotes(ctx, opts.Owner, opts.Repo, opts.Branch, previousTag, client)
		if err != nil {
			return err
		}
		opts.ReleaseNotes = buff.String()
	}

	fmt.Printf("create release options: %+v\n", *opts)

	if dryRun {
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
