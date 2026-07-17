package prbuilder

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/v85/github"
	"github.com/rancher/ecm-distro-tools/cmd/prbuilder/git"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/sirupsen/logrus"
)

type Publisher struct {
	remote         string
	tag            string
	componentName  string
	componentOwner string
	componentRepo  string
	dryRun         bool
}

type PublishOptions struct {
	BranchResult *BranchResult
	TargetRepo   string
	TargetBranch string
}

type PublishResult struct {
	PRURL string
	Error error
}

func NewPublisher(remote, tag, componentName, componentOwner, componentRepo string, dryRun bool) *Publisher {
	return &Publisher{
		remote:         remote,
		tag:            tag,
		componentName:  componentName,
		componentOwner: componentOwner,
		componentRepo:  componentRepo,
		dryRun:         dryRun,
	}
}

func (p *Publisher) Publish(ctx context.Context, opts PublishOptions) *PublishResult {
	result := &PublishResult{}

	if !opts.BranchResult.HasChanges {
		logrus.Debugf("Skipping publish for %s - no changes", opts.TargetRepo)
		return result
	}

	if p.dryRun {
		logrus.Infof("[DRY RUN] Would push branch %s and create PR in %s targeting %s",
			opts.BranchResult.Branch, opts.TargetRepo, opts.TargetBranch)
		return result
	}

	if err := p.pushBranch(opts); err != nil {
		result.Error = err
		return result
	}

	prURL, err := p.createPullRequest(ctx, opts)
	if err != nil {
		result.Error = err
		return result
	}

	result.PRURL = prURL
	logrus.Infof("Created PR: %s", prURL)
	return result
}

func (p *Publisher) pushBranch(opts PublishOptions) error {
	repo, err := git.Open(opts.BranchResult.WorkDir)
	if err != nil {
		return fmt.Errorf("failed to open repository for push: %w", err)
	}

	if err := repo.Push(p.remote); err != nil {
		return fmt.Errorf("failed to push branch: %w", err)
	}

	logrus.Debugf("Pushed branch %s to %s", opts.BranchResult.Branch, p.remote)
	return nil
}

func (p *Publisher) createPullRequest(ctx context.Context, opts PublishOptions) (string, error) {
	parts := strings.Split(opts.TargetRepo, "/")
	if len(parts) != 2 {
		return "", errors.New("invalid repo format " + opts.TargetRepo + ", expected owner/repo")
	}
	owner, repoName := parts[0], parts[1]

	token := getGitHubToken()
	if token == "" {
		return "", errors.New("GH_TOKEN or GITHUB_TOKEN environment variable is required")
	}

	ghClient := repository.NewGithub(ctx, token)

	title := p.generatePRTitle(opts.TargetBranch)
	body := p.generatePRBody(opts.TargetBranch)

	maintainerCanModify := true
	pr := &github.NewPullRequest{
		Title:               &title,
		Base:                &opts.TargetBranch,
		Head:                &opts.BranchResult.Branch,
		Body:                &body,
		MaintainerCanModify: &maintainerCanModify,
	}

	createdPR, _, err := ghClient.PullRequests.Create(ctx, owner, repoName, pr)
	if err != nil {
		return "", fmt.Errorf("failed to create pull request: %w", err)
	}

	return createdPR.GetHTMLURL(), nil
}

// generatePRTitle creates the PR title in format: [{target-branch}] Bump {component} to {tag}
func (p *Publisher) generatePRTitle(targetBranch string) string {
	return fmt.Sprintf("[%s] Bump %s to %s", targetBranch, p.componentName, p.tag)
}

func (p *Publisher) generatePRBody(targetBranch string) string {
	bodyParts := []string{
		fmt.Sprintf("Automated version bump to %s from upstream release.", "`"+p.tag+"`"),
		"",
		"This PR updates the dependencies to use the newly released version.",
		"",
		fmt.Sprintf("**Component:** %s", p.componentName),
		fmt.Sprintf("**Release tag:** %s", p.tag),
		fmt.Sprintf("**Target branch:** %s", targetBranch),
	}

	if p.componentOwner != "" && p.componentRepo != "" {
		releaseURL := fmt.Sprintf("https://github.com/%s/%s/releases/tag/%s",
			p.componentOwner, p.componentRepo, p.tag)
		bodyParts = append(bodyParts, fmt.Sprintf("**Release notes:** %s", releaseURL))
	}

	bodyParts = append(bodyParts, "", "---", "_This PR was automatically created by prbuilder_")
	return strings.Join(bodyParts, "\n")
}

func getGitHubToken() string {
	token := os.Getenv("GH_TOKEN")
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	return token
}
