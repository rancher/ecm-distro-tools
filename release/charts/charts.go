package charts

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/rancher/ecm-distro-tools/cmd/release/config"
)

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
		return output, err
	}

	// Change back working dir for the caller
	if err := os.Chdir(ecmWorkDir); err != nil {
		return output, err
	}

	return output, nil
}
