package charts

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	ecmConfig "github.com/rancher/ecm-distro-tools/cmd/release/config"
)

// ListLifecycleStatus prints the lifecycle status of the charts
func ListLifecycleStatus(ctx context.Context, c *ecmConfig.ChartsRelease, branch, chart string) error {

	var branchArg, chartArg string = "", ""

	branchArg = "--branch-version=" + branch
	if chart != "" {
		chartArg = "--chart=" + chart
	}

	err := executeChartsBuildScripts(c.Workspace, "lifecycle-status", branchArg, chartArg)
	if err != nil {
		return err
	}

	fmt.Printf("generated log files for inspection at: \n%s\n", c.Workspace+"/logs/")
	return nil
}

func executeChartsBuildScripts(chartsRepoPath string, args ...string) error {
	// save current working dir
	ecmWorkDir, err := os.Getwd()
	if err != nil {
		return err
	}

	// change working dir to the charts repo
	err = os.Chdir(chartsRepoPath)
	if err != nil {
		return err
	}

	bin := strings.Join([]string{chartsRepoPath, "bin", "charts-build-scripts"}, string(os.PathSeparator))

	cmd := exec.Command(bin, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to execute charts-build-scripts: %w, output: %s", err, output)
	}
	fmt.Printf("charts-build-scripts output: %s\n", output)

	// Change back working dir for the caller
	err = os.Chdir(ecmWorkDir)
	if err != nil {
		return err
	}

	return nil
}
