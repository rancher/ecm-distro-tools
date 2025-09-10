package imagebuild

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v39/github"
)

const (
	imageBuildK8s  = "image-build-kubernetes"
	imageBuildBase = "image-build-base"
)

// Sync checks the releases of upstream repository (owner, repo)
// with the given repo, and creates the missing latest tags from upstream.
func Sync(ctx context.Context, client *github.Client, owner, repo, upstreamOwner, upstreamRepo, tagPrefix string, dryrun bool) error {
	// retrieve the last 10 upstream releases
	upstreamReleases, _, err := client.Repositories.ListReleases(ctx, upstreamOwner, upstreamRepo, &github.ListOptions{PerPage: 10})
	if err != nil {
		return fmt.Errorf("failed to retrieve '%s/%s' releases: %v", upstreamOwner, upstreamRepo, err)
	}

	if len(upstreamReleases) == 0 {
		return fmt.Errorf("retrieved list of releases is empty for '%s/%s'", upstreamOwner, upstreamRepo)
	}

	// retrieve the last 100 image build releases
	releases, _, err := client.Repositories.ListReleases(ctx, owner, repo, &github.ListOptions{PerPage: 100})
	if err != nil {
		return fmt.Errorf("failed to retrieve '%s/%s' releases: %v", owner, repo, err)
	}

	tags := make(map[string]struct{}, len(releases))
	for _, release := range releases {
		// removes suffixes to build a map to check
		// the existence of tags in image-build repo
		tag, _, _ := strings.Cut(release.GetTagName(), "-")
		tags[tag] = struct{}{}
	}

	for _, upstreamRelease := range upstreamReleases {
		upstreamTag := upstreamRelease.GetTagName()
		if tagPrefix != "" {
			if !strings.HasPrefix(upstreamTag, tagPrefix) {
				continue
			}
			upstreamTag = strings.TrimPrefix(upstreamTag, tagPrefix)
		}

		// skip current upstream release if not GA
		if strings.Contains(upstreamTag, "rc") || strings.Contains(upstreamTag, "alpha") || strings.Contains(upstreamTag, "beta") {
			continue
		}

		if _, found := tags[upstreamTag]; found {
			continue
		}

		fmt.Printf("'%s/%s' tag '%s' not found in 'rancher/%s'.\n", upstreamOwner, upstreamRepo, upstreamTag, repo)

		imageBuildTag := upstreamTag

		// for image-build-kubernetes repo, there's a -rker1 suffix for new k8s releases.
		if repo == imageBuildK8s {
			imageBuildTag += "-rke2r1"
		}

		// specifically for image-build-base the only suffix is the build number, as
		// this automation only detects new releases we can hardcode it to 'b1'.
		if repo == imageBuildBase {
			imageBuildTag += "b1"
		} else {
			now := time.Now()
			imageBuildTag += fmt.Sprintf("-build%d%02d%02d", now.Year(), now.Month(), now.Day())
		}

		newRelease := &github.RepositoryRelease{
			TagName:         github.String(imageBuildTag),
			TargetCommitish: github.String("master"),
			Name:            github.String(imageBuildTag),
			Draft:           github.Bool(false),
		}

		if dryrun {
			fmt.Printf("Dry run, skipping tag '%s' creation for '%s/%s'\n", imageBuildTag, owner, repo)
			continue
		}
		if _, _, err := client.Repositories.CreateRelease(ctx, owner, repo, newRelease); err != nil {
			return fmt.Errorf("failed to create '%s/%s' release '%s': %v", owner, repo, imageBuildTag, err)
		}

		fmt.Printf("Successfully created '%s/%s' release '%s'\n", owner, repo, imageBuildTag)
	}
	return nil
}
