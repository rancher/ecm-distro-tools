package charts

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/google/go-github/v82/github"
	"github.com/rancher/ecm-distro-tools/cmd/release/config"
	ecmExec "github.com/rancher/ecm-distro-tools/exec"
	"github.com/rancher/ecm-distro-tools/repository"
)

// chartsReleasePRBody is the default PR body for the charts release PR
const chartsReleasePRBody = `
## Charts Checklist (built for v0.9.X charts-build-scripts)

### Checkpoint 0: Validate **release.yaml**

Validation steps:
- [ ] Each chart version in **release.yaml** DOES NOT modify an already released chart. If so, stop and modify the versions so that it releases a net-new chart.
- [ ] Each chart version in **release.yaml** IS exactly 1 more patch version than the last released chart version. If not, stop and modify the versions so that it releases a net-new chart.

### Checkpoint 1: Compare contents of assets/ to charts/

Validation steps:
- [ ] Running **make unzip** to regenerate the **charts/** from scratch, then **git diff** to check differences between **assets/** and **charts/** yields NO differences or innocuous differences.

IMPORTANT: Do not undo these changes for future steps since we want to keep the charts/ that match the current contents of assets!

### Checkpoint 2: Compare assets against index.yaml

Validation steps:
- [ ] The **index.yaml** file has an entry for each chart version.
- [ ] The **index.yaml** entries for each chart matches the **Chart.yaml** for each chart.
- [ ] Each chart has ALL required annotations
- kube-version annotation
- rancher-version annotation
- permits-os annotation (indicates Windows and/or Linux)
`

// List prints the lifecycle status of the charts
func List(ctx context.Context, c *config.ChartsRelease, branch, chart string) (string, error) {
	var branchArg, chartArg string

	branchArg = "--branch-version=" + branch
	if chart != "" {
		chartArg = "--chart=" + chart
	}

	output, err := runChartsBuild(c.Workspace, "lifecycle-status", branchArg, chartArg)
	if err != nil {
		return "", err
	}

	response := string(output) + fmt.Sprintf("\ngenerated log files for inspection at: \n%s\n", c.Workspace+"/logs/")
	return response, nil
}

// Update will pull the target chart version to the local branch and create a PR to release the chart
func Update(ctx context.Context, c *config.ChartsRelease, br, ch, vr string) (string, error) {
	var branchArg, chartArg, versionArg, forkArg string

	branchArg = "--branch-version=" + br
	chartArg = "--chart=" + ch
	versionArg = "--version=" + vr
	forkArg = "--fork=" + c.ChartsForkURL

	output, err := runChartsBuild(c.Workspace, "release", branchArg, chartArg, versionArg, forkArg)
	if err != nil {
		return string(output), err
	}

	r, err := git.PlainOpen(c.Workspace)
	if err != nil {
		return string(output), err
	}

	wt, err := r.Worktree()
	if err != nil {
		return string(output), err
	}

	if err := wt.AddWithOptions(&git.AddOptions{All: true}); err != nil {
		return string(output), err
	}

	commitMsg := "release chart: " + ch + " - version: " + vr
	if _, err := wt.Commit(commitMsg, &git.CommitOptions{All: true}); err != nil {
		return string(output), err
	}

	return string(output), nil
}

// Push will push the charts updates to the remote upstream charts repository and create a PR.
func Push(ctx context.Context, conf *config.ChartsRelease, user *config.User, ghc *github.Client, branch, token string, debug bool) (string, error) {
	const repoOwner = "rancher"
	const repoName = "charts"

	r, err := git.PlainOpen(conf.Workspace)
	if err != nil {
		return "", err
	}

	remote, err := repository.UpstreamRemote(r, conf.ChartsRepoURL)
	if err != nil {
		return "", err
	}

	h, err := r.Head()
	if err != nil {
		return "", err
	}

	// create a new PR
	pr := &github.NewPullRequest{
		Title:               github.String("[" + branch + "] batch release"),
		Base:                github.String(branch),
		Head:                github.String(h.Name().Short()),
		Body:                github.String(chartsReleasePRBody),
		MaintainerCanModify: github.Bool(true),
	}

	// debug mode
	if debug {
		if err := debugPullRequest(r, remote, branch); err != nil {
			return "", err
		}
	}

	if err := repository.PushRemoteBranch(r, remote, user.GithubUsername, token, debug); err != nil {
		return "", err
	}

	prResp, _, err := ghc.PullRequests.Create(ctx, repoOwner, repoName, pr)
	if err != nil {
		return "", err
	}

	return prResp.GetHTMLURL(), nil
}

func runChartsBuild(chartsRepoPath string, args ...string) ([]byte, error) {
	// save current working dir
	ecmWorkDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// change working dir to the charts repo
	if err := os.Chdir(chartsRepoPath); err != nil {
		return nil, err
	}

	bin := strings.Join([]string{chartsRepoPath, "bin", "charts-build-scripts"}, string(os.PathSeparator))

	cmd := exec.Command(bin, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.New(err.Error() + ": " + string(output))
	}

	// Change back working dir for the caller
	if err := os.Chdir(ecmWorkDir); err != nil {
		return nil, errors.New(err.Error() + ": " + string(output))
	}

	return output, nil
}

// debugPullRequest will prompt the user to check the files and commits that will be pushed to the remote repository
func debugPullRequest(r *git.Repository, remote, branch string) error {
	if execute := ecmExec.UserInput("Check files that will be pushed?"); execute {
		// commit history
		iter, err := r.Log(&git.LogOptions{})
		if err != nil {
			return err
		}

		err = iter.ForEach(func(c *object.Commit) error {
			fileStats, err := c.Stats()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			fmt.Println(c)
			fmt.Println(fileStats)

			if execute = ecmExec.UserInput("Check next files that will be pushed?"); !execute {
				return storer.ErrStop
			}

			return nil
		})
		if err != nil {
			return err
		}
	}

	if execute := ecmExec.UserInput("Check commits that will be pushed?"); execute {
		if err := repository.DiffLocalToRemote(r, remote, branch); err != nil {
			return err
		}
	}

	if execute := ecmExec.UserInput("Push and Create PR to the remote repository?"); !execute {
		return errors.New("user aborted the push")
	}

	return nil
}
