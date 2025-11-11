package cli

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/go-github/v77/github"
	ecmExec "github.com/rancher/ecm-distro-tools/exec"
	"github.com/rancher/ecm-distro-tools/release"
	"github.com/rancher/ecm-distro-tools/repository"
	"golang.org/x/mod/semver"
)

const (
	cliOrg           = "rancher"
	cliRepo          = "cli"
	cliImagesBaseURL = "https://github.com/" + cliOrg + "/" + cliRepo + "/releases"
)

// CreateRelease will create a new tag and a new release with given params.
func CreateRelease(ctx context.Context, client *github.Client, opts *repository.CreateReleaseOpts, rc bool, releaseType, previousTag string, dryRun bool) error {
	if !semver.IsValid(opts.Tag) {
		return errors.New("tag isn't a valid semver: " + opts.Tag)
	}

	latestPreRelease, err := release.LatestPreRelease(ctx, client, opts.Owner, opts.Repo, opts.Tag, releaseType)
	if err != nil {
		return err
	}

	opts.Name = opts.Tag
	opts.Prerelease = true
	opts.ReleaseNotes = ""

	if rc {
		latestRCNumber := 1
		if latestPreRelease != nil {
			// v2.9.0-rc.N / -alpha.N
			_, trimmedRCNumber, found := strings.Cut(*latestPreRelease, "-"+releaseType+".")
			if !found {
				return errors.New("failed to parse rc number from " + *latestPreRelease)
			}
			currentRCNumber, err := strconv.Atoi(trimmedRCNumber)
			if err != nil {
				return err
			}
			latestRCNumber = currentRCNumber + 1
		}
		opts.Tag = fmt.Sprintf("%s-%s.%d", opts.Tag, releaseType, latestRCNumber)
	} else {
		fmt.Printf("release.GenReleaseNotes(ctx, %s, %s, %s, %s, client)", opts.Owner, opts.Repo, opts.Branch, previousTag)
		buff, err := release.GenReleaseNotes(ctx, opts.Owner, opts.Repo, opts.Branch, previousTag, client)
		if err != nil {
			return err
		}
		opts.ReleaseNotes = buff.String()
	}

	fmt.Printf("create release options: %+v\n", *opts)

	if dryRun {
		fmt.Println("dry run, skipping creating release")
		return nil
	}

	createdRelease, err := repository.CreateRelease(ctx, client, opts)
	if err != nil {
		return err
	}

	fmt.Println("release created: " + *createdRelease.HTMLURL)
	return nil
}

func ReleaseBranchFromTag(tag string) (string, error) {
	majorMinor := semver.MajorMinor(tag)

	if majorMinor == "" {
		return "", errors.New("the tag isn't a valid semver: " + tag)
	}

	return majorMinor, nil
}

func UpdateRancherReferences(ctx context.Context, ghClient *github.Client, tag, rancherRepoName, rancherRepoOwner, rancherUpstreamURL, cliReleaseBranch, cliRepoName, githubUsername string, dryRun bool) error {
	commitSHA, err := getRancherPkgSHA(ctx, ghClient, rancherRepoOwner, rancherRepoName, tag)
	if err != nil {
		return err
	}

	if err := updateRancherReferencesAndPush(tag, cliReleaseBranch, commitSHA, dryRun); err != nil {
		return err
	}

	return createCLIReferencesPR(ctx, ghClient, tag, cliReleaseBranch, cliRepoName, rancherRepoOwner, githubUsername)
}

func getRancherPkgSHA(ctx context.Context, ghClient *github.Client, owner, repo, tag string) (string, error) {
	ref, _, err := ghClient.Git.GetRef(ctx, owner, repo, "tags/"+tag)
	if err != nil {
		return "", fmt.Errorf("error getting tag reference: %v", err)
	}

	if ref.Object.GetType() == "commit" {
		return ref.Object.GetSHA(), nil
	}

	if ref.Object.GetType() == "tag" {
		tagObj, _, err := ghClient.Git.GetTag(ctx, owner, repo, ref.Object.GetSHA())
		if err != nil {
			return "", fmt.Errorf("error getting tag object: %v", err)
		}
		return tagObj.Object.GetSHA(), nil
	}

	return "", fmt.Errorf("unexpected reference type: %s", ref.Object.GetType())
}

func UpdateCLIRefsBranchName(tag string) string {
	return "update-cli-build-refs-" + tag
}

func updateRancherReferencesAndPush(tag, releaseBranch, rancherCommitSHA string, dryRun bool) error {
	updateScriptVars := map[string]string{
		"Tag":              tag,
		"ReleaseBranch":    releaseBranch,
		"RancherCommitSHA": rancherCommitSHA,
		"DryRun":           strconv.FormatBool(dryRun),
		"BranchName":       UpdateCLIRefsBranchName(tag),
	}

	fmt.Println("creating update cli references script template")
	updateScriptOut, err := ecmExec.RunTemplatedScript("./", "replace_cli_ref.sh", updateRancherReferencesScript, nil, updateScriptVars)
	if err != nil {
		fmt.Println("error executing script")
		return err
	}
	fmt.Println(updateScriptOut)
	return nil
}

func createCLIReferencesPR(ctx context.Context, ghClient *github.Client, tag, releaseBranch, cliRepoName, rancherRepoOwner, githubUsername string) error {
	pull := &github.NewPullRequest{
		Title:               github.String("Bump Rancher version to " + tag),
		Base:                github.String(releaseBranch),
		Head:                github.String(githubUsername + ":" + UpdateCLIRefsBranchName(tag)),
		MaintainerCanModify: github.Bool(true),
	}

	// creating a pr from your fork branch
	pr, _, err := ghClient.PullRequests.Create(ctx, rancherRepoOwner, cliRepoName, pull)
	if err != nil {
		return err
	}

	fmt.Println("Pull Request created successfully:", pr.GetHTMLURL())

	return nil
}

const updateRancherReferencesScript = `#!/bin/sh
set -ex
OS=$(uname -s)
# Set variables (these are populated by Go's template engine)
DRY_RUN="{{ .DryRun }}"
TAG="{{ .Tag }}"
BRANCH_NAME="{{ .BranchName }}"
RELEASE_BRANCH="{{ .ReleaseBranch }}"
RANCHER_COMMIT_SHA="{{ .RancherCommitSHA }}"
UPSTREAM_URL="{{ .CLIUpstreamURL }}"

# Add upstream remote if it doesn't exist
# Note: Using ls | grep is not recommended for general use, but it's okay here
# since we're only checking for 'rancher'
git remote -v | grep -w upstream || git remote add upstream "$UPSTREAM_URL"
git fetch upstream
git stash

# Delete the branch if it exists, then create a new one based on upstream
git branch -D "${BRANCH_NAME}" > /dev/null 2>&1 || true
git checkout -B "${BRANCH_NAME}" "upstream/$RELEASE_BRANCH"
# git clean -xfd

# Function to update the file
update_go_mod() {
	echo "Updating pkg/apis module..."
	go get "github.com/rancher/rancher/pkg/apis@$RANCHER_COMMIT_SHA"
	sleep 2

	echo "Updating pkg/client module..."
	go get "github.com/rancher/rancher/pkg/client@$RANCHER_COMMIT_SHA"

	sleep 2
	go mod tidy
}

# Run the update function
update_go_mod

git add go.mod go.sum

# Cleaning temp files/scripts
git clean -f

git commit --signoff -m "Update Rancher refs to ${RANCHER_VERSION}"

# Push the changes if not a dry run
if [ "${DRY_RUN}" = false ]; then
	git push --set-upstream origin "${BRANCH_NAME}" # run git remote -v for your origin
fi`
