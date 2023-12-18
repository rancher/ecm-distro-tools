package main

import (
	"testing"
	"time"

	"github.com/drone/drone-go/drone"
)

func TestBuildListItemView(t *testing.T) {
	ttime := time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)
	duration := formatDuration(time.Since(ttime))
	tests := []struct {
		name        string
		build       droneBuild
		title       string
		description string
		key         string
		filterValue string
		completed   bool
	}{
		{
			name:        "Empty build",
			build:       droneBuild{},
			title:       "[Drone] ",
			description: "  unknown     /  ",
			key:         "drone_build___",
			filterValue: "drone   / ",
		},
		{
			name: "passing build missing server",
			build: droneBuild{
				org:    "my-org",
				repo:   "my-repo",
				number: "1234",
				build: drone.Build{
					Title:   "My Build",
					Status:  drone.StatusPassing,
					Started: ttime.Unix(),
				},
			},
			title:       "[Drone] 1234",
			description: "  passing     my-org/my-repo  " + duration,
			key:         "drone_build__my-org_my-repo1234",
			filterValue: "drone 1234 My Build my-org/my-repo success",
			completed:   true,
		},
		{
			name: "passing build rancher pr",
			build: droneBuild{
				org:    "my-org",
				repo:   "my-repo",
				number: "1234",
				server: droneRancherPrServer,
				build: drone.Build{
					Title:   "My Build",
					Status:  drone.StatusPassing,
					Started: ttime.Unix(),
				},
			},
			title:       "[Drone PR] 1234",
			description: "  passing     my-org/my-repo  " + duration,
			key:         "drone_build_https://drone-pr.rancher.io_my-org_my-repo1234",
			filterValue: "drone 1234 My Build my-org/my-repo success",
			completed:   true,
		},
		{
			name: "failing build",
			build: droneBuild{
				org:    "my-org",
				repo:   "my-repo",
				number: "1234",
				build: drone.Build{
					Title:   "My Build",
					Status:  drone.StatusFailing,
					Started: ttime.Unix(),
				},
			},
			title:       "[Drone] 1234",
			description: "  failing     my-org/my-repo  " + duration,
			key:         "drone_build__my-org_my-repo1234",
			filterValue: "drone 1234 My Build my-org/my-repo failure",
			completed:   true,
		},
		{
			name: "blocked build",
			build: droneBuild{
				org:    "my-org",
				repo:   "my-repo",
				number: "1234",
				build: drone.Build{
					Title:   "My Build",
					Status:  drone.StatusBlocked,
					Started: ttime.Unix(),
				},
			},
			title:       "[Drone] 1234",
			description: "  blocked     my-org/my-repo  " + duration,
			key:         "drone_build__my-org_my-repo1234",
			filterValue: "drone 1234 My Build my-org/my-repo blocked",
			completed:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			description := tt.build.Description()
			title := tt.build.Title()
			key := tt.build.Key()
			filterValue := tt.build.FilterValue()
			completed := tt.build.Completed()

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
