package metrics

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/google/go-github/v85/github"
	"github.com/rancher/ecm-distro-tools/cmd/release/config"
)

const (
	reportsFolder = "reports"
)

type Reports struct {
	Harvester          []CVE
	K3s                []CVE
	Longhorn           []CVE
	Observability      []CVE
	ObservabilityAgent []CVE
	Rancher            []CVE
	RKE2               []CVE
}

type ProjectReport struct {
	Name string
	CVEs []CVE
}

type ReportData struct {
	MinSeverity string
	Projects    []ProjectReport
}

// PrintCVEsBySeverity prints a formatted report of all CVEs that meet or exceed the given severity using a template.
func (r *Reports) PrintCVEsBySeverity(minSeverity string) error {
	minScore := severityScore(minSeverity)
	minSeverityDisplay := strings.ToUpper(minSeverity[:1]) + strings.ToLower(minSeverity[1:])

	// Group the projects for easy iteration in the template
	allProjects := []ProjectReport{
		{"Harvester", r.Harvester},
		{"K3s", r.K3s},
		{"Longhorn", r.Longhorn},
		{"Observability", r.Observability},
		{"Observability Agent", r.ObservabilityAgent},
		{"Rancher", r.Rancher},
		{"RKE2", r.RKE2},
	}

	data := ReportData{
		MinSeverity: minSeverityDisplay,
	}

	// Filter CVEs based on severity
	for _, p := range allProjects {
		var matched []CVE
		for _, cve := range p.CVEs {
			if severityScore(cve.Severity) >= minScore {
				matched = append(matched, cve)
			}
		}
		data.Projects = append(data.Projects, ProjectReport{
			Name: p.Name,
			CVEs: matched,
		})
	}

	// Register a custom function to format the severity block
	funcMap := template.FuncMap{
		"formatSeverity": func(s string) string {
			return fmt.Sprintf("%-8s", strings.ToUpper(s))
		},
	}

	tmpl, err := template.New("cve-report").Funcs(funcMap).Parse(reportTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse report template: %w", err)
	}

	// Execute the template directly to standard output
	if err := tmpl.Execute(os.Stdout, data); err != nil {
		return fmt.Errorf("failed to execute report template: %w", err)
	}

	return nil
}

type CVE struct {
	Image           string
	Release         string
	PackageName     string
	PackageVersion  string
	Type            string
	VulnerabilityID string
	Severity        string
	URL             string
	Target          string
	PatchedVersion  string
	Mirrored        bool
	Status          string
	Justification   string
}

// CVEsMetrics fetches and unmarshals CSV reports into the Reports struct.
func CVEsMetrics(ctx context.Context, client *github.Client) (*Reports, error) {
	opts := &github.RepositoryContentGetOptions{
		Ref: "main",
	}

	_, directoryContent, _, err := client.Repositories.GetContents(
		ctx,
		config.RancherGithubOrganization,
		config.ImageScanningRepositoryName,
		reportsFolder,
		opts,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get contents for %s/%s at path %s: %w",
			config.RancherGithubOrganization, config.ImageScanningRepositoryName, reportsFolder, err)
	}

	// Regex to match the project name and ensure we only process -cves.csv files.
	// the observability-agent should be before observability so the match doesn't clip the name.
	cveFileRegex := regexp.MustCompile(`^report-(harvester|k3s|longhorn|observability-agent|observability|rancher|rke2)-.*-cves\.csv$`)

	reports := &Reports{}

	for _, item := range directoryContent {
		if item.GetType() != "file" {
			continue
		}

		matches := cveFileRegex.FindStringSubmatch(item.GetName())
		// Skip files that don't match our CVE reports pattern (like stats.csv)
		if len(matches) < 2 {
			continue
		}

		projectName := matches[1]

		cves, err := downloadAndParseCSV(item.GetDownloadURL())
		if err != nil {
			return nil, fmt.Errorf("failed to process %s: %w", item.GetName(), err)
		}

		// Route the parsed CVEs to the correct project slice
		switch projectName {
		case "harvester":
			reports.Harvester = append(reports.Harvester, cves...)
		case "k3s":
			reports.K3s = append(reports.K3s, cves...)
		case "longhorn":
			reports.Longhorn = append(reports.Longhorn, cves...)
		case "observability-agent":
			reports.ObservabilityAgent = append(reports.ObservabilityAgent, cves...)
		case "observability":
			reports.Observability = append(reports.Observability, cves...)
		case "rancher":
			reports.Rancher = append(reports.Rancher, cves...)
		case "rke2":
			reports.RKE2 = append(reports.RKE2, cves...)
		}
	}

	return reports, nil
}

// downloadAndParseCSV handles fetching the file and unmarshaling the rows into CVE structs.
func downloadAndParseCSV(downloadURL string) ([]CVE, error) {
	resp, err := http.Get(downloadURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download csv: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d when downloading csv", resp.StatusCode)
	}

	reader := csv.NewReader(resp.Body)

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read csv records: %w", err)
	}

	var cves []CVE
	for i, row := range records {
		// Skip the header row
		if i == 0 {
			continue
		}

		// padding the row to avoid "index out of bounds" panics
		for len(row) < 13 {
			row = append(row, "")
		}

		mirrored, _ := strconv.ParseBool(row[10])

		cve := CVE{
			Image:           row[0],
			Release:         row[1],
			PackageName:     row[2],
			PackageVersion:  row[3],
			Type:            row[4],
			VulnerabilityID: row[5],
			Severity:        row[6],
			URL:             row[7],
			Target:          row[8],
			PatchedVersion:  row[9],
			Mirrored:        mirrored,
			Status:          row[11],
			Justification:   row[12],
		}
		cves = append(cves, cve)
	}

	return cves, nil
}

const reportTemplate = `=== Vulnerability Report (Severity: {{ .MinSeverity }} or higher) ===
{{ range .Projects }}
--------------------------------------------------
Project: {{ .Name }}
--------------------------------------------------
{{- if not .CVEs }}
No CVEs found matching or exceeding severity '{{ $.MinSeverity }}'.
{{- else }}
{{- range .CVEs }}
- [{{ formatSeverity .Severity }}] {{ printf "%-15s" .VulnerabilityID }} | Package: {{ .PackageName }} ({{ .PackageVersion }}) | Release: {{ .Release }}
{{- end }}
{{- end }}
{{ end }}
`

// severityScore assigns a numerical value to severities for easy comparison.
func severityScore(severity string) int {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}
