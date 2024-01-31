package drone

import (
	"testing"
	"time"

	"github.com/drone/drone-go/drone"
	"github.com/rancher/ecm-distro-tools/cmd/release/watch/style"
)

func TestBuildListItemView(t *testing.T) {
	ttime := time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)
	duration := style.FormatDuration(time.Since(ttime))
	tests := []struct {
		name        string
		build       Build
		title       string
		status      string
		filterValue string
		completed   bool
	}{
		{
			name:        "Empty build",
			build:       Build{},
			title:       "[drone] ",
			status:      "unavailable ",
			filterValue: "drone    ",
		},
		{
			name: "passing build without server",
			build: Build{
				org:    "my-org",
				repo:   "my-repo",
				number: "1234",
				build: &drone.Build{
					Title:   "My Build",
					Status:  drone.StatusPassing,
					Started: ttime.Unix(),
				},
			},
			title:       "[drone] 1234 My Build",
			status:      "  passing    " + duration,
			filterValue: "drone My Build my-org my-repo 1234",
			completed:   true,
		},
		{
			name: "passing build rancher pr",
			build: Build{
				org:    "my-org",
				repo:   "my-repo",
				number: "1234",
				server: RancherPrServer,
				build: &drone.Build{
					Title:   "My Build",
					Status:  drone.StatusPassing,
					Started: ttime.Unix(),
				},
			},
			title:       "[drone-pr] 1234 My Build",
			status:      "  passing    " + duration,
			filterValue: "drone My Build my-org my-repo 1234",
			completed:   true,
		},
		{
			name: "failing build",
			build: Build{
				org:    "my-org",
				repo:   "my-repo",
				number: "1234",
				build: &drone.Build{
					Title:   "My Build",
					Status:  drone.StatusFailing,
					Started: ttime.Unix(),
				},
			},
			title:       "[drone] 1234 My Build",
			status:      "  failing    " + duration,
			filterValue: "drone My Build my-org my-repo 1234",
			completed:   true,
		},
		{
			name: "blocked build",
			build: Build{
				org:    "my-org",
				repo:   "my-repo",
				number: "1234",
				build: &drone.Build{
					Title:   "My Build",
					Status:  drone.StatusBlocked,
					Started: ttime.Unix(),
				},
			},
			title:       "[drone] 1234 My Build",
			status:      "  blocked    " + duration,
			filterValue: "drone My Build my-org my-repo 1234",
			completed:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := tt.build.Status()
			title := tt.build.Title()
			filterValue := tt.build.FilterValue()
			completed := tt.build.Completed()

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
