package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// BranchOrBranches handles a value that can be a single branch or a list of branches.
type BranchOrBranches []string

func (s *BranchOrBranches) UnmarshalYAML(value *yaml.Node) error {
	var str string
	if err := value.Decode(&str); err == nil {
		*s = []string{str}
		return nil
	}

	var slice []string
	if err := value.Decode(&slice); err != nil {
		return fmt.Errorf("failed to parse string or list of strings: %w", err)
	}

	*s = slice
	return nil
}

// Config represents the prbuilder configuration file
type Config struct {
	VersionMappingType string                      `yaml:"version_mapping_type"` // "major" or "major.minor"
	VersionBranchMap   map[string]BranchOrBranches `yaml:"version_branch_map"`   // Can be string or []string
	Target             *Target                     `yaml:"target,omitempty"`     // Single-target mode (singular)
	Targets            []Target                    `yaml:"targets,omitempty"`    // Multi-target mode (plural)
}

// Target represents a target repository configuration
type Target struct {
	Repo             string                      `yaml:"repo"`                         // e.g. "rancher/rancher"
	UpdateScript     string                      `yaml:"update_script"`                // e.g. "./scripts/bump.sh"
	PostUpdateScript string                      `yaml:"post_update_script,omitempty"` // Optional post-update script
	VersionBranchMap map[string]BranchOrBranches `yaml:"version_branch_map,omitempty"` // Optional per-target version mapping (can be string or []string)
}

// Load reads and parses the config file from the given path
func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	defer f.Close()

	d := yaml.NewDecoder(f)

	var cfg Config
	if err := d.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Set defaults
	if cfg.VersionMappingType == "" {
		cfg.VersionMappingType = "major"
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate checks that the config has all required fields and valid values
func (c *Config) Validate() error {
	// Validate version mapping type
	if c.VersionMappingType != "major" && c.VersionMappingType != "major.minor" {
		return errors.New("invalid version_mapping_type: " + c.VersionMappingType + " (must be 'major' or 'major.minor')")
	}

	// Check that we have either target (singular) or targets (plural), but not both
	if c.Target != nil && len(c.Targets) > 0 {
		return errors.New("config cannot have both 'target' and 'targets' fields - use 'target' for single-target mode or 'targets' for multi-target mode")
	}

	if c.Target == nil && len(c.Targets) == 0 {
		return errors.New("config must define either 'target' (single-target mode) or 'targets' (multi-target mode)")
	}

	// Validate single-target mode
	if c.Target != nil {
		if c.Target.Repo == "" {
			return errors.New("target.repo is required")
		}
		if c.Target.UpdateScript == "" {
			return errors.New("target.update_script is required")
		}
	}

	// Validate multi-target mode
	for i, target := range c.Targets {
		if target.Repo == "" {
			return errors.New("targets[" + strconv.Itoa(i) + "]: repo is required")
		}
		if target.UpdateScript == "" {
			return fmt.Errorf("targets[%d] (%s): update_script is required", i, target.Repo)
		}
	}

	return nil
}

// IsSingleTarget returns true if this is a single-target config
func (c *Config) IsSingleTarget() bool {
	return c.Target != nil
}

// TargetsList returns the targets list, converting single target if needed
func (c *Config) Targets() []Target {
	if c.Target != nil {
		return []Target{*c.Target}
	}
	return c.Targets
}

// TargetBranches resolves the branch(es) for a given version and target.
// Supports both single branch (string) and multiple branches ([]string).
// Uses a priority-based resolution strategy:
// 1. Target-specific mapping (allows per-repo overrides for the version)
// 2. Target-specific wildcard (catch-all for this specific repo)
// 3. Global mapping (default mapping for all repos for this version)
// 4. Global wildcard (final fallback for all repos)
// This priority order allows repos to override global settings while
// maintaining a sensible default for most cases.
func (c *Config) TargetBranches(version string, target *Target) ([]string, error) {
	// Priority 1: Target-specific mapping
	if target.VersionBranchMap != nil {
		if branches, ok := target.VersionBranchMap[version]; ok {
			if len(branches) > 0 {
				return branches, nil
			}
		}
		// Priority 2: Target-specific wildcard
		if branches, ok := target.VersionBranchMap["*"]; ok {
			if len(branches) > 0 {
				return branches, nil
			}
		}
	}

	// Priority 3: Global mapping
	if branches, ok := c.VersionBranchMap[version]; ok {
		if len(branches) > 0 {
			return branches, nil
		}
	}

	// Priority 4: Global wildcard (final fallback)
	if branches, ok := c.VersionBranchMap["*"]; ok {
		if len(branches) > 0 {
			return branches, nil
		}
	}

	return nil, errors.New("no branch mapping found for version " + version + " in config (checked target-specific and global mappings, including wildcards)")
}
