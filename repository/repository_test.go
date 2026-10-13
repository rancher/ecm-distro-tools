package repository

import (
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
)

func TestStripBackportTag(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{
			line: "[Release-1.24] Some backport",
			want: "Some backport",
		},
		{
			line: " [Release-1.24]  Some backport",
			want: "Some backport",
		},
		{
			line: "[Release 1.24] Some backport",
			want: "Some backport",
		},
		{
			line: "[release-1.24] Some backport",
			want: "Some backport",
		},
		{
			line: "Release race condition",
			want: "Release race condition",
		},
		{
			line: "[release-1.24] Release race condition",
			want: "Release race condition",
		},
		{
			line: "[master] Release race condition",
			want: "Release race condition",
		},
		{
			line: "[master] Feature",
			want: "Feature",
		},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			if got := stripBackportTag(tt.line); got != tt.want {
				t.Errorf("stripBackportTag() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpstreamRemote(t *testing.T) {
	tests := []struct {
		name          string
		remoteURL     string
		configuredURL string
		remoteName    string
		expectError   bool
	}{
		{
			name:          "Exact match HTTPS with .git",
			remoteURL:     "https://github.com/rancher/charts.git",
			configuredURL: "https://github.com/rancher/charts.git",
			remoteName:    "upstream",
			expectError:   false,
		},
		{
			name:          "Config has .git, lookup without",
			remoteURL:     "https://github.com/rancher/charts",
			configuredURL: "https://github.com/rancher/charts.git",
			remoteName:    "upstream",
			expectError:   false,
		},
		{
			name:          "Lookup has .git, config without",
			remoteURL:     "https://github.com/rancher/charts.git",
			configuredURL: "https://github.com/rancher/charts",
			remoteName:    "upstream",
			expectError:   false,
		},
		{
			name:          "SSH with .git suffix",
			remoteURL:     "https://github.com/rancher/charts.git",
			configuredURL: "git@github.com:rancher/charts.git",
			remoteName:    "upstream",
			expectError:   false,
		},
		{
			name:          "SSH without .git suffix",
			remoteURL:     "https://github.com/rancher/charts",
			configuredURL: "git@github.com:rancher/charts",
			remoteName:    "upstream",
			expectError:   false,
		},
		{
			name:          "Mixed SSH lookup, HTTPS config",
			remoteURL:     "git@github.com:rancher/charts.git",
			configuredURL: "https://github.com/rancher/charts",
			remoteName:    "origin",
			expectError:   false,
		},
		{
			name:          "Wrong repository",
			remoteURL:     "https://github.com/rancher/charts.git",
			configuredURL: "https://github.com/rancher/other-repo.git",
			remoteName:    "upstream",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary test repository
			tmpDir := t.TempDir()
			repo, err := git.PlainInit(tmpDir, false)
			if err != nil {
				t.Fatalf("failed to init test repo: %v", err)
			}

			// Add remote with configured URL
			_, err = repo.CreateRemote(&config.RemoteConfig{
				Name: tt.remoteName,
				URLs: []string{tt.configuredURL},
			})
			if err != nil {
				t.Fatalf("failed to create remote: %v", err)
			}

			// Test UpstreamRemote function
			remoteName, err := UpstreamRemote(repo, tt.remoteURL)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if remoteName != tt.remoteName {
					t.Errorf("expected remote name %q, got %q", tt.remoteName, remoteName)
				}
			}
		})
	}
}

func TestIsGitHubRepoURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://github.com/rancher/rancher-prime", true},
		{"https://github.com/rancher/rancher-prime.git", true},
		{"git@github.com:rancher/rancher-prime", true},
		{"git@github.com:rancher/rancher-prime.git", true},
		{"https://x-access-token:token@github.com/rancher/rancher-prime.git", false},
		{"ssh://git@github.com/rancher/rancher-prime.git", false},
		{"git@github.com:Rancher/Rancher-Prime", false},
		{"https://github.com/rancher/rancher-prime/", false},
		{"https://gitlab.com/rancher/rancher-prime.git", false},
		{"https://github.com.example.com/rancher/rancher-prime.git", false},
		{"git@example.com:rancher/rancher-prime.git", false},
		{"https://github.com/fork/rancher/rancher-prime.git", false},
		{"https://github.com/rancher/rancher.git", false},
		{"http://github.com/rancher/rancher-prime.git", false},
		{"/tmp/rancher/rancher-prime.git", false},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			if got := isGitHubRepoURL(tt.url, "rancher", "rancher-prime"); got != tt.want {
				t.Errorf("isGitHubRepoURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}
