package github

import (
	"testing"
	"time"

	"github.com/google/go-github/v39/github"
)

func TestReleaseListItemView(t *testing.T) {
	ttime := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name        string
		release     Release
		title       string
		status      string
		filterValue string
		completed   bool
	}{
		{
			name:        "Empty release",
			release:     Release{},
			title:       "[release] ",
			status:      "unavailable ",
			filterValue: "release  /",
		},
		{
			name: "unpublished release",
			release: Release{
				org:  "my-org",
				repo: "my-repo",
				tag:  "v1.2.3",
				release: &github.RepositoryRelease{
					TagName:     github.String("v1.2.3"),
					Name:        github.String("My Unpublished Release"),
					Draft:       github.Bool(false),
					Prerelease:  github.Bool(true),
					PublishedAt: nil,
				},
			},
			title:       "[release] v1.2.3 My Unpublished Release",
			status:      " prerelease ",
			filterValue: "release v1.2.3 My Unpublished Release my-org/my-repo",
			completed:   false,
		},
		{
			name: "draft release",
			release: Release{
				org:  "my-org",
				repo: "my-repo",
				tag:  "v1.2.3",
				release: &github.RepositoryRelease{
					TagName:     github.String("v1.2.3"),
					Name:        github.String("My Draft Release"),
					Draft:       github.Bool(true),
					Prerelease:  github.Bool(true),
					PublishedAt: nil,
				},
			},
			title:       "[release] v1.2.3 My Draft Release",
			status:      "   draft    ",
			filterValue: "release v1.2.3 My Draft Release my-org/my-repo",
			completed:   false,
		},
		{
			name: "published release",
			release: Release{
				org:  "my-org",
				repo: "my-repo",
				tag:  "v1.2.3",
				release: &github.RepositoryRelease{
					TagName:     github.String("v1.2.3"),
					Name:        github.String("My Published Release"),
					Draft:       github.Bool(false),
					PublishedAt: &github.Timestamp{Time: ttime},
					Prerelease:  github.Bool(false),
				},
			},
			title:       "[release] v1.2.3 My Published Release",
			status:      " published  ",
			filterValue: "release v1.2.3 My Published Release my-org/my-repo",
			completed:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := tt.release.Status()
			title := tt.release.Title()
			filterValue := tt.release.FilterValue()
			completed := tt.release.Completed()

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
