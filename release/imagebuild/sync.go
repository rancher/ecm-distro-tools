// Package imagebuild handles tags of image-build repos
package imagebuild

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/google/go-github/v82/github"
	"github.com/sirupsen/logrus"
)

const (
	imageBuildK8s  = "image-build-kubernetes"
	imageBuildBase = "image-build-base"
)

// Define the cutoff time: 2 days ago
var cutoff = time.Now().Add(-time.Hour * 24 * 2)

// Sync checks the releases of upstream repository (owner, repo)
// with the given repo, and creates the missing latest tags from upstream.
func Sync(ctx context.Context, client *github.Client, owner, repo, upstreamOwner, upstreamRepo, tagPrefix string, dryrun bool) error {
	logrus.Infof("Retrieving all upstream tags for '%s/%s'...", upstreamOwner, upstreamRepo)

	// This slice will hold all tags gathered from all pages.
	var upstreamTags []*github.RepositoryTag
	opt := &github.ListOptions{PerPage: 100}

	for {
		tagsPage, resp, err := client.Repositories.ListTags(ctx, upstreamOwner, upstreamRepo, opt)
		if err != nil {
			return fmt.Errorf("failed to retrieve page %d of '%s/%s' tags: %w", opt.Page, upstreamOwner, upstreamRepo, err)
		}

		upstreamTags = append(upstreamTags, tagsPage...)

		// If NextPage is 0 there's no more tags to retrieve so we can break out of the loop
		if resp.NextPage == 0 {
			break
		}

		// set the page for the next iteration.
		opt.Page = resp.NextPage
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
		upstreamTagName := upstreamTag.GetName()

		// skip if the current upstream tag name isn't valid.
		if !validateTagFormat(upstreamTagName, tagPrefix) {
			logrus.Infof("'%s/%s' tag '%s' is not in expected format, skipping release.", upstreamOwner, upstreamRepo, upstreamTagName)
			continue
		}

		isOlder, err := isTagOlderThanCutoff(ctx, client, upstreamOwner, upstreamRepo, upstreamTagName, cutoff)
		if err != nil {
			logrus.Warnf("Could not determine age of upstream tag '%s', skipping: %v", upstreamTagName, err)
			continue
		}

		// if the tag is older than the defined cutoff time
		if isOlder {
			logrus.Infof("'%s/%s' tag '%s' is older than 2 days, skipping release.", upstreamOwner, upstreamRepo, upstreamTagName)
			continue
		}
		// if the release is older than a couple of day it can be ignored
		if tagPrefix != "" {
			if !strings.HasPrefix(upstreamTagName, tagPrefix) {
				continue
			}
			upstreamTagName = strings.TrimPrefix(upstreamTagName, tagPrefix)
		}

		// skip current upstream release if not GA
		if strings.Contains(upstreamTagName, "rc") || strings.Contains(upstreamTagName, "alpha") || strings.Contains(upstreamTagName, "beta") || strings.Contains(upstreamTagName, "dev") {
			continue
		}

		if _, found := tagsMap[upstreamTagName]; found {
			logrus.Infof("'%s/%s' tag '%s' found in '%s/%s', skipping release.", upstreamOwner, upstreamRepo, upstreamTagName, owner, repo)
			continue
		}

		logrus.Infof("'%s/%s' tag '%s' not found in 'rancher/%s'.", upstreamOwner, upstreamRepo, upstreamTagName, repo)

		imageBuildTag := upstreamTagName

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
			TagName:         github.Ptr(imageBuildTag),
			TargetCommitish: github.Ptr("master"),
			Name:            github.Ptr(imageBuildTag),
			Draft:           github.Ptr(false),
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

// isTagOlderThanCutoff checks if a given tag in a repository was created before the cutoff time.
// It returns true if the tag is older, and false otherwise; also handles both annotated tags (using the tagger date) and lightweight tags (using the committer date).
func isTagOlderThanCutoff(ctx context.Context, client *github.Client, owner, repo, tagName string, cutoff time.Time) (bool, error) {
	var tagDate time.Time

	ref, _, err := client.Git.GetRef(ctx, owner, repo, "tags/"+tagName)
	if err != nil {
		return false, fmt.Errorf("could not get ref for tag '%s': %w", tagName, err)
	}

	// Determine the date based on the object type returned by the ref.
	switch ref.Object.GetType() {
	case "tag": // This is an annotated tag.
		tagObject, _, err := client.Git.GetTag(ctx, owner, repo, ref.Object.GetSHA())
		if err != nil {
			return false, fmt.Errorf("could not get annotated tag object for '%s': %w", tagName, err)
		}
		tagDate = tagObject.Tagger.GetDate().Time

	case "commit": // This is a lightweight tag that points directly to a commit.
		commit, _, err := client.Git.GetCommit(ctx, owner, repo, ref.Object.GetSHA())
		if err != nil {
			return false, fmt.Errorf("could not get commit object for '%s': %w", tagName, err)
		}
		tagDate = commit.Committer.GetDate().Time

	default:
		return false, errors.New("unknown object type '" + ref.Object.GetType() + "' for tag '" + tagName + "'")
	}

	// Compare the fetched date with the cutoff and return the result.
	return tagDate.Before(cutoff), nil
}

// validateTagFormat checks if a provided tag respects the expected format.
// If tagPrefix is provided, the tagName must start with it.
// If tagPrefix is empty, the tagName must not have any other prefix (besides the optional 'v').
// It returns a boolean indicating if the format is valid.
func validateTagFormat(tagName, tagPrefix string) bool {
	var versionStr string

	// Case 1: A specific prefix is required.
	if tagPrefix != "" {
		if !strings.HasPrefix(tagName, tagPrefix) {
			return false
		}
		versionStr = strings.TrimPrefix(tagName, tagPrefix)
	} else {
		// Case 2: No prefix is allowed, the tag itself must be the version.
		versionStr = tagName
	}

	// semver library to validate the version string, if it contains a prefix (besides 'v' it fails).
	v, err := semver.NewVersion(versionStr)
	if err != nil {
		// If parsing fails, it's not a valid semantic version.
		return false
	}

	// this checks for any suffix that a tag may have, and we ignore those with suffixes:
	// example:
	// - v3.21.1 <------------ correct
	// - v3.21.1-typha <------ will be skipped
	// - v3.21.1-pod2daemon <- will be skipped
	// - v3.24.2-0.dev <------ will be skipped
	if v.Prerelease() != "" {
		return false
	}

	return true
}
