package prbuilder

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rancher/ecm-distro-tools/cmd/prbuilder/config"
	"github.com/rancher/ecm-distro-tools/exec"
	"github.com/sirupsen/logrus"
)

// PRBuilder handles the creation of PRs in target repositories
type PRBuilder struct {
	config        *config.Config
	tag           string
	version       string
	sourceRepoDir string
	dryRun        bool
	targetDir     string // For ad-hoc mode: path to already-cloned repo
	remote        string // Git remote name (default: origin)
}

// PRResult represents the result of processing a single target
type PRResult struct {
	TargetRepo string
	PRURL      string
	Error      error
}

// Options contains configuration for creating a PRBuilder
type Options struct {
	Config        *config.Config
	Tag           string
	SourceRepoDir string
	DryRun        bool
	TargetDir     string // For ad-hoc mode: path to already-cloned repo
	Remote        string // Git remote name (default: origin)
}

// NewPRBuilder creates a new PR builder instance
func NewPRBuilder(opts Options) (*PRBuilder, error) {
	// Parse version from tag
	version, err := ParseVersion(opts.Tag, opts.Config.VersionMappingType)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version from tag %s: %w", opts.Tag, err)
	}

	// Set default remote if not specified
	remote := opts.Remote
	if remote == "" {
		remote = "origin"
	}

	return &PRBuilder{
		config:        opts.Config,
		tag:           opts.Tag,
		version:       version,
		sourceRepoDir: opts.SourceRepoDir,
		dryRun:        opts.DryRun,
		targetDir:     opts.TargetDir,
		remote:        remote,
	}, nil
}

// ProcessTargets processes all configured targets and creates PRs
func (pb *PRBuilder) ProcessTargets(ctx context.Context) ([]PRResult, error) {
	targets := pb.config.GetTargets()

	logrus.Infof("Starting PR creation process for tag: %s", pb.tag)
	logrus.Infof("Extracted version: %s", pb.version)
	logrus.Infof("Found %d target(s)", len(targets))

	results := make([]PRResult, 0)

	for i, target := range targets {
		logrus.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		logrus.Infof("Processing target %d/%d: %s", i+1, len(targets), target.Repo)

		// Get target branches (may be multiple)
		branches, err := pb.config.GetTargetBranches(pb.version, &target)
		if err != nil {
			result := PRResult{
				TargetRepo: target.Repo,
				Error:      err,
			}
			results = append(results, result)
			logrus.Warnf("Failed to resolve branches for %s: %v", target.Repo, err)
			continue
		}

		logrus.Infof("Target branches: %v", branches)

		// Process each branch
		for _, branch := range branches {
			if len(branches) > 1 {
				logrus.Infof("Processing branch: %s", branch)
			}

			result := pb.processTarget(ctx, &target, branch)
			results = append(results, result)

			if result.Error != nil {
				logrus.Warnf("Failed to process %s (branch: %s): %v", target.Repo, branch, result.Error)
				continue
			}

			if result.PRURL != "" {
				logrus.Infof("Successfully created PR: %s", result.PRURL)
			}
		}
	}

	return results, nil
}

// processTarget handles a single target repository for a specific branch
func (pb *PRBuilder) processTarget(ctx context.Context, target *config.Target, targetBranch string) PRResult {
	result := PRResult{TargetRepo: target.Repo + " (" + targetBranch + ")"}
	logrus.Infof("Update script: %s", target.UpdateScript)
	if target.PostUpdateScript != "" {
		logrus.Infof("Post-update script: %s", target.PostUpdateScript)
	}
	logrus.Infof("Git remote: %s", pb.remote)

	var workDir string
	var cleanupDir bool
	var err error

	// Two modes of operation:
	// 1. Ad-hoc mode (targetDir set): Uses an existing local clone of the target repo.
	//    This is useful for local development and testing without repeatedly cloning.
	//    Requires single-target config mode (--target-dir flag).
	// 2. Normal mode (targetDir empty): Clones the target repo into a temp directory.
	//    This is used in CI/automation where fresh clones are needed for each run.
	//    The temp directory is cleaned up after processing.
	if pb.targetDir != "" {
		// Ad-hoc mode: use existing repository clone
		workDir = pb.targetDir
		cleanupDir = false
		logrus.Infof("Using existing repository at: %s", workDir)

		// Checkout the target branch to ensure we're on the right branch
		logrus.Infof("Checking out branch %s...", targetBranch)
		_, err = exec.RunCommand(workDir, "git", "checkout", targetBranch)
		if err != nil {
			result.Error = fmt.Errorf("failed to checkout branch %s: %w", targetBranch, err)
			return result
		}

		// Pull latest changes to ensure we're up-to-date
		_, err = exec.RunCommand(workDir, "git", "pull", pb.remote, targetBranch)
		if err != nil {
			// Non-fatal: continue even if pull fails (might be working offline or with local changes)
			logrus.Warnf("Failed to pull latest changes: %v (continuing anyway)", err)
		}
	} else {
		// Normal mode: clone repository into temporary directory
		workDir, err = os.MkdirTemp("", "prbuilder-*")
		if err != nil {
			result.Error = fmt.Errorf("failed to create temp directory: %w", err)
			return result
		}
		cleanupDir = true
		defer func() {
			if cleanupDir {
				os.RemoveAll(workDir)
			}
		}()

		logrus.Infof("Working directory: %s", workDir)

		// Clone target repo with shallow clone for faster operation
		logrus.Infof("Cloning %s...", target.Repo)
		_, err = exec.RunCommand(workDir, "gh", "repo", "clone", target.Repo, ".", "--", "--depth=1", fmt.Sprintf("--branch=%s", targetBranch))
		if err != nil {
			result.Error = fmt.Errorf("failed to clone %s on branch %s: %w", target.Repo, targetBranch, err)
			return result
		}
	}

	// Create PR branch
	prBranch := fmt.Sprintf("bump-to-%s-%d", pb.tag, time.Now().Unix())
	_, err = exec.RunCommand(workDir, "git", "checkout", "-b", prBranch)
	if err != nil {
		result.Error = fmt.Errorf("failed to create branch %s: %w", prBranch, err)
		return result
	}

	logrus.Infof("Created branch: %s", prBranch)

	// Execute update script
	if err := pb.executeUpdateScript(workDir, target, targetBranch); err != nil {
		result.Error = err
		return result
	}

	// Check for changes
	status, err := exec.RunCommand(workDir, "git", "status", "--porcelain")
	if err != nil {
		result.Error = fmt.Errorf("failed to check git status: %w", err)
		return result
	}

	if strings.TrimSpace(status) == "" {
		logrus.Warn("No changes detected after running scripts")
		return result
	}

	// Commit changes
	_, err = exec.RunCommand(workDir, "git", "add", "-A")
	if err != nil {
		result.Error = fmt.Errorf("failed to stage changes: %w", err)
		return result
	}

	_, err = exec.RunCommand(workDir, "git", "config", "user.name", "github-actions[bot]")
	if err != nil {
		result.Error = fmt.Errorf("failed to set git user.name: %w", err)
		return result
	}

	_, err = exec.RunCommand(workDir, "git", "config", "user.email", "github-actions[bot]@users.noreply.github.com")
	if err != nil {
		result.Error = fmt.Errorf("failed to set git user.email: %w", err)
		return result
	}

	commitMsg := fmt.Sprintf("Bump to %s", pb.tag)
	_, err = exec.RunCommand(workDir, "git", "commit", "-m", commitMsg, "-m", "Automated version bump from upstream release")
	if err != nil {
		result.Error = fmt.Errorf("failed to commit changes: %w", err)
		return result
	}

	logrus.Info("Changes committed")

	// Handle dry-run mode
	if pb.dryRun {
		logrus.Warn("DRY RUN: Would create PR with these changes:")
		diff, _ := exec.RunCommand(workDir, "git", "diff", "HEAD~1")
		logrus.Info(diff)
		logrus.Warn("DRY RUN: Skipping push and PR creation")
		return result
	}

	// Push branch
	_, err = exec.RunCommand(workDir, "git", "push", pb.remote, prBranch)
	if err != nil {
		result.Error = fmt.Errorf("failed to push branch: %w", err)
		return result
	}

	// Create PR
	prURL, err := pb.createPullRequest(workDir, target.Repo, targetBranch, prBranch)
	if err != nil {
		result.Error = fmt.Errorf("failed to create PR: %w", err)
		return result
	}

	result.PRURL = strings.TrimSpace(prURL)
	return result
}

// executeUpdateScript runs the update script with environment variables
func (pb *PRBuilder) executeUpdateScript(workDir string, target *config.Target, targetBranch string) error {
	logrus.Info("Running update script...")
	updateScriptPath := filepath.Join(pb.sourceRepoDir, target.UpdateScript)

	if _, err := os.Stat(updateScriptPath); err != nil {
		return fmt.Errorf("update script not found: %s: %w", updateScriptPath, err)
	}

	// Make script executable
	if err := os.Chmod(updateScriptPath, 0755); err != nil {
		return fmt.Errorf("failed to make update script executable: %w", err)
	}

	// Set environment variables for the script
	env := os.Environ()
	env = append(env,
		"PRBUILDER_TAG="+pb.tag,
		"PRBUILDER_VERSION="+pb.version,
		"PRBUILDER_TARGET_DIR="+workDir,
		"PRBUILDER_TARGET_REPO="+target.Repo,
		"PRBUILDER_TARGET_BRANCH="+targetBranch,
		"PRBUILDER_SOURCE_DIR="+pb.sourceRepoDir,
	)

	logrus.Debugf("Environment variables: PRBUILDER_TAG=%s PRBUILDER_VERSION=%s PRBUILDER_TARGET_REPO=%s PRBUILDER_TARGET_BRANCH=%s",
		pb.tag, pb.version, target.Repo, targetBranch)

	// Run update script with environment variables
	output, err := exec.RunCommandWithEnv(workDir, updateScriptPath, env)
	if err != nil {
		return fmt.Errorf("update script failed: %w", err)
	}

	if len(output) > 0 {
		logrus.Debugf("Update script output: %s", output)
	}
	return nil
}

// createPullRequest creates a PR using the gh CLI
func (pb *PRBuilder) createPullRequest(workDir, repo, base, head string) (string, error) {
	title := fmt.Sprintf("Bump to %s", pb.tag)
	body := fmt.Sprintf(`Automated version bump to %s from upstream release.

This PR updates the dependencies to use the newly released version.

**Release tag:** %s
**Target branch:** %s

---
_This PR was automatically created by prbuilder_`, "`"+pb.tag+"`", pb.tag, base)

	prURL, err := exec.RunCommand(workDir, "gh", "pr", "create",
		"--base", base,
		"--head", head,
		"--title", title,
		"--body", body)

	if err != nil {
		return "", err
	}

	return prURL, nil
}

// WriteGitHubOutput writes the PR results to the GitHub Actions output file
func WriteGitHubOutput(results []PRResult) error {
	outputFile := os.Getenv("GITHUB_OUTPUT")
	if outputFile == "" {
		// Not running in GitHub Actions, skip
		return nil
	}

	// Collect successful PR URLs
	prURLs := make([]string, 0)
	for _, result := range results {
		if result.Error == nil && result.PRURL != "" {
			prURLs = append(prURLs, result.PRURL)
		}
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(prURLs)
	if err != nil {
		return fmt.Errorf("failed to marshal PR URLs to JSON: %w", err)
	}

	// Append to output file
	f, err := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open GITHUB_OUTPUT file: %w", err)
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "prs=%s\n", string(jsonData))
	if err != nil {
		return fmt.Errorf("failed to write to GITHUB_OUTPUT file: %w", err)
	}

	return nil
}
