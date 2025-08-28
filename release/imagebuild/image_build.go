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

// SyncImageBuild checks the releases of upstream repository (owner, repo)
// with the given imageBuildRepo, and creates the missing latest tags from upstream.
func SyncImageBuild(ctx context.Context, client *github.Client, imageBuildOwner, imageBuildRepo, upstreamOwner, upstreamRepo, tagPrefix string, dryrun bool) error {
	opts := &github.ListOptions{
		PerPage: 100,
	}

	upstreamReleases, _, err := client.Repositories.ListReleases(ctx, upstreamOwner, upstreamRepo, opts)
	if err != nil {
		return fmt.Errorf("failed to retrieve '%s/%s' releases: %v", upstreamOwner, upstreamRepo, err)
	}

	if len(upstreamReleases) == 0 {
		return fmt.Errorf("retrieved list of releases is empty for '%s/%s'", upstreamOwner, upstreamRepo)
	}

	imageBuildReleases, _, err := client.Repositories.ListReleases(ctx, imageBuildOwner, imageBuildRepo, opts)
	if err != nil {
		return fmt.Errorf("failed to retrieve '%s/%s' releases: %v", imageBuildOwner, imageBuildRepo, err)
	}

	imageBuildTags := make(map[string]struct{})
	for _, release := range imageBuildReleases {
		// removes suffixes to build a map to check
		// the existence of tags in image-build repo
		tag, _, _ := strings.Cut(release.GetTagName(), "-")
		imageBuildTags[tag] = struct{}{}
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

		if _, found := imageBuildTags[upstreamTag]; found {
			continue
		}

		fmt.Printf("'%s/%s' tag '%s' not found in 'rancher/%s'.\n", upstreamOwner, upstreamRepo, upstreamTag, imageBuildRepo)

		imageBuildTag := upstreamTag

		// for image-build-kubernetes repo, there's a -rker1 suffix for new k8s releases.
		if imageBuildRepo == imageBuildK8s {
			imageBuildTag += "-rke2r1"
		}

		// specifically for image-build-base the only suffix is the build number, as
		// this automation only detects new releases we can hardcode it to 'b1'.
		if imageBuildRepo == imageBuildBase {
			imageBuildTag += "b1"
		} else {
			now := time.Now()
			imageBuildTag += fmt.Sprintf("-build%d%02d%02d", now.Year(), now.Month(), now.Day())
		}

		imageBuildRelease := &github.RepositoryRelease{
			TagName:         github.String(imageBuildTag),
			TargetCommitish: github.String("master"),
			Name:            github.String(imageBuildTag),
			Draft:           github.Bool(false),
		}

		if dryrun {
			fmt.Printf("Dry run, skipping tag '%s' creation for '%s/%s'\n", imageBuildTag, imageBuildOwner, imageBuildRepo)
			continue
		}
		if _, _, err := client.Repositories.CreateRelease(ctx, imageBuildOwner, imageBuildRepo, imageBuildRelease); err != nil {
			return fmt.Errorf("failed to create '%s/%s' release '%s': %v", imageBuildOwner, imageBuildRepo, imageBuildTag, err)
		}

		fmt.Printf("Successfully created '%s/%s' release '%s'\n", imageBuildOwner, imageBuildRepo, imageBuildTag)
	}
	return nil
}
