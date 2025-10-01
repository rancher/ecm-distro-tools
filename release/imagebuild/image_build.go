package imagebuild

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v39/github"
	"github.com/sirupsen/logrus"
)

const (
	imageBuildK8s  = "image-build-kubernetes"
	imageBuildBase = "image-build-base"
)

// Sync checks the releases of upstream repository (owner, repo)
// with the given repo, and creates the missing latest tags from upstream.
func Sync(ctx context.Context, client *github.Client, owner, repo, upstreamOwner, upstreamRepo, tagPrefix string, dryrun bool) error {
	// retrieve the last 150 upstream releases
	upstreamTags, _, err := client.Repositories.ListTags(ctx, upstreamOwner, upstreamRepo, &github.ListOptions{PerPage: 150})
	if err != nil {
		return fmt.Errorf("failed to retrieve '%s/%s' tags: %v", upstreamOwner, upstreamRepo, err)
	}

	if len(upstreamTags) == 0 {
		return fmt.Errorf("retrieved list of tags is empty for '%s/%s'", upstreamOwner, upstreamRepo)
	}

	// retrieve the last 300 image build releases
	tags, _, err := client.Repositories.ListTags(ctx, owner, repo, &github.ListOptions{PerPage: 300})
	if err != nil {
		return fmt.Errorf("failed to retrieve '%s/%s' releases: %v", owner, repo, err)
	}

	tagsMap := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		// removes any suffixes (e.g. -buildYYYYMMDD) to build a map to check
		// the existence of tags in image-build repo
		tag, _, _ := strings.Cut(tag.GetName(), "-")
		tagsMap[tag] = struct{}{}
	}

	for _, upstreamTag := range upstreamTags {
		upstreamTag := upstreamTag.GetName()
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

		if _, found := tagsMap[upstreamTag]; found {
			logrus.Infof("'%s/%s' tag '%s' found in '%s/%s', skipping release.", upstreamOwner, upstreamRepo, upstreamTag, owner, repo)
			continue
		}

		logrus.Infof("'%s/%s' tag '%s' not found in 'rancher/%s'.", upstreamOwner, upstreamRepo, upstreamTag, repo)

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
			logrus.Infof("Dry run, skipping tag '%s' creation for '%s/%s'", imageBuildTag, owner, repo)
			continue
		}
		if _, _, err := client.Repositories.CreateRelease(ctx, owner, repo, newRelease); err != nil {
			return fmt.Errorf("failed to create '%s/%s' release '%s': %v", owner, repo, imageBuildTag, err)
		}

		logrus.Infof("Successfully created '%s/%s' release '%s'", owner, repo, imageBuildTag)
	}
	return nil
}
