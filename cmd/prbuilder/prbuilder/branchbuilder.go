package prbuilder

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rancher/ecm-distro-tools/cmd/prbuilder/config"
	"github.com/rancher/ecm-distro-tools/cmd/prbuilder/git"
	"github.com/rancher/ecm-distro-tools/exec"
	"github.com/sirupsen/logrus"
)

// BranchBuilder handles local repository operations:
// cloning/opening repos, creating branches, running update scripts, and committing changes.
// It does NOT interact with GitHub APIs or push to remotes.
type BranchBuilder struct {
	sourceRepoDir string
	tag           string
	version       string
	componentName string
}

type BranchOptions struct {
	Target       *config.Target
	TargetBranch string
	TargetDir    string // Optional: for single-target mode with existing clone
	Remote       string
}

type BranchResult struct {
	WorkDir    string
	Branch     string
	HasChanges bool
	TempDir    bool
	Error      error
}

type LocalRepo struct {
	Repo    *git.Repository
	WorkDir string
	IsTemp  bool
}

func NewBranchBuilder(sourceRepoDir, tag, version, componentName string) *BranchBuilder {
	return &BranchBuilder{
		sourceRepoDir: sourceRepoDir,
		tag:           tag,
		version:       version,
		componentName: componentName,
	}
}

func (bb *BranchBuilder) BuildBranch(opts BranchOptions) *BranchResult {
	result := &BranchResult{}

	repoInfo, err := bb.getRepository(opts)
	if err != nil {
		result.Error = err
		return result
	}

	result.WorkDir = repoInfo.WorkDir
	result.TempDir = repoInfo.IsTemp

	branchName := bb.generateBranchName(opts.TargetBranch)
	if err := repoInfo.Repo.CreateBranch(branchName); err != nil {
		result.Error = fmt.Errorf("failed to create branch %s: %w", branchName, err)
		return result
	}
	result.Branch = branchName

	if err := bb.executeUpdateScript(repoInfo.WorkDir, opts.Target, opts.TargetBranch); err != nil {
		result.Error = err
		return result
	}

	hasChanges, err := repoInfo.Repo.HasChanges()
	if err != nil {
		result.Error = fmt.Errorf("failed to check git status: %w", err)
		return result
	}

	result.HasChanges = hasChanges

	if !hasChanges {
		logrus.Infof("No changes detected for %s on branch %s - skipping commit", opts.Target.Repo, opts.TargetBranch)
		if !repoInfo.IsTemp {
			if err := repoInfo.Repo.CheckoutBranch(opts.TargetBranch); err != nil {
				logrus.Warnf("Failed to checkout back to %s: %v", opts.TargetBranch, err)
			} else if err := repoInfo.Repo.DeleteBranch(branchName); err != nil {
				logrus.Warnf("Failed to delete unused branch %s: %v", branchName, err)
			} else {
				logrus.Debugf("Cleaned up unused branch %s", branchName)
			}
		}
		return result
	}

	if err := repoInfo.Repo.AddAll(); err != nil {
		result.Error = fmt.Errorf("failed to stage changes: %w", err)
		return result
	}

	commitMsg := fmt.Sprintf("Bump to %s", bb.tag)
	commitBody := "Automated version bump from upstream release"
	if err := repoInfo.Repo.Commit(git.CommitOptions{
		Message:     commitMsg + "\n\n" + commitBody,
		AuthorName:  "github-actions[bot]",
		AuthorEmail: "github-actions[bot]@users.noreply.github.com",
	}); err != nil {
		result.Error = fmt.Errorf("failed to commit changes: %w", err)
		return result
	}

	logrus.Infof("Created branch %s with changes for %s", branchName, opts.Target.Repo)
	return result
}

func (bb *BranchBuilder) getRepository(opts BranchOptions) (LocalRepo, error) {
	if opts.TargetDir != "" {
		// Single-target mode: use existing repository
		repo, err := git.Open(opts.TargetDir)
		if err != nil {
			return LocalRepo{
				nil, "", false,
			}, fmt.Errorf("failed to open repository: %w", err)
		}

		// Ensure remote exists
		repoURL := "https://github.com/" + opts.Target.Repo + ".git"
		if err := repo.EnsureRemote(opts.Remote, repoURL); err != nil {
			return LocalRepo{
				nil, "", false,
			}, fmt.Errorf("failed to ensure remote %s: %w", opts.Remote, err)
		}

		// Fetch latest from remote (mandatory - prevents stale branch issues)
		if err := repo.Fetch(opts.Remote); err != nil {
			return LocalRepo{
				nil, "", false,
			}, fmt.Errorf("failed to fetch from %s: %w", opts.Remote, err)
		}

		// Checkout remote branch, creating/resetting local branch
		if err := repo.CheckoutRemoteBranch(opts.Remote, opts.TargetBranch, opts.TargetBranch); err != nil {
			return LocalRepo{
				nil, "", false,
			}, fmt.Errorf("failed to checkout %s from %s/%s: %w", opts.TargetBranch, opts.Remote, opts.TargetBranch, err)
		}

		return LocalRepo{repo, opts.TargetDir, false}, nil
	}

	// Multi-target mode: create temp directory and clone
	workDir, err := os.MkdirTemp("", "prbuilder-*")
	if err != nil {
		return LocalRepo{
			nil, "", false,
		}, fmt.Errorf("failed to create temp directory: %w", err)
	}

	repoURL := "https://github.com/" + opts.Target.Repo + ".git"
	repo, err := git.Clone(repoURL, workDir, opts.TargetBranch, 1) // depth=1 for shallow clone
	if err != nil {
		if rerr := os.RemoveAll(workDir); rerr != nil {
			logrus.Warnf("Failed to clean up temp directory %s: %v", workDir, rerr)
		}
		return LocalRepo{
			nil, "", false,
		}, fmt.Errorf("failed to clone %s: %w", opts.Target.Repo, err)
	}

	return LocalRepo{repo, workDir, true}, nil
}

func (bb *BranchBuilder) generateBranchName(targetBranch string) string {
	return fmt.Sprintf("bot/%s-%s-bump-%s-%d", targetBranch, bb.componentName, bb.tag, time.Now().Unix())
}

// executeUpdateScript runs the update script with environment variables
func (bb *BranchBuilder) executeUpdateScript(workDir string, target *config.Target, targetBranch string) error {
	updateScriptPath := filepath.Join(bb.sourceRepoDir, target.UpdateScriptPath)

	if _, err := os.Stat(updateScriptPath); err != nil {
		return fmt.Errorf("update script not found: %s: %w", updateScriptPath, err)
	}

	if err := os.Chmod(updateScriptPath, 0755); err != nil {
		return fmt.Errorf("failed to make update script executable: %w", err)
	}

	env := os.Environ()
	env = append(env,
		"PRBUILDER_TAG="+bb.tag,
		"PRBUILDER_VERSION="+bb.version,
		"PRBUILDER_TARGET_DIR="+workDir,
		"PRBUILDER_TARGET_REPO="+target.Repo,
		"PRBUILDER_TARGET_BRANCH="+targetBranch,
		"PRBUILDER_SOURCE_DIR="+bb.sourceRepoDir,
	)

	logrus.Debugf("Environment variables: PRBUILDER_TAG=%s PRBUILDER_VERSION=%s PRBUILDER_TARGET_REPO=%s PRBUILDER_TARGET_BRANCH=%s",
		bb.tag, bb.version, target.Repo, targetBranch)

	output, err := exec.RunCommandWithEnv(workDir, updateScriptPath, env)

	if err != nil {
		// On error, the error contains stderr from the script
		// Log both for debugging context
		logrus.Errorf("Update script failed for %s", target.Repo)
		if len(output) > 0 {
			logrus.Errorf("Script stdout:\n%s", output)
		}
		logrus.Errorf("Script stderr: %v", err)
		return fmt.Errorf("update script failed for %s: %w", target.Repo, err)
	}

	// On success, log output if present
	if len(output) > 0 {
		logrus.Debugf("Update script output for %s:\n%s", target.Repo, output)
	}

	return nil
}

// Cleanup removes the working directory if it's a temp directory
func (br *BranchResult) Cleanup() {
	if br.TempDir && br.WorkDir != "" {
		if err := os.RemoveAll(br.WorkDir); err != nil {
			logrus.Warnf("Failed to clean up temp directory %s: %v", br.WorkDir, err)
		}
	}
}
