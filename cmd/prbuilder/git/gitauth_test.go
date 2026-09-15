package git

import "testing"

func TestExtractOwnerRepo(t *testing.T) {
	tests := []struct {
		name          string
		remoteURL     string
		expectedOwner string
		expectedRepo  string
		expectError   bool
	}{
		{
			name:          "HTTPS URL with .git suffix",
			remoteURL:     "https://github.com/rancher/backup-restore-operator.git",
			expectedOwner: "rancher",
			expectedRepo:  "backup-restore-operator",
			expectError:   false,
		},
		{
			name:          "HTTPS URL without .git suffix",
			remoteURL:     "https://github.com/k3s-io/k3s",
			expectedOwner: "k3s-io",
			expectedRepo:  "k3s",
			expectError:   false,
		},
		{
			name:          "SSH URL (explicit)",
			remoteURL:     "ssh://git@github.com/rancher/charts.git",
			expectedOwner: "rancher",
			expectedRepo:  "charts",
			expectError:   false,
		},
		{
			name:          "SSH URL (SCP-like)",
			remoteURL:     "git@github.com:rancher/rancher.git",
			expectedOwner: "rancher",
			expectedRepo:  "rancher",
			expectError:   false,
		},
		{
			name:          "SSH URL (SCP-like) without .git",
			remoteURL:     "git@github.com:k3s-io/k3s",
			expectedOwner: "k3s-io",
			expectedRepo:  "k3s",
			expectError:   false,
		},
		{
			name:        "Invalid URL - no path",
			remoteURL:   "https://github.com",
			expectError: true,
		},
		{
			name:        "Invalid URL - single component path",
			remoteURL:   "https://github.com/rancher",
			expectError: true,
		},
		{
			name:        "Invalid URL - empty path",
			remoteURL:   "https://github.com/",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := ExtractOwnerRepo(tt.remoteURL)

			if tt.expectError {
				if err == nil {
					t.Errorf("ExtractOwnerRepo() expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ExtractOwnerRepo() unexpected error: %v", err)
				return
			}

			if owner != tt.expectedOwner {
				t.Errorf("ExtractOwnerRepo() owner = %q, want %q", owner, tt.expectedOwner)
			}

			if repo != tt.expectedRepo {
				t.Errorf("ExtractOwnerRepo() repo = %q, want %q", repo, tt.expectedRepo)
			}
		})
	}
}

func TestParseRemote(t *testing.T) {
	tests := []struct {
		name           string
		remoteURL      string
		expectedScheme string
		expectedHost   string
		expectedUser   string
		expectedPath   string
	}{
		{
			name:           "HTTPS URL",
			remoteURL:      "https://github.com/rancher/rancher.git",
			expectedScheme: "https",
			expectedHost:   "github.com",
			expectedUser:   "",
			expectedPath:   "/rancher/rancher.git",
		},
		{
			name:           "HTTP URL",
			remoteURL:      "http://github.com/rancher/rancher.git",
			expectedScheme: "http",
			expectedHost:   "github.com",
			expectedUser:   "",
			expectedPath:   "/rancher/rancher.git",
		},
		{
			name:           "SSH explicit URL",
			remoteURL:      "ssh://git@github.com/rancher/rancher.git",
			expectedScheme: "ssh",
			expectedHost:   "github.com",
			expectedUser:   "git",
			expectedPath:   "/rancher/rancher.git",
		},
		{
			name:           "SSH SCP-like URL",
			remoteURL:      "git@github.com:rancher/rancher.git",
			expectedScheme: "ssh",
			expectedHost:   "github.com",
			expectedUser:   "git",
			expectedPath:   "rancher/rancher.git",
		},
		{
			name:           "SSH SCP-like URL with different user",
			remoteURL:      "myuser@gitlab.com:myorg/myrepo.git",
			expectedScheme: "ssh",
			expectedHost:   "gitlab.com",
			expectedUser:   "myuser",
			expectedPath:   "myorg/myrepo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme, host, user, path := ParseRemote(tt.remoteURL)

			if scheme != tt.expectedScheme {
				t.Errorf("ParseRemote() scheme = %q, want %q", scheme, tt.expectedScheme)
			}

			if host != tt.expectedHost {
				t.Errorf("ParseRemote() host = %q, want %q", host, tt.expectedHost)
			}

			if user != tt.expectedUser {
				t.Errorf("ParseRemote() user = %q, want %q", user, tt.expectedUser)
			}

			if path != tt.expectedPath {
				t.Errorf("ParseRemote() path = %q, want %q", path, tt.expectedPath)
			}
		})
	}
}
