package kdm

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"sigs.k8s.io/yaml"
)

type (
	ChartsFile struct {
		Charts []Chart `yaml:"charts"`
	}

	Chart struct {
		Repo     string `json:"repo,omitempty" yaml:"repo,omitempty"`
		Version  string `json:"version,omitempty" yaml:"version,omitempty"`
		Filename string `json:"filename,omitempty" yaml:"filename,omitempty"`
	}
)

func chartsFromVersion(version string) (map[string]Chart, error) {
	chartsURL := "https://raw.githubusercontent.com/rancher/rke2/" + version + "/charts/chart_versions.yaml"

	resp, err := http.Get(chartsURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorBody []byte
		errorBody, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, errors.New("received an error from GitHub API: " + string(errorBody))
	}

	chartsFileContent, err := io.ReadAll(resp.Body) // io.ReadAll is preferred over ioutil.ReadAll since Go 1.16
	if err != nil {
		return nil, err
	}

	var chartFile ChartsFile
	if err := yaml.Unmarshal(chartsFileContent, &chartFile); err != nil {
		return nil, err
	}

	charts := make(map[string]Chart)

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

func UpdatedCharts(milestone, prevMilestone string) (string, error) {
	currentCharts, err := chartsFromVersion(milestone)
	if err != nil {
		return "", err
	}

	previousCharts, err := chartsFromVersion(prevMilestone)
	if err != nil {
		return "", err
	}

	updatedCharts := make(map[string]Chart)

	for name, details := range currentCharts {
		// if a new chart was added in the current release,
		// it won't be found in the previous release's charts.
		if prevChart, ok := previousCharts[name]; !ok {
			updatedCharts[name] = details
		} else if prevChart.Version != details.Version {
			updatedCharts[name] = details
		}
	}

	b, err := yaml.Marshal(updatedCharts)
	if err != nil {
		return "", err
	}

	return string(b), nil
}
