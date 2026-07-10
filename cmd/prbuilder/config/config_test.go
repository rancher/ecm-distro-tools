package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name: "valid single-target config",
			config: Config{
				VersionMappingType: "major",
				Target: &Target{
					Repo:         "rancher/rancher",
					UpdateScript: "./scripts/bump.sh",
				},
			},
			expectError: false,
		},
		{
			name: "valid multi-target config",
			config: Config{
				VersionMappingType: "major.minor",
				Targets: []Target{
					{
						Repo:         "rancher/rancher",
						UpdateScript: "./scripts/bump.sh",
					},
					{
						Repo:         "rancher/rke2",
						UpdateScript: "./scripts/update.sh",
					},
				},
			},
			expectError: false,
		},
		{
			name: "invalid version mapping type",
			config: Config{
				VersionMappingType: "invalid",
				Target: &Target{
					Repo:         "rancher/rancher",
					UpdateScript: "./scripts/bump.sh",
				},
			},
			expectError: true,
		},
		{
			name: "both target and targets defined",
			config: Config{
				VersionMappingType: "major",
				Target: &Target{
					Repo:         "rancher/rancher",
					UpdateScript: "./scripts/bump.sh",
				},
				Targets: []Target{
					{
						Repo:         "rancher/rke2",
						UpdateScript: "./scripts/update.sh",
					},
				},
			},
			expectError: true,
		},
		{
			name: "neither target nor targets defined",
			config: Config{
				VersionMappingType: "major",
			},
			expectError: true,
		},
		{
			name: "single-target missing repo",
			config: Config{
				VersionMappingType: "major",
				Target: &Target{
					UpdateScript: "./scripts/bump.sh",
				},
			},
			expectError: true,
		},
		{
			name: "single-target missing update_script",
			config: Config{
				VersionMappingType: "major",
				Target: &Target{
					Repo: "rancher/rancher",
				},
			},
			expectError: true,
		},
		{
			name: "multi-target missing repo",
			config: Config{
				VersionMappingType: "major",
				Targets: []Target{
					{
						UpdateScript: "./scripts/bump.sh",
					},
				},
			},
			expectError: true,
		},
		{
			name: "multi-target missing update_script",
			config: Config{
				VersionMappingType: "major",
				Targets: []Target{
					{
						Repo: "rancher/rancher",
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError && err == nil {
				t.Errorf("Validate() expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
		})
	}
}

func TestConfig_IsSingleTarget(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected bool
	}{
		{
			name: "single-target config",
			config: Config{
				Target: &Target{
					Repo:         "rancher/rancher",
					UpdateScript: "./scripts/bump.sh",
				},
			},
			expected: true,
		},
		{
			name: "multi-target config",
			config: Config{
				Targets: []Target{
					{
						Repo:         "rancher/rancher",
						UpdateScript: "./scripts/bump.sh",
					},
				},
			},
			expected: false,
		},
		{
			name:     "no target config",
			config:   Config{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.IsSingleTarget()
			if got != tt.expected {
				t.Errorf("IsSingleTarget() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConfig_GetTargets(t *testing.T) {
	tests := []struct {
		name          string
		config        Config
		expectedCount int
	}{
		{
			name: "single-target config",
			config: Config{
				Target: &Target{
					Repo:         "rancher/rancher",
					UpdateScript: "./scripts/bump.sh",
				},
			},
			expectedCount: 1,
		},
		{
			name: "multi-target config",
			config: Config{
				Targets: []Target{
					{
						Repo:         "rancher/rancher",
						UpdateScript: "./scripts/bump.sh",
					},
					{
						Repo:         "rancher/rke2",
						UpdateScript: "./scripts/update.sh",
					},
				},
			},
			expectedCount: 2,
		},
		{
			name:          "no targets",
			config:        Config{},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targets := tt.config.TargetsList()
			if len(targets) != tt.expectedCount {
				t.Errorf("GetTargets() returned %d targets, want %d", len(targets), tt.expectedCount)
			}
		})
	}
}

func TestConfig_GetTargetBranches(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		target      Target
		version     string
		expected    []string
		expectError bool
	}{
		{
			name: "global mapping found",
			config: Config{
				VersionBranchMap: map[string]BranchOrBranches{
					"10": {"release-v2.10"},
				},
			},
			target:      Target{Repo: "rancher/rancher"},
			version:     "10",
			expected:    []string{"release-v2.10"},
			expectError: false,
		},
		{
			name: "global mapping multiple branches",
			config: Config{
				VersionBranchMap: map[string]BranchOrBranches{
					"10": {"release-v2.10", "dev-v2.10"},
				},
			},
			target:      Target{Repo: "rancher/rancher"},
			version:     "10",
			expected:    []string{"release-v2.10", "dev-v2.10"},
			expectError: false,
		},
		{
			name: "target-specific mapping overrides global",
			config: Config{
				VersionBranchMap: map[string]BranchOrBranches{
					"10": {"global-branch"},
				},
			},
			target: Target{
				Repo: "rancher/rancher",
				VersionBranchMap: map[string]BranchOrBranches{
					"10": {"target-specific-branch"},
				},
			},
			version:     "10",
			expected:    []string{"target-specific-branch"},
			expectError: false,
		},
		{
			name: "global wildcard fallback",
			config: Config{
				VersionBranchMap: map[string]BranchOrBranches{
					"*": {"main"},
				},
			},
			target:      Target{Repo: "rancher/rancher"},
			version:     "99",
			expected:    []string{"main"},
			expectError: false,
		},
		{
			name: "target-specific wildcard",
			config: Config{
				VersionBranchMap: map[string]BranchOrBranches{
					"*": {"global-main"},
				},
			},
			target: Target{
				Repo: "rancher/rancher",
				VersionBranchMap: map[string]BranchOrBranches{
					"*": {"target-main"},
				},
			},
			version:     "99",
			expected:    []string{"target-main"},
			expectError: false,
		},
		{
			name:        "no mapping found",
			config:      Config{},
			target:      Target{Repo: "rancher/rancher"},
			version:     "99",
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			branches, err := tt.config.TargetBranches(tt.version, &tt.target)

			if tt.expectError {
				if err == nil {
					t.Errorf("GetTargetBranches() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("GetTargetBranches() unexpected error: %v", err)
				return
			}

			if len(branches) != len(tt.expected) {
				t.Errorf("GetTargetBranches() returned %d branches, want %d", len(branches), len(tt.expected))
				return
			}

			for i, branch := range branches {
				if branch != tt.expected[i] {
					t.Errorf("GetTargetBranches()[%d] = %q, want %q", i, branch, tt.expected[i])
				}
			}
		})
	}
}

func TestBranchOrBranches_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		expected BranchOrBranches
	}{
		{
			name:     "single string",
			yaml:     "branch: main",
			expected: BranchOrBranches{"main"},
		},
		{
			name:     "array of strings",
			yaml:     "branch:\n  - main\n  - develop",
			expected: BranchOrBranches{"main", "develop"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result struct {
				Branch BranchOrBranches `yaml:"branch"`
			}

			err := yaml.Unmarshal([]byte(tt.yaml), &result)
			if err != nil {
				t.Errorf("Unmarshal() unexpected error: %v", err)
				return
			}

			if len(result.Branch) != len(tt.expected) {
				t.Errorf("Unmarshal() returned %d branches, want %d", len(result.Branch), len(tt.expected))
				return
			}

			for i, branch := range result.Branch {
				if branch != tt.expected[i] {
					t.Errorf("Unmarshal()[%d] = %q, want %q", i, branch, tt.expected[i])
				}
			}
		})
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		expectError bool
	}{
		{
			name: "valid config with defaults",
			configYAML: `
target:
  repo: rancher/rancher
  update_script: ./scripts/bump.sh
version_branch_map:
  "10": release-v2.10
`,
			expectError: false,
		},
		{
			name: "valid config with explicit version_mapping_type",
			configYAML: `
version_mapping_type: major.minor
target:
  repo: rancher/rancher
  update_script: ./scripts/bump.sh
version_branch_map:
  "10.3": release-v2.10
`,
			expectError: false,
		},
		{
			name: "invalid YAML",
			configYAML: `
invalid: yaml: content:
`,
			expectError: true,
		},
		{
			name: "missing required fields",
			configYAML: `
version_mapping_type: major
`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			configFile := filepath.Join(tmpDir, "config.yml")
			err := os.WriteFile(configFile, []byte(tt.configYAML), 0644)
			if err != nil {
				t.Fatalf("Failed to write test config: %v", err)
			}

			// Test Load
			cfg, err := Load(configFile)

			if tt.expectError {
				if err == nil {
					t.Errorf("Load() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Load() unexpected error: %v", err)
				return
			}

			// Verify defaults are set
			if cfg.VersionMappingType == "" {
				t.Errorf("Load() did not set default version_mapping_type")
			}
		})
	}
}
