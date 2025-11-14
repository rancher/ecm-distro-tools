package dashboard

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/go-github/v78/github"
	"github.com/rancher/ecm-distro-tools/release"
	"github.com/rancher/ecm-distro-tools/repository"
	"golang.org/x/mod/semver"
)

const (
	dashboardOrg           = "rancher"
	dashboardRepo          = "dashboard"
	uiOrg                  = "rancher"
	uiRepo                 = "ui"
	dashboardImagesBaseURL = "https://github.com/" + dashboardOrg + "/" + dashboardRepo + "/releases"
)

// CreateRelease will create a new tag and a new release with given params.
func CreateRelease(ctx context.Context, client *github.Client, opts *repository.CreateReleaseOpts, rc, dryRun bool, releaseType, previousTag string) error {
	if !semver.IsValid(opts.Tag) {
		return errors.New("tag isn't a valid semver: " + opts.Tag)
	}

	latestPreRelease, err := release.LatestPreRelease(ctx, client, opts.Owner, opts.Repo, opts.Tag, releaseType)
	if err != nil {
		return err
	}

	if rc {
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

	if !rc {
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

// ReleaseBranchFromTag generates the ui release branch for a release line with the format of 'release-{major}.{minor}'. The generated release branch might not be valid depending on multiple factors that cannot be treated on this function such as it being 'master'.
// Please make sure that this is the expected format before using the generated release branch.
// This format is used by both `dashboard` and `ui` but might change at any time.
func ReleaseBranchFromTag(tag string) (string, error) {
	majorMinor := semver.MajorMinor(tag)

	if majorMinor == "" {
		return "", errors.New("the tag isn't a valid semver: " + tag)
	}

	v, _ := strings.CutPrefix(majorMinor, "v")

	releaseBranch := "release-" + v

	return releaseBranch, nil
}
