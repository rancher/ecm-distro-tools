package watch

import (
	"context"
	"fmt"

	"github.com/rancher/ecm-distro-tools/cmd/release/config"
	"github.com/rancher/ecm-distro-tools/cmd/release/watch/drone"
	"github.com/rancher/ecm-distro-tools/cmd/release/watch/github"
	"github.com/rancher/ecm-distro-tools/cmd/release/watch/style"
)

// item represents a watch list item, and provides methods to refresh the item's source
// and to render the list item in the list.
type item struct {
	source source
	Kind   string
	Org    string
	Repo   string
	ID     string
	Server string
}

type source interface {
	Title() string
	Status() string
	FilterValue() string
	Completed() bool
	Error() error
}

func (i item) Completed() bool {
	if i.source == nil {
		return false
	}
	return i.source.Completed()
}

func (i item) Key() string {
	return fmt.Sprintf("%s %s %s %s %s", i.Kind, i.Server, i.Org, i.Repo, i.ID)
}

// FilterValue satisfies list.Item interface, required even if filtering is not enabled for the list.
func (i item) FilterValue() string {
	if i.source != nil {
		return i.source.FilterValue()
	}
	return ""
}

func (i item) Title() string {
	if i.source != nil {
		return i.source.Title()
	}
	return fmt.Sprintf("%s %s", i.ID, i.Kind)

}

func (i item) Description() string {
	var errMsg string
	var status string
	if i.source == nil {
		status = style.Badge("unavailable", style.Contrast)
	} else {
		status = i.source.Status()
		if err := i.source.Error(); err != nil {
			errMsg = " " + err.Error()
		}
	}

	return status + " " + i.Org + "/" + i.Repo + errMsg
}

func (i item) refresh(ctx context.Context, c config.Auth) item {
	switch i.Kind {
	case drone.BuildKind:
		i.source = drone.New(ctx, *c.Drone, i.Server, i.Org, i.Repo, i.ID)
	case github.PullRequestKind:
		i.source = github.NewPullRequest(ctx, c.GithubToken, i.Org, i.Repo, i.ID)
	case github.ReleaseKind:
		i.source = github.NewRelease(ctx, c.GithubToken, i.Org, i.Repo, i.ID)
	}
	return i
}
