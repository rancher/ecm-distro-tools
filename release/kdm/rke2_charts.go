package kdm

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"gopkg.in/yaml.v2"
)

type (
	ChartsFile struct {
		Charts []Chart `yaml:"charts"`
	}

	Chart struct {
		Repo     string `yaml:"repo,omitempty"`
		Version  string `yaml:"version,omitempty"`
		Filename string `yaml:"filename,omitempty"`
	}
)

const (
	chartsURLFormat = "https://raw.githubusercontent.com/rancher/rke2/%s/charts/chart_versions.yaml"
)

func getChartsFromVersion(version string) (map[string]Chart, error) {
	chartsURL := fmt.Sprintf(chartsURLFormat, version)

	resp, err := http.Get(chartsURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorBodyBytes []byte
		errorBodyBytes, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("received an error from GitHub API: %s", string(errorBodyBytes))
	}

	chartsFileContent, err := io.ReadAll(resp.Body) // io.ReadAll is preferred over ioutil.ReadAll since Go 1.16
	if err != nil {
		return nil, err
	}

	var chartFile ChartsFile
	if err = yaml.Unmarshal(chartsFileContent, &chartFile); err != nil {
		return nil, err
	}

	charts := map[string]Chart{}

	for _, chart := range chartFile.Charts {
		chartName := strings.TrimSuffix(chart.Filename, ".yaml")
		chartName = strings.TrimPrefix(chartName, "/charts/")
		charts[chartName] = Chart{
			Repo:    "rancher-rke2-charts",
			Version: chart.Version,
		}
	}

	return charts, nil
}

func GetUpdatedCharts(milestone, prevMilestone string) (string, error) {
	currentCharts, err := getChartsFromVersion(milestone)
	if err != nil {
		return "", err
	}

	previousCharts, err := getChartsFromVersion(prevMilestone)
	if err != nil {
		return "", err
	}

	updatedCharts := map[string]Chart{}

	for name, details := range currentCharts {
		prevChart, ok := previousCharts[name]
		// if a new chart was added in the current release,
		// it won't be found in the previous release's charts.
		if !ok {
			updatedCharts[name] = details
			continue
		}
		if prevChart.Version != details.Version {
			updatedCharts[name] = details
			continue
		}
	}

	b, err := yaml.Marshal(updatedCharts)
	if err != nil {
		return "", err
	}

	return string(b), nil
}
