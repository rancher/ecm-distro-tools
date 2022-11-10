package main

import (
	"context"
	"errors"
	"time"

	"github.com/drone/drone-go/drone"
	"github.com/google/go-github/v39/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/mod/semver"
)

type Pull struct {
	pull    *github.PullRequest
	commits []*github.RepositoryCommit
}

type RKE2Release struct {
	version string
	branch  string
	k8s     *github.RepositoryRelease
	ga      *github.RepositoryRelease
	rcs     []*github.RepositoryRelease
	prs     []*Pull
	builds  []*Build
}

type buildLister interface {
	BuildList(namespace string, name string, opts drone.ListOptions) ([]*drone.Build, error)
}

type repositoriesBrowser interface {
	GetReleaseByTag(ctx context.Context, owner, repo, tag string) (*github.RepositoryRelease, *github.Response, error)
	ListReleases(ctx context.Context, owner, repo string, opts *github.ListOptions) ([]*github.RepositoryRelease, *github.Response, error)
}

type pullRequestsBrowser interface {
	List(ctx context.Context, owner string, repo string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error)
	ListCommits(ctx context.Context, owner string, repo string, number int, opts *github.ListOptions) ([]*github.RepositoryCommit, *github.Response, error)
}

type Client struct {
	Repositories repositoriesBrowser
	PullRequests pullRequestsBrowser
	DronePr      buildLister
	DronePub     buildLister
}

func (c Client) getRKE2Release(ctx context.Context, version, branch string) (RKE2Release, error) {
	rke2Release := RKE2Release{
		version: version,
		branch:  branch,
		rcs:     make([]*github.RepositoryRelease, 0),
		prs:     make([]*Pull, 0),
	}
	if !semver.IsValid(version) {
		return rke2Release, errors.New("invalid version")
	}

	// get K8s patch release
	k8sTag := patch(version)
	k8s, _, err := c.Repositories.GetReleaseByTag(ctx, "kubernetes", "kubernetes", k8sTag)
	if err != nil {
		logrus.Errorln("failed to get Kubernetes release:", err)
		return rke2Release, errors.New("Failed to get Kubernetes release " + k8sTag)
	}
	rke2Release.k8s = k8s

	// get RKE2 GA release
	ga, _, err := c.Repositories.GetReleaseByTag(ctx, "rancher", "rke2", version)
	if err != nil {
		logrus.Errorln("failed to get RKE2 release:", err)
		return rke2Release, errors.New("Failed to get RKE2 release " + version)
	}
	rke2Release.ga = ga

	// Get all rancher/rke2 prereleases
	page := 1
	for {
		list, resp, err := c.Repositories.ListReleases(ctx, "rancher", "rke2", &github.ListOptions{Page: page})
		if err != nil {
			logrus.Errorln("failed to get RKE2 releases:", err)
			return rke2Release, errors.New("Failed to get RKE2 releases " + version)
		}

		for _, release := range list {
			if mainPatch(*release.TagName) == version && *release.Prerelease {
				rke2Release.rcs = append(rke2Release.rcs, release)
			}
		}

		if resp.NextPage == 0 {
			break
		}
		page++
	}

	// Get all rancher/rke2 pull requests merged betweek the K8s release and the RKE2 release
	page = 1
	for {
		opts := &github.PullRequestListOptions{
			ListOptions: github.ListOptions{
				Page: page,
			},
			State:     "closed",
			Base:      branch,
			Sort:      "created",
			Direction: "desc",
		}
		pullRequests, resp, err := c.PullRequests.List(ctx, "rancher", "rke2", opts)
		if err != nil {
			logrus.Errorln("failed to get RKE2 pull requests:", err)
			return rke2Release, err
		}

		// do not continue if all pull requests are older than the release
		cont := false
		for _, pr := range pullRequests {
			if pr.MergedAt == nil {
				continue
			}
			if pr.CreatedAt.After(k8s.PublishedAt.Time) && pr.MergedAt.Before(ga.PublishedAt.Time) {
				rke2Release.prs = append(rke2Release.prs, &Pull{pull: pr})
				cont = true
			}
		}
		if !cont {
			break
		}

		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}

	// get all commits for each pull request
	for _, pr := range rke2Release.prs {
		for {
			commits, resp, err := c.PullRequests.ListCommits(ctx, "rancher", "rke2", *pr.pull.Number, &github.ListOptions{})
			if err != nil {
				logrus.Errorln("failed to get RKE2 pull request commits:", err)
				return rke2Release, err
			}

			pr.commits = append(pr.commits, commits...)

			if resp.NextPage == 0 {
				break
			}
		}

	}

	// get Drone builds in a time range inclusive of the RKE2 release
	start := k8s.PublishedAt.Time
	end := ga.PublishedAt.Time.Add(time.Hour * 24)
	rke2PrBuilds, err := c.BuildList("rancher", "rke2", start, end)
	if err != nil {
		logrus.Errorln("Failed to get rancher/rke2 builds from drone-pr.rancher.io:", err)
		return rke2Release, err
	}
	rke2Release.builds = append(rke2Release.builds, rke2PrBuilds...)

	rke2PubBuilds, err := c.BuildList("rancher", "rke2", start, end.Add(time.Hour))
	if err != nil {
		logrus.Errorln("Failed to get rancher/rke2 builds from drone-publish.rancher.io:", err)
		return rke2Release, err
	}
	rke2Release.builds = append(rke2Release.builds, rke2PubBuilds...)

	rke2PkgBuilds, err := c.BuildList("rancher", "rke2-packaging", start, end)
	if err != nil {
		logrus.Errorln("failed to get rancher/rke2-packaging builds from drone-publish.rancher.io:", err)
		return rke2Release, err
	}
	rke2Release.builds = append(rke2Release.builds, rke2PkgBuilds...)

	k8sBuilds, err := c.BuildList("rancher", "image-build-kubernetes", start, end)
	if err != nil {
		logrus.Errorln("failed to get rancher/image-build-kubernetes builds from drone-publish.rancher.io:", err)
		return rke2Release, err
	}
	rke2Release.builds = append(rke2Release.builds, k8sBuilds...)

	return rke2Release, nil
}

type Build struct {
	Url   string
	Owner string
	Repo  string
	*drone.Build
}

func (c *Client) BuildList(owner, repo string, start, end time.Time) ([]*Build, error) {
	var result []*Build
	dronePrList, err := buildList(c.DronePr, owner, repo, start, end)
	if err != nil {
		return result, err
	}
	dronePubList, err := buildList(c.DronePub, owner, repo, start, end)
	if err != nil {
		return result, err
	}
	for _, build := range dronePrList {
		result = append(result, &Build{"drone-pr.rancher.io", owner, repo, build})
	}
	for _, build := range dronePubList {
		result = append(result, &Build{"drone-publish.rancher.io", owner, repo, build})
	}

	return result, nil
}

// buildList returns a slice of Drone Builds which were created or finished between the given times
func buildList(d buildLister, owner, repo string, start, end time.Time) ([]*drone.Build, error) {
	var builds []*drone.Build
	page := 1
	for {
		list, err := d.BuildList(owner, repo, drone.ListOptions{Page: page})
		if err != nil {
			logrus.Errorln("failed to get Drone publish list:", err)
			return builds, err
		}

		for _, build := range list {
			created := time.Unix(build.Created, 0)
			finished := time.Unix(build.Finished, 0)
			if finished.After(start) && created.Before(end) {
				builds = append(builds, build)
			}
		}

		// stop if the list contains builds older than the release
		cutoff := start.Add(-time.Hour * 24)
		last := list[len(list)-1]
		if time.Unix(last.Finished, 0).Before(cutoff) {
			break
		}
		page++
	}

	return builds, nil
}

// latest returns the greatest provided time
func latest(times ...time.Time) (result time.Time) {
	for _, t := range times {
		if t.After(result) {
			result = t
		}
	}
	return result
}

// patch returns the patch version portion of v
// e.g. "v1.21.1-rc1+rke2r1" -> "v1.21.1"
func patch(v string) string {
	c := semver.Canonical(v)
	p := semver.Prerelease(v)
	return c[0 : len(c)-len(p)]
}

// mainPatch returns the version of v without the prerelease identifier
// e.g. "v1.21.1-rc1+rke2r1" -> "v1.21.1+rke2r1"
func mainPatch(v string) string {
	return patch(v) + semver.Build(v)
}
