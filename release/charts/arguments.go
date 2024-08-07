package charts

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/rancher/ecm-distro-tools/cmd/release/config"
)

type status struct {
	ToRelease     map[string][]asset `json:"to_be_released"`
	ToForwardPort map[string][]asset `json:"to_be_forward_ported"`
}

type asset struct {
	Version string `json:"version"`
}

// loadState will load the lifecycle-status state from an existing state.json file at charts repo
func loadState(filePath string) (*status, error) {
	s := &status{}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}

	return s, nil
}

// BranchArgs will return the list of available branch version lines
func BranchArgs() []string {
	return []string{"2.9", "2.8"}
}

// ChartArgs will return the list of available charts in the current branch
func ChartArgs(ctx context.Context, c *config.ChartsRelease) ([]string, error) {
	assets := filepath.Join(c.Workspace, "assets")

	// Read the directory assets contents
	files, err := os.ReadDir(assets)
	if err != nil {
		return nil, err
	}

	var dirs []string
	for _, file := range files {
		if file.IsDir() {
			dirs = append(dirs, file.Name())
		}
	}

	// Sort alphabetically
	sort.Strings(dirs)
	return dirs, nil
}

// VersionArgs will return the list of available versions for the target chart
func VersionArgs(ctx context.Context, c *config.ChartsRelease, ch string) ([]string, error) {
	status, err := loadState(filepath.Join(c.Workspace, "state.json"))
	if err != nil {
		return nil, err
	}

	versionsRelease, foundVersionsToRelease := status.ToRelease[ch]
	versionsForwardPort, foundVersionsToForwardPort := status.ToForwardPort[ch]

	if !foundVersionsToRelease && !foundVersionsToForwardPort {
		return []string{"no versions found..."}, nil
	}

	versions := make([]string, 0)
	for _, v := range versionsRelease {
		versions = append(versions, v.Version)
	}
	for _, v := range versionsForwardPort {
		versions = append(versions, v.Version)
	}

	return versions, nil
}

// CheckBranchArgs will check if the branch line exists
func CheckBranchArgs(branch string) bool {
	var found bool
	availableBranches := BranchArgs()

	for _, b := range availableBranches {
		if b == branch {
			found = true
			break
		}
	}

	return found
}

// CheckChartArgs will check if the chart exists in the available charts
func CheckChartArgs(ctx context.Context, conf *config.ChartsRelease, ch string) (bool, error) {
	var found bool
	availableCharts, err := ChartArgs(ctx, conf)
	if err != nil {
		return found, err
	}

	for _, c := range availableCharts {
		if c == ch {
			found = true
			break
		}
	}

	return found, nil
}

// CheckVersionArgs exists to be released or forward ported
func CheckVersionArgs(ctx context.Context, conf *config.ChartsRelease, ch, v string) (bool, error) {
	var found bool
	availableVersions, err := VersionArgs(ctx, conf, ch)
	if err != nil {
		return found, err
	}

	for _, c := range availableVersions {
		if c == v {
			found = true
			break
		}
	}

	return found, nil
}
