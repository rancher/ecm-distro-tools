package metrics

import (
	"strings"

	"github.com/google/go-github/v77/github"
)

type (
	yearMonthMap map[int]*[12]int
	yearMap      map[int]int
)

type Metrics struct {
	Rancher      ReleaseMetrics   `json:"rancher"`
	RancherPrime ReleaseMetrics   `json:"rancher_prime"`
	Workflows    WorkflowsMetrics `json:"actions"`
}

type ReleaseMetrics struct {
	// Number of GA releases per year and month
	// Value: Number of releases per month (Jan: 0, Feb: 1, ..., Dec: 11)
	GAReleasesPerMonth yearMonthMap `json:"ga_releases_per_month"`
	// Number of Pre-releases per year and month (any release with a suffix that starts with a dash '-*')
	// Value: Number of releases per month (Jan: 0, Feb: 1, ..., Dec: 11)
	PreReleasesPerMonth yearMonthMap `json:"pre_releases_per_month"`
	// Number of GA releases per year
	// Value: Number of releases per year
	GAReleasesPerYear yearMap `json:"ga_releases_per_year"`
	// Number of Pre-releases per year
	// Value: Number of releases per year
	PreReleasesPerYear yearMap `json:"pre_releases_per_year"`
}

type WorkflowsMetrics struct {
	// Number of successful actions per year and month
	// Key: Year
	// Value: Number of successful actions per month (Jan: 0, Feb: 1, ..., Dec: 11)
	SuccessfulWorkflowsPerMonth yearMonthMap `json:"successful_actions_per_month"`
	// Number of failed actions per year and month
	// Key: Year
	// Value: Number of failed actions per month (Jan: 0, Feb: 1, ..., Dec: 11)
	FailedWorkflowsPerMonth yearMonthMap `json:"failed_actions_per_month"`
}

func ExtractMetrics(rancherReleases []github.RepositoryRelease, primeReleases []github.RepositoryRelease, workflows []github.WorkflowRun) (Metrics, error) {
	var metrics Metrics

	rancher, err := extractReleaseMetrics(rancherReleases)
	if err != nil {
		return metrics, err
	}

	rancherPrimeReleases, err := extractReleaseMetrics(primeReleases)

	workflowsMetrics, err := extractWorkflowsMetrics(workflows)
	if err != nil {
		return metrics, err
	}

	metrics.Rancher = rancher
	metrics.RancherPrime = rancherPrimeReleases
	metrics.Workflows = workflowsMetrics

	return metrics, nil
}

func extractReleaseMetrics(releases []github.RepositoryRelease) (ReleaseMetrics, error) {
	var metrics ReleaseMetrics

	metrics.GAReleasesPerMonth = make(yearMonthMap)
	metrics.PreReleasesPerMonth = make(yearMonthMap)
	metrics.GAReleasesPerYear = make(yearMap)
	metrics.PreReleasesPerYear = make(yearMap)

	for _, release := range releases {
		releaseDate := release.GetCreatedAt().Time

		monthIndex := int(releaseDate.Month() - 1)
		year := releaseDate.Year()

		if _, ok := metrics.GAReleasesPerMonth[year]; !ok {
			metrics.GAReleasesPerMonth[year] = &[12]int{}
		}
		if _, ok := metrics.PreReleasesPerMonth[year]; !ok {
			metrics.PreReleasesPerMonth[year] = &[12]int{}
		}

		// is pre-release
		if strings.Contains(release.GetTagName(), "-") {
			metrics.PreReleasesPerMonth[year][monthIndex]++
		} else {
			metrics.GAReleasesPerMonth[year][monthIndex]++
		}
	}

	for year, releases := range metrics.GAReleasesPerMonth {
		for _, count := range releases {
			metrics.GAReleasesPerYear[year] += count
		}
	}

	for year, releases := range metrics.PreReleasesPerMonth {
		for _, count := range releases {
			metrics.PreReleasesPerYear[year] += count
		}
	}

	return metrics, nil
}

func extractWorkflowsMetrics(workflows []github.WorkflowRun) (WorkflowsMetrics, error) {
	var metrics WorkflowsMetrics

	metrics.SuccessfulWorkflowsPerMonth = make(yearMonthMap)
	metrics.FailedWorkflowsPerMonth = make(yearMonthMap)

	for _, workflow := range workflows {
		workflowDate := workflow.GetCreatedAt().Time

		monthIndex := int(workflowDate.Month() - 1)
		year := workflowDate.Year()

		if _, ok := metrics.SuccessfulWorkflowsPerMonth[year]; !ok {
			metrics.SuccessfulWorkflowsPerMonth[year] = &[12]int{}
		}
		if _, ok := metrics.FailedWorkflowsPerMonth[year]; !ok {
			metrics.FailedWorkflowsPerMonth[year] = &[12]int{}
		}

		if workflow.GetConclusion() == "success" {
			metrics.SuccessfulWorkflowsPerMonth[year][monthIndex]++
		} else {
			metrics.FailedWorkflowsPerMonth[year][monthIndex]++
		}
	}
	return metrics, nil
}
