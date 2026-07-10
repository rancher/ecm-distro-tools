package prbuilder

import (
	"errors"
	"testing"

	"github.com/rancher/ecm-distro-tools/cmd/prbuilder/config"
)

func TestNewPRBuilder(t *testing.T) {
	tests := []struct {
		name        string
		opts        Options
		expectError bool
	}{
		{
			name: "valid options with major version mapping",
			opts: Options{
				Config: &config.Config{
					VersionMappingType: "major",
					Target: &config.Target{
						Repo:         "rancher/rancher",
						UpdateScript: "./scripts/bump.sh",
					},
				},
				Tag:           "v10.3.2",
				SourceRepoDir: "/tmp/source",
				DryRun:        false,
			},
			expectError: false,
		},
		{
			name: "valid options with major.minor version mapping",
			opts: Options{
				Config: &config.Config{
					VersionMappingType: "major.minor",
					Target: &config.Target{
						Repo:         "rancher/rancher",
						UpdateScript: "./scripts/bump.sh",
					},
				},
				Tag:           "v10.3.2",
				SourceRepoDir: "/tmp/source",
				DryRun:        false,
			},
			expectError: false,
		},
		{
			name: "invalid tag format",
			opts: Options{
				Config: &config.Config{
					VersionMappingType: "major",
					Target: &config.Target{
						Repo:         "rancher/rancher",
						UpdateScript: "./scripts/bump.sh",
					},
				},
				Tag:           "invalid",
				SourceRepoDir: "/tmp/source",
			},
			expectError: true,
		},
		{
			name: "default remote",
			opts: Options{
				Config: &config.Config{
					VersionMappingType: "major",
					Target: &config.Target{
						Repo:         "rancher/rancher",
						UpdateScript: "./scripts/bump.sh",
					},
				},
				Tag:           "v10.3.2",
				SourceRepoDir: "/tmp/source",
				Remote:        "", // Should default to "origin"
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb, err := NewPRBuilder(tt.opts)

			if tt.expectError {
				if err == nil {
					t.Errorf("NewPRBuilder() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("NewPRBuilder() unexpected error: %v", err)
				return
			}

			if pb == nil {
				t.Errorf("NewPRBuilder() returned nil PRBuilder")
				return
			}

			// Verify remote defaults to "origin"
			if tt.opts.Remote == "" && pb.remote != "origin" {
				t.Errorf("NewPRBuilder() remote = %q, want %q", pb.remote, "origin")
			}

			// Verify version was parsed correctly
			if tt.opts.Config.VersionMappingType == "major" {
				expectedVersion := "10"
				if pb.version != expectedVersion {
					t.Errorf("NewPRBuilder() version = %q, want %q", pb.version, expectedVersion)
				}
			}
			if tt.opts.Config.VersionMappingType == "major.minor" {
				expectedVersion := "10.3"
				if pb.version != expectedVersion {
					t.Errorf("NewPRBuilder() version = %q, want %q", pb.version, expectedVersion)
				}
			}
		})
	}
}

func TestPRResult(t *testing.T) {
	tests := []struct {
		name     string
		result   Result
		hasError bool
		hasPRURL bool
	}{
		{
			name: "successful result",
			result: Result{
				TargetRepo: "rancher/rancher (release-v2.10)",
				PRURL:      "https://github.com/rancher/rancher/pull/123",
				Error:      nil,
			},
			hasError: false,
			hasPRURL: true,
		},
		{
			name: "error result",
			result: Result{
				TargetRepo: "rancher/rancher (release-v2.10)",
				PRURL:      "",
				Error:      errors.New("failed to create PR"),
			},
			hasError: true,
			hasPRURL: false,
		},
		{
			name: "no changes result",
			result: Result{
				TargetRepo: "rancher/rancher (release-v2.10)",
				PRURL:      "",
				Error:      nil,
			},
			hasError: false,
			hasPRURL: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if (tt.result.Error != nil) != tt.hasError {
				t.Errorf("PRResult.Error presence = %v, want %v", tt.result.Error != nil, tt.hasError)
			}
			if (tt.result.PRURL != "") != tt.hasPRURL {
				t.Errorf("PRResult.PRURL presence = %v, want %v", tt.result.PRURL != "", tt.hasPRURL)
			}
		})
	}
}
