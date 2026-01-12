package imagebuild

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v81/github"
	"github.com/sirupsen/logrus"
)

func Republish(ctx context.Context, client *github.Client, owner, repo, targetCommitish string, dryrun bool) error {
	logrus.Infof("Retrieving latest release of '%s/%s'...", owner, repo)

	release, _, err := client.Repositories.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		return fmt.Errorf("failed to retrieve latest release of '%s/%s': %w", owner, repo, err)
	}

	if release == nil {
		return fmt.Errorf("failed to retrieve latest release, client call returned nil for '%s/%s'", owner, repo)
	}

	// removes the build suffix ( -buildYYYYMMDD )
	tag, _, _ := strings.Cut(release.GetTagName(), "-")

	now := time.Now()
	tag += fmt.Sprintf("-build%d%02d%02d", now.Year(), now.Month(), now.Day())

	newReleaseOpts := &github.RepositoryRelease{
		TagName:         github.Ptr(tag),
		TargetCommitish: github.Ptr(targetCommitish),
		Name:            github.Ptr(tag),
		Draft:           github.Ptr(false),
	}

	if dryrun {
		logrus.Infof("Dry run, skipping tag '%s' creation for '%s/%s'", tag, owner, repo)
		return nil
	}

	newRelease, _, err := client.Repositories.CreateRelease(ctx, owner, repo, newReleaseOpts)
	if err != nil {
		return fmt.Errorf("failed to create '%s/%s' release '%s': %v", owner, repo, tag, err)
	}

	logrus.Infof("Successfully created '%s/%s' release '%s': %s", owner, repo, tag, newRelease.GetHTMLURL())

	return nil
}
