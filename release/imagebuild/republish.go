package imagebuild

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/go-github/v90/github"
)

func Republish(ctx context.Context, client *github.Client, owner, repo, targetCommitish string, dryrun bool) error {
	slog.Info("retrieving latest release", slog.String("owner", owner), slog.String("repo", repo))

	release, _, err := client.Repositories.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		return fmt.Errorf("failed to retrieve latest release of '%s/%s': %w", owner, repo, err)
	}

	if release == nil {
		return fmt.Errorf("failed to retrieve latest release, client call returned nil for '%s/%s'", owner, repo)
	}

	// Remove the build suffix (e.g. -buildYYYYMMDD) while preserving any
	// "-k3sN" prerelease suffix. strings.Cut at "-build" leaves everything
	// before the suffix intact: "v3.6.7-k3s1-build20260415" → "v3.6.7-k3s1".
	tag, _, _ := strings.Cut(release.GetTagName(), "-build")

	now := time.Now()
	tag += fmt.Sprintf("-build%d%02d%02d", now.Year(), now.Month(), now.Day())

	newReleaseOpts := github.CreateReleaseRequest{
		TagName:         tag,
		TargetCommitish: new(targetCommitish),
		Name:            new(tag),
		Draft:           new(false),
	}

	if dryrun {
		slog.Info("dry run, skipping tag creation", slog.String("tag", tag), slog.String("owner", owner), slog.String("repo", repo))
		return nil
	}

	newRelease, _, err := client.Repositories.CreateRelease(ctx, owner, repo, newReleaseOpts)
	if err != nil {
		return fmt.Errorf("failed to create '%s/%s' release '%s': %v", owner, repo, tag, err)
	}

	slog.Info("successfully created release", slog.String("owner", owner), slog.String("repo", repo), slog.String("tag", tag), slog.String("release-url", newRelease.GetHTMLURL()))

	return nil
}
