package github

import (
	"testing"
	"time"

	"github.com/google/go-github/v39/github"
)

func TestPullRequestListItemView(t *testing.T) {
	ttime := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name        string
		pullRequest PullRequest
		title       string
		status      string
		filterValue string
		completed   bool
	}{
		{
			name:        "Empty PR",
			pullRequest: PullRequest{},
			title:       "[pr] ",
			status:      "unavailable ",
			filterValue: "pull  /",
		},
		{
			name: "open pr",
			pullRequest: PullRequest{
				org:    "my-org",
				repo:   "my-repo",
				number: "1234",
				pr: &github.PullRequest{
					Merged: github.Bool(false),
					State:  github.String("open"),
					Title:  github.String("My Open PR"),
					Draft:  github.Bool(false),
				},
			},
			title:       "[pr] 1234 My Open PR",
			status:      "    open    ",
			filterValue: "pull 1234 My Open PR my-org/my-repo open",
			completed:   false,
		},
		{
			name: "draft pr",
			pullRequest: PullRequest{
				org:    "my-org",
				repo:   "my-repo",
				number: "1234",
				pr: &github.PullRequest{
					Title:  github.String("My Draft PR"),
					State:  github.String("open"),
					Draft:  github.Bool(true),
					Merged: github.Bool(false),
				},
			},
			title:       "[pr] 1234 My Draft PR",
			status:      "   draft    ",
			filterValue: "pull 1234 My Draft PR my-org/my-repo open",
			completed:   false,
		},
		{
			name: "merged pr",
			pullRequest: PullRequest{
				org:    "my-org",
				repo:   "my-repo",
				number: "1234",
				pr: &github.PullRequest{
					Title:    github.String("My Published PR"),
					State:    github.String("closed"),
					Draft:    github.Bool(false),
					Merged:   github.Bool(true),
					MergedAt: &ttime,
				},
			},
			title:       "[pr] 1234 My Published PR",
			status:      "   merged   ",
			filterValue: "pull 1234 My Published PR my-org/my-repo merged",
			completed:   true,
		},
		{
			name: "closed pr",
			pullRequest: PullRequest{
				org:    "my-org",
				repo:   "my-repo",
				number: "1234",
				pr: &github.PullRequest{
					Title:    github.String("My Published Release"),
					Draft:    github.Bool(false),
					Merged:   github.Bool(true),
					State:    github.String("closed"),
					MergedAt: &ttime,
				},
			},
			title:       "[pr] 1234 My Published Release",
			status:      "   merged   ",
			filterValue: "pull 1234 My Published Release my-org/my-repo merged",
			completed:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := tt.pullRequest.Status()
			title := tt.pullRequest.Title()
			filterValue := tt.pullRequest.FilterValue()
			completed := tt.pullRequest.Completed()

			if title != tt.title {
				t.Errorf("Expected %s, got %s", tt.title, title)
			}
			if status != tt.status {
				t.Errorf("Expected %s, got %s", tt.status, status)
			}
			if filterValue != tt.filterValue {
				t.Errorf("Expected %s, got %s", tt.filterValue, filterValue)
			}
			if completed != tt.completed {
				t.Errorf("Expected %t, got %t", tt.completed, completed)
			}
		})
	}
}
