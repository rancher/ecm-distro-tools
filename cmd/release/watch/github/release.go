package github

import (
	"context"

	"github.com/google/go-github/v39/github"
	"github.com/rancher/ecm-distro-tools/cmd/release/watch/style"
)

var ReleaseKind = "github_release"

type Release struct {
	tag     string
	org     string
	repo    string
	release *github.RepositoryRelease
	err     error
}

func NewRelease(ctx context.Context, token, org, repo, tag string) *Release {
	release := &Release{
		tag:  tag,
		org:  org,
		repo: repo,
	}
	client := newClient(ctx, token)
	rel, res, err := client.Repositories.GetReleaseByTag(ctx, org, repo, tag)
	if err != nil {
		if res != nil && res.StatusCode == 404 {
			release.release = nil
			release.err = nil
		} else {
			release.err = err
		}
	}
	release.release = rel
	return release
}

func (r Release) Title() string {
	if r.release == nil || r.tag == *r.release.Name {
		return "[release] " + r.tag
	}
	return "[release] " + r.tag + " " + *r.release.Name
}

func (r Release) Status() string {
	switch {
	case r.err != nil:
		return style.Badge("error", style.Red)
	case r.release == nil:
		return style.Badge("unavailable", style.Contrast)
	case *r.release.Draft:
		return style.Badge("draft", style.Contrast)
	case *r.release.Prerelease:
		return style.Badge("prerelease", style.Yellow)
	case r.release.PublishedAt != nil:
		return style.Badge("published", style.Green)
	default:
		return style.Badge("not published", style.Yellow)
	}
}

func (r Release) Error() error {
	return r.err
}

func (r Release) FilterValue() string {
	if r.release != nil {
		return "release " + r.tag + " " + *r.release.Name + " " + r.org + "/" + r.repo
	}
	return "release " + r.tag + " " + r.org + "/" + r.repo
}

func (r Release) Completed() bool {
	switch {
	case r.release == nil:
		return false
	case r.release.PublishedAt == nil:
		return false
	case *r.release.Draft:
		return false
	case *r.release.Prerelease:
		return false
	default:
		return true
	}
}
