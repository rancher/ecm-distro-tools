package prbuilder

import (
	"context"
	"fmt"

	"github.com/rancher/ecm-distro-tools/cmd/prbuilder/config"
	"github.com/rancher/ecm-distro-tools/cmd/prbuilder/git"
	"github.com/sirupsen/logrus"
)

type Builder struct {
	config        *config.Config
	version       string
	targetDir     string
	remote        string
	branchBuilder *BranchBuilder
	publisher     *Publisher
}

type Result struct {
	TargetRepo string
	PRURL      string
	Error      error
}

type Options struct {
	Config        *config.Config
	Tag           string
	SourceRepoDir string
	TargetDir     string
	Remote        string
	DryRun        bool
}

func NewPRBuilder(opts Options) (*Builder, error) {
	version, err := ParseVersion(opts.Tag, opts.Config.VersionStrategy)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version from tag %s: %w", opts.Tag, err)
	}

	remote := opts.Remote
	if remote == "" {
		remote = "origin"
	}

	componentOwner, componentRepo, componentName := extractComponentInfo(opts.SourceRepoDir)

	branchBuilder := NewBranchBuilder(opts.SourceRepoDir, opts.Tag, version, componentName)
	publisher := NewPublisher(remote, opts.Tag, componentName, componentOwner, componentRepo, opts.DryRun)

	return &Builder{
		config:        opts.Config,
		version:       version,
		targetDir:     opts.TargetDir,
		remote:        remote,
		branchBuilder: branchBuilder,
		publisher:     publisher,
	}, nil
}

func (b *Builder) ProcessTargets(ctx context.Context) ([]Result, error) {
	targets := b.config.Targets()
	results := make([]Result, 0)

	for _, target := range targets {
		branches, err := b.config.TargetBranches(b.version, &target)
		if err != nil {
			results = append(results, Result{
				TargetRepo: target.Repo,
				Error:      err,
			})
			continue
		}

		for _, branch := range branches {
			result := b.processTarget(ctx, &target, branch)
			results = append(results, result)
		}
	}

	return results, nil
}

func (b *Builder) processTarget(ctx context.Context, target *config.Target, targetBranch string) Result {
	result := Result{TargetRepo: target.Repo + " (" + targetBranch + ")"}

	branchResult := b.branchBuilder.BuildBranch(BranchOptions{
		Target:       target,
		TargetBranch: targetBranch,
		TargetDir:    b.targetDir,
		Remote:       b.remote,
	})

	defer branchResult.Cleanup()

	if branchResult.Error != nil {
		result.Error = branchResult.Error
		return result
	}

	if !branchResult.HasChanges {
		return result
	}

	publishResult := b.publisher.Publish(ctx, PublishOptions{
		BranchResult: branchResult,
		TargetRepo:   target.Repo,
		TargetBranch: targetBranch,
	})

	if publishResult.Error != nil {
		result.Error = publishResult.Error
		return result
	}

	result.PRURL = publishResult.PRURL
	return result
}

func extractComponentInfo(sourceRepoDir string) (owner, repo, name string) {
	owner, repo, name = "", "", "component"

	if sourceRepoDir == "" {
		return
	}

	sourceRepo, err := git.Open(sourceRepoDir)
	if err != nil {
		logrus.Warnf("Failed to open source repository for component extraction: %v", err)
		return
	}

	remoteURL, err := sourceRepo.GetRemoteURL("origin")
	if err != nil {
		logrus.Warnf("Failed to get remote 'origin' from source repository: %v", err)
		return
	}

	extractedOwner, extractedRepo, err := git.ExtractOwnerRepo(remoteURL)
	if err != nil {
		logrus.Warnf("Failed to extract owner/repo from remote URL %s: %v", remoteURL, err)
		return
	}

	owner = extractedOwner
	repo = extractedRepo
	name = extractedRepo
	logrus.Debugf("Extracted component information: owner=%s, repo=%s, name=%s", owner, repo, name)
	return
}
