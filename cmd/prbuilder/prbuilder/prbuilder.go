package prbuilder

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-github/v85/github"
	"github.com/rancher/ecm-distro-tools/cmd/prbuilder/config"
	"github.com/rancher/ecm-distro-tools/exec"
	"github.com/rancher/ecm-distro-tools/repository"
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
	results := make([]PRResult, 0)

	for _, target := range targets {
		// Get target branches (may be multiple)
		branches, err := pb.config.GetTargetBranches(pb.version, &target)
		if err != nil {
			result := PRResult{
				TargetRepo: target.Repo,
				Error:      err,
			}
			results = append(results, result)
			continue
		}

		// Process each branch
		for _, branch := range branches {
			result := pb.processTarget(ctx, &target, branch)
			results = append(results, result)
		}
	}

	return results, nil
}

func (pb *PRBuilder) processTarget(ctx context.Context, target *config.Target, targetBranch string) PRResult {
	result := PRResult{TargetRepo: target.Repo + " (" + targetBranch + ")"}

	var workDir string
	var cleanupDir bool
	var err error

	if pb.targetDir != "" {
		workDir = pb.targetDir
		cleanupDir = false

		_, err = exec.RunCommand(workDir, "git", "checkout", targetBranch)
		if err != nil {
			result.Error = fmt.Errorf("failed to checkout branch %s: %w", targetBranch, err)
			return result
		}

		_, err = exec.RunCommand(workDir, "git", "pull", pb.remote, targetBranch)
		if err != nil {
			logrus.Debugf("failed to pull latest changes: %v (continuing anyway)", err)
		}
	} else {
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

		_, err = exec.RunCommand(workDir, "gh", "repo", "clone", target.Repo, ".", "--", "--depth=1", fmt.Sprintf("--branch=%s", targetBranch))
		if err != nil {
			result.Error = fmt.Errorf("failed to clone %s on branch %s: %w", target.Repo, targetBranch, err)
			return result
		}
	}

	prBranch := fmt.Sprintf("bump-to-%s-%d", pb.tag, time.Now().Unix())
	_, err = exec.RunCommand(workDir, "git", "checkout", "-b", prBranch)
	if err != nil {
		result.Error = fmt.Errorf("failed to create branch %s: %w", prBranch, err)
		return result
	}

	if err := pb.executeUpdateScript(workDir, target, targetBranch); err != nil {
		result.Error = err
		return result
	}

	status, err := exec.RunCommand(workDir, "git", "status", "--porcelain")
	if err != nil {
		result.Error = fmt.Errorf("failed to check git status: %w", err)
		return result
	}

	if strings.TrimSpace(status) == "" {
		return result
	}

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

	if pb.dryRun {
		return result
	}

	_, err = exec.RunCommand(workDir, "git", "push", pb.remote, prBranch)
	if err != nil {
		result.Error = fmt.Errorf("failed to push branch: %w", err)
		return result
	}

	prURL, err := pb.createPullRequest(ctx, target.Repo, targetBranch, prBranch)
	if err != nil {
		result.Error = fmt.Errorf("failed to create PR: %w", err)
		return result
	}

	result.PRURL = prURL
	return result
}

// executeUpdateScript runs the update script with environment variables
func (pb *PRBuilder) executeUpdateScript(workDir string, target *config.Target, targetBranch string) error {
	updateScriptPath := filepath.Join(pb.sourceRepoDir, target.UpdateScript)

	if _, err := os.Stat(updateScriptPath); err != nil {
		return fmt.Errorf("update script not found: %s: %w", updateScriptPath, err)
	}

	if err := os.Chmod(updateScriptPath, 0755); err != nil {
		return fmt.Errorf("failed to make update script executable: %w", err)
	}

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

	output, err := exec.RunCommandWithEnv(workDir, updateScriptPath, env)
	if err != nil {
		return fmt.Errorf("update script failed: %w", err)
	}

	if len(output) > 0 {
		logrus.Debugf("Update script output: %s", output)
	}
	return nil
}

// createPullRequest creates a PR using the GitHub API
func (pb *PRBuilder) createPullRequest(ctx context.Context, repo, base, head string) (string, error) {
	// Parse owner/repo from "owner/repo" format
	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid repo format %s, expected owner/repo", repo)
	}
	owner, repoName := parts[0], parts[1]

	// Get GitHub token from environment
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return "", fmt.Errorf("GITHUB_TOKEN environment variable is required")
	}

	// Create GitHub client
	ghClient := repository.NewGithub(ctx, token)

	title := fmt.Sprintf("Bump to %s", pb.tag)
	body := fmt.Sprintf(`Automated version bump to %s from upstream release.

This PR updates the dependencies to use the newly released version.

**Release tag:** %s
**Target branch:** %s

---
_This PR was automatically created by prbuilder_`, "`"+pb.tag+"`", pb.tag, base)

	// Create pull request
	pr := &github.NewPullRequest{
		Title:               new(title),
		Base:                new(base),
		Head:                new(head),
		Body:                new(body),
		MaintainerCanModify: new(true),
	}

	createdPR, _, err := ghClient.PullRequests.Create(ctx, owner, repoName, pr)
	if err != nil {
		return "", fmt.Errorf("failed to create pull request: %w", err)
	}

	return createdPR.GetHTMLURL(), nil
}

// WriteGitHubOutput writes the PR results to the GitHub Actions output file
func WriteGitHubOutput(results []PRResult) error {
	outputFile := os.Getenv("GITHUB_OUTPUT")
	if outputFile == "" {
		return nil
	}

	prURLs := make([]string, 0)
	for _, result := range results {
		if result.Error == nil && result.PRURL != "" {
			prURLs = append(prURLs, result.PRURL)
		}
	}

	b, err := json.Marshal(prURLs)
	if err != nil {
		return fmt.Errorf("failed to marshal PR URLs to JSON: %w", err)
	}

	f, err := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open GITHUB_OUTPUT file: %w", err)
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "prs=%s\n", string(b))
	if err != nil {
		return fmt.Errorf("failed to write to GITHUB_OUTPUT file: %w", err)
	}

	return nil
}
