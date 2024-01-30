package github

import (
	"context"
	"strconv"

	"github.com/charmbracelet/lipgloss"
	"github.com/google/go-github/v39/github"
	"github.com/rancher/ecm-distro-tools/cmd/release/watch/style"
)

var PullRequestKind = "github_pull_request"

type PullRequest struct {
	number string
	org    string
	repo   string
	pr     *github.PullRequest
	status *github.CombinedStatus
	err    error
}

type PullRequestGetter interface {
	Get(ctx context.Context, owner string, repo string, number int) (*github.PullRequest, *github.Response, error)
}
type RepositoryStatusGetter interface {
	GetCombinedStatus(ctx context.Context, owner string, repo string, ref string, opt *github.ListOptions) (*github.CombinedStatus, *github.Response, error)
}
type PullRequestClient struct {
	PullRequests PullRequestGetter
	Repositories RepositoryStatusGetter
}

func NewPullRequest(ctx context.Context, token, org, repo, number string) *PullRequest {
	pr := &PullRequest{
		number: number,
		org:    org,
		repo:   repo,
	}

	var id int
	id, pr.err = strconv.Atoi(number)
	if pr.err != nil {
		return pr
	}

	client := newClient(ctx, token)
	pr.pr, _, pr.err = client.PullRequests.Get(ctx, org, repo, id)
	if pr.err != nil {
		return pr
	}
	pr.status, _, pr.err = client.Repositories.GetCombinedStatus(ctx, org, repo, pr.pr.Head.GetSHA(), nil)

	return pr
}

func (p PullRequest) Title() string {
	if p.pr == nil {
		return "[pr] " + p.number
	}
	return "[pr] " + p.number + " " + *p.pr.Title
}

func (p *PullRequest) Check() string {
	if p.status != nil {
		switch p.status.GetState() {
		case "success":
			return lipgloss.NewStyle().Foreground(style.Green).Render("✔")
		case "pending":
			return lipgloss.NewStyle().Foreground(style.Yellow).Render("•")
		case "failure":
			return lipgloss.NewStyle().Foreground(style.Red).Render("✘")
		}
	}
	return ""
}

func (p PullRequest) Status() string {
	var s string
	switch {
	case p.err != nil:
		s = style.Badge("error", style.Red)
	case p.pr == nil:
		s = style.Badge("unavailable", style.Contrast)
	case *p.pr.Draft:
		s = style.Badge("draft", style.Yellow)
	case *p.pr.Merged:
		s = style.Badge("merged", style.Purple)
	case p.pr.State != nil:
		switch *p.pr.State {
		case "open":
			s = style.Badge("open", style.Green)
		case "closed":
			s = style.Badge("closed", style.Red)
		default:
			s = style.Badge(*p.pr.State, style.Contrast)
		}
	default:
		s = style.Badge("unknown", style.Contrast)
	}

	if check := p.Check(); check != "" {
		return s + " " + check
	}

	return s
}

func (p PullRequest) Error() error {
	return p.err
}

func (p *PullRequest) FilterValue() string {
	var state string
	var title string
	if p.pr == nil {
		return "pull " + p.number + " " + p.org + "/" + p.repo
	}
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

func (p *PullRequest) Completed() bool {
	if p.pr != nil {
		if *p.pr.Merged {
			return true
		}
		if *p.pr.State == "closed" {
			return true
		}
	}
	return false
}
