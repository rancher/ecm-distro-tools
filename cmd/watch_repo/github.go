package main

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/charmbracelet/lipgloss"
	"github.com/google/go-github/v39/github"
	"golang.org/x/oauth2"
)

type release struct {
	tag     string
	org     string
	repo    string
	release github.RepositoryRelease
	err     error
}

func (r release) Key() string {
	return "release_" + r.tag + "_" + r.org + "_" + r.repo
}

func (r release) Title() string {
	if r.release.Name != nil {
		return "[release] " + r.tag + " " + *r.release.Name
	}
	return "[release] " + r.tag
}

func (r release) Description() string {
	var status string
	switch {
	case r.err != nil:
		status = redBlock.Width(12).Align(lipgloss.Center).Render("error")
		return status + "  " + subtle.Render(r.org+"/"+r.repo) + "  " + r.err.Error()
	case r.release.TagName == nil:
		status = greyBlock.Width(14).Render("unavailable")
	case r.release.Draft != nil && *r.release.Draft:
		status = greyBlock.Width(12).Align(lipgloss.Center).Render("draft")
	case r.release.Prerelease != nil && *r.release.Prerelease:
		status = yellowBlock.Width(12).Align(lipgloss.Center).Render("prerelease")
	case r.release.PublishedAt != nil:
		status = greenBlock.Width(12).Align(lipgloss.Center).Render("published")
	default:
		status = greenBlock.Width(16).Render("not published")
	}

	return status + "  " + subtle.Render(r.org+"/"+r.repo)

}

func (r release) FilterValue() string {
	if r.release.Name != nil {
		return "release " + r.tag + " " + *r.release.Name + " " + r.org + "/" + r.repo
	}
	return "release " + r.tag + " " + r.org + "/" + r.repo
}

func (r release) Completed() bool {
	var published bool
	var draft bool
	var prerelease bool
	if r.release.PublishedAt != nil {
		published = true
	}
	if r.release.Draft != nil {
		draft = *r.release.Draft
	}
	if r.release.Prerelease != nil {
		prerelease = *r.release.Prerelease
	}

	return published && !draft && !prerelease
}

func (r release) Record() record {
	return record{
		Type: "github_release",
		Id:   r.tag,
		Org:  r.org,
		Repo: r.repo,
	}
}

func (r *release) update(ctx context.Context, token string) error {
	client := githubClient(ctx, token)
	release, _, err := client.Repositories.GetReleaseByTag(ctx, r.org, r.repo, r.tag)
	if err != nil {
		return err
	}
	r.release = *release

	return nil
}

func (r release) Refresh(ctx context.Context, c config) listItem {
	r.err = nil
	if err := r.update(ctx, c.GitHubToken); err != nil {
		var ghErr *github.ErrorResponse
		if errors.As(err, &ghErr) {
			if ghErr.Response.StatusCode == http.StatusNotFound {
				return r
			}
		}
		r.err = err
	}
	return r
}

type pullRequest struct {
	number string
	org    string
	repo   string
	pr     github.PullRequest
	status string
	err    error
}

func (p pullRequest) Key() string {
	return "pull_request_" + p.number + "_" + p.org + "_" + p.repo
}

func (p pullRequest) Title() string {
	if p.pr.Title != nil {
		return "[PR] " + p.number + " " + *p.pr.Title
	}
	return "[PR] " + p.number
}

func (p pullRequest) Description() string {
	var state string
	switch {
	case p.err != nil:
		state = redBlock.Width(12).Align(lipgloss.Center).Render("error") + " " + p.err.Error()
	case p.pr.Draft != nil && *p.pr.Draft:
		state = greenBlock.Width(12).Align(lipgloss.Center).Render("draft")
	case p.pr.Merged != nil && *p.pr.Merged:
		state = purpleBlock.Width(12).Align(lipgloss.Center).Render("merged")
	case p.pr.State != nil:
		switch *p.pr.State {
		case "open":
			state = greenBlock.Width(12).Align(lipgloss.Center).Render("open")
		case "closed":
			state = redBlock.Width(12).Align(lipgloss.Center).Render("closed")
		default:
			state = whiteBlock.Width(12).Align(lipgloss.Center).Render(*p.pr.State)
		}
	default:
		state = greyBlock.Width(12).Align(lipgloss.Center).Render("unknown")
	}

	if p.err != nil {
		state = redBlock.Width(12).Render("unavailable")
		return state + "  " + subtle.Render(p.org+"/"+p.repo) + "  " + p.err.Error()
	}

	var status string
	switch p.status {
	case "success":
		status = greenText.Render(" ✔ ")
	case "pending":
		status = yellowText.Render(" • ")
	case "failure":
		status = redText.Render(" ✘ ")
	}

	return state + status + "  " + subtle.Render(p.org+"/"+p.repo)
}

func (p pullRequest) FilterValue() string {
	var state string
	var title string
	if p.pr.Title != nil {
		title = *p.pr.Title
	}
	if p.pr.State != nil {
		state = *p.pr.State
	}
	if p.pr.Merged != nil && *p.pr.Merged {
		state = "merged"
	}

	return "pull " + p.number + " " + title + " " + p.org + "/" + p.repo + " " + state
}

func (p pullRequest) Record() record {
	return record{
		Type: "github_pull_request",
		Id:   p.number,
		Org:  p.org,
		Repo: p.repo,
	}
}

func (p pullRequest) Completed() bool {
	if p.pr.Merged != nil && *p.pr.Merged {
		return true
	}
	if p.pr.State != nil && *p.pr.State == "closed" {
		return true
	}
	return false
}

func (p *pullRequest) update(ctx context.Context, token string) error {
	client := githubClient(ctx, token)
	number, err := strconv.Atoi(p.number)
	if err != nil {
		return err
	}
	pr, _, err := client.PullRequests.Get(ctx, p.org, p.repo, number)
	if err != nil {
		return err
	}
	p.pr = *pr

	status, _, err := client.Repositories.GetCombinedStatus(ctx, p.org, p.repo, pr.Head.GetSHA(), nil)
	if err != nil {
		return err
	}
	p.status = status.GetState()

	return nil
}

func (p pullRequest) Refresh(ctx context.Context, c config) listItem {
	p.err = nil
	if err := p.update(ctx, c.GitHubToken); err != nil {
		p.err = err
	}
	return p
}

func githubClient(ctx context.Context, token string) *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return github.NewClient(tc)
}
