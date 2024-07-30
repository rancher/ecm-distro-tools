package charts

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/rancher/ecm-distro-tools/cmd/release/config"
)

// List prints the lifecycle status of the charts
func List(ctx context.Context, c *config.ChartsRelease, branch, chart string) (string, error) {
	var branchArg, chartArg string

	branchArg = "--branch-version=" + branch
	if chart != "" {
		chartArg = "--chart=" + chart
	}

	output, err := runChartsBuildScripts(c.Workspace, "lifecycle-status", branchArg, chartArg)
	if err != nil {
		return "", err
	}

	response := string(output) + fmt.Sprintf("\ngenerated log files for inspection at: \n%s\n", c.Workspace+"/logs/")
	return response, nil
}

func runChartsBuildScripts(chartsRepoPath string, args ...string) ([]byte, error) {
	// save current working dir
	ecmWorkDir, err := os.Getwd()
	if err != nil {
		return []byte{}, err
	}

	// change working dir to the charts repo
	if err := os.Chdir(chartsRepoPath); err != nil {
		return []byte{}, err
	}

	bin := strings.Join([]string{chartsRepoPath, "bin", "charts-build-scripts"}, string(os.PathSeparator))

	cmd := exec.Command(bin, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return []byte{}, err
	}

	// Change back working dir for the caller
	if err := os.Chdir(ecmWorkDir); err != nil {
		return []byte{}, err
	}

	return output, nil
}
