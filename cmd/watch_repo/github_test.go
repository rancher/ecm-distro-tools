package main

import (
	"testing"
	"time"

	"github.com/google/go-github/v39/github"
)

func TestReleaseListItemView(t *testing.T) {
	ttime := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name        string
		release     release
		title       string
		description string
		key         string
		filterValue string
		completed   bool
	}{
		{
			name:        "Empty release",
			release:     release{},
			title:       "[release] ",
			description: " unavailable    /",
			key:         "release___",
			filterValue: "release  /",
		},
		{
			name: "unpublished release",
			release: release{
				org:  "my-org",
				repo: "my-repo",
				tag:  "v1.2.3",
				release: github.RepositoryRelease{
					TagName: github.String("v1.2.3"),
					Name:    github.String("My Unpublished Release"),
				},
			},
			title:       "[release] v1.2.3 My Unpublished Release",
			description: " not published    my-org/my-repo",
			key:         "release_v1.2.3_my-org_my-repo",
			filterValue: "release v1.2.3 My Unpublished Release my-org/my-repo",
			completed:   false,
		},
		{
			name: "draft release",
			release: release{
				org:  "my-org",
				repo: "my-repo",
				tag:  "v1.2.3",
				release: github.RepositoryRelease{
					TagName: github.String("v1.2.3"),
					Name:    github.String("My Draft Release"),
					Draft:   github.Bool(true),
				},
			},
			title:       "[release] v1.2.3 My Draft Release",
			description: "   draft      my-org/my-repo",
			key:         "release_v1.2.3_my-org_my-repo",
			filterValue: "release v1.2.3 My Draft Release my-org/my-repo",
			completed:   false,
		},
		{
			name: "published release",
			release: release{
				org:  "my-org",
				repo: "my-repo",
				tag:  "v1.2.3",
				release: github.RepositoryRelease{
					TagName:     github.String("v1.2.3"),
					Name:        github.String("My Published Release"),
					Draft:       github.Bool(false),
					PublishedAt: &github.Timestamp{Time: ttime},
				},
			},
			title:       "[release] v1.2.3 My Published Release",
			description: " published    my-org/my-repo",
			key:         "release_v1.2.3_my-org_my-repo",
			filterValue: "release v1.2.3 My Published Release my-org/my-repo",
			completed:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			description := tt.release.Description()
			title := tt.release.Title()
			key := tt.release.Key()
			filterValue := tt.release.FilterValue()
			completed := tt.release.Completed()

			if title != tt.title {
				t.Errorf("Expected %s, got %s", tt.title, title)
			}
			if description != tt.description {
				t.Errorf("Expected %s, got %s", tt.description, description)
			}
			if key != tt.key {
				t.Errorf("Expected %s, got %s", tt.key, key)
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

func TestPullRequestListItemView(t *testing.T) {
	ttime := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name        string
		pullRequest pullRequest
		title       string
		description string
		key         string
		filterValue string
		completed   bool
	}{
		{
			name:        "Empty PR",
			pullRequest: pullRequest{},
			title:       "[PR] ",
			description: "  unknown     /",
			key:         "pull_request___",
			filterValue: "pull   / ",
		},
		{
			name: "open pr",
			pullRequest: pullRequest{
				org:    "my-org",
				repo:   "my-repo",
				number: "1234",
				pr: github.PullRequest{
					Merged: github.Bool(false),
					State:  github.String("open"),
					Title:  github.String("My Open PR"),
				},
			},
			title:       "[PR] 1234 My Open PR",
			description: "    open      my-org/my-repo",
			key:         "pull_request_1234_my-org_my-repo",
			filterValue: "pull 1234 My Open PR my-org/my-repo open",
			completed:   false,
		},
		{
			name: "draft pr",
			pullRequest: pullRequest{
				org:    "my-org",
				repo:   "my-repo",
				number: "1234",
				pr: github.PullRequest{
					Title: github.String("My Draft PR"),
					State: github.String("open"),
					Draft: github.Bool(true),
				},
			},
			title:       "[PR] 1234 My Draft PR",
			description: "   draft      my-org/my-repo",
			key:         "pull_request_1234_my-org_my-repo",
			filterValue: "pull 1234 My Draft PR my-org/my-repo open",
			completed:   false,
		},
		{
			name: "merged pr",
			pullRequest: pullRequest{
				org:    "my-org",
				repo:   "my-repo",
				number: "1234",
				pr: github.PullRequest{
					Title:    github.String("My Published PR"),
					State:    github.String("closed"),
					Draft:    github.Bool(false),
					Merged:   github.Bool(true),
					MergedAt: &ttime,
				},
			},
			title:       "[PR] 1234 My Published PR",
			description: "   merged     my-org/my-repo",
			key:         "pull_request_1234_my-org_my-repo",
			filterValue: "pull 1234 My Published PR my-org/my-repo merged",
			completed:   true,
		},
		{
			name: "closed pr",
			pullRequest: pullRequest{
				org:    "my-org",
				repo:   "my-repo",
				number: "1234",
				pr: github.PullRequest{
					Title:    github.String("My Published Release"),
					Draft:    github.Bool(false),
					Merged:   github.Bool(true),
					MergedAt: &ttime,
				},
			},
			title:       "[PR] 1234 My Published Release",
			description: "   merged     my-org/my-repo",
			key:         "pull_request_1234_my-org_my-repo",
			filterValue: "pull 1234 My Published Release my-org/my-repo merged",
			completed:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			description := tt.pullRequest.Description()
			title := tt.pullRequest.Title()
			key := tt.pullRequest.Key()
			filterValue := tt.pullRequest.FilterValue()
			completed := tt.pullRequest.Completed()

			if title != tt.title {
				t.Errorf("Expected %s, got %s", tt.title, title)
			}
			if description != tt.description {
				t.Errorf("Expected %s, got %s", tt.description, description)
			}
			if key != tt.key {
				t.Errorf("Expected %s, got %s", tt.key, key)
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
