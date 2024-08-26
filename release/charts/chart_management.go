package charts

import (
	"context"
	"encoding/json"
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

// ChartArgs will return the list of available charts in the current branch
func ChartArgs(ctx context.Context, c *config.ChartsRelease) ([]string, error) {
	var dirs []string

	assets := filepath.Join(c.Workspace, "assets")

	files, err := os.ReadDir(assets)
	if err != nil {
		return nil, err
	}

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

// MountReleaseBranch will mount the release branch name from the line provided
func MountReleaseBranch(line string) string {
	return "release-v" + line
}

// IsBranchAvailable will check if the branch line exists
func IsBranchAvailable(branch string, availableBranches []string) bool {
	for _, b := range availableBranches {
		if b == branch {
			return true
		}
	}

	return false
}

// IsChartAvailable will check if the chart exists in the available charts
func IsChartAvailable(ctx context.Context, conf *config.ChartsRelease, ch string) (bool, error) {
	availableCharts, err := ChartArgs(ctx, conf)
	if err != nil {
		return false, err
	}

	for _, c := range availableCharts {
		if c == ch {
			return true, nil
		}
	}

	return false, nil
}

// IsVersionAvailable exists to be released or forward ported
func IsVersionAvailable(ctx context.Context, conf *config.ChartsRelease, ch, version string) (bool, error) {
	availableVersions, err := VersionArgs(ctx, conf, ch)
	if err != nil {
		return false, err
	}

	for _, c := range availableVersions {
		if c == version {
			return true, nil
		}
	}

	return false, nil
}

// loadState will load the lifecycle-status state from an existing state.json file at charts repo
func loadState(filePath string) (*status, error) {
	s := &status{}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	if err := json.NewDecoder(file).Decode(&s); err != nil {
		return nil, err
	}

	return s, nil
}
