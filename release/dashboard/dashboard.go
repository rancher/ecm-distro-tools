// Package dashboard holds tools to release dashboard and ui
package dashboard

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/go-github/v90/github"
	"github.com/rancher/ecm-distro-tools/repository"
	"golang.org/x/mod/semver"
)

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

func CreateTag(ctx context.Context, ghClient *github.Client, owner, repo, baseTag, sha, branch, releaseType string, preRelease, dryRun bool) (string, string, error) {
	if !semver.IsValid(baseTag) {
		return "", "", errors.New("the base tag is invalid: " + baseTag)
	}

	if sha == "" {
		latestCommit, err := repository.BranchLatestCommitSHA(ctx, ghClient, owner, repo, branch)
		if err != nil {
			return "", "", err
		}
		sha = latestCommit
	}

	tag := baseTag
	if preRelease {
		preReleaseTag, err := repository.LatestPreReleaseTag(ctx, ghClient, owner, repo, releaseType, baseTag)
		if err != nil {
			return "", "", err
		}
		tag = preReleaseTag
	}

	if dryRun {
		fmt.Println("dry run, skipping creating tag")
		return tag, sha, nil
	}
	return repository.CreateTag(ctx, ghClient, owner, repo, tag, sha)
}
