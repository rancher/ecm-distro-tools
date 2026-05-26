package metrics

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/google/go-github/v85/github"
	"github.com/rancher/ecm-distro-tools/cmd/release/config"
)

const reportsFolder = "reports"

// Slack severity emojis.
const (
	emojiCritical = "🔴"
	emojiHigh     = "🟠"
	emojiMedium   = "🟡"
	emojiLow      = "🔵"
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

type SeverityCounts struct {
	Critical int
	High     int
	Medium   int
	Low      int
	Other    int
}

func (s SeverityCounts) Total() int {
	return s.Critical + s.High + s.Medium + s.Low + s.Other
}

type ProjectReport struct {
	Name   string
	CVEs   []CVE
	Counts SeverityCounts
}

type ReportData struct {
	MinSeverity string
	Projects    []ProjectReport
	Totals      SeverityCounts
}

// PrintCVEsBySeverity filters and renders a CVE report as a Slack payload.
func (r *Reports) CVEsBySeverity(minSeverity string) error {
	data := r.buildReportData(minSeverity)
	return renderSlack(data)
}

// buildReportData filters, sorts, and counts CVEs.
func (r *Reports) buildReportData(minSeverity string) ReportData {
	minScore := severityScore(minSeverity)
	minSeverityDisplay := strings.ToUpper(minSeverity[:1]) + strings.ToLower(minSeverity[1:])

	allProjects := []ProjectReport{
		{Name: "Harvester", CVEs: r.Harvester},
		{Name: "K3s", CVEs: r.K3s},
		{Name: "Longhorn", CVEs: r.Longhorn},
		{Name: "Observability", CVEs: r.Observability},
		{Name: "Observability Agent", CVEs: r.ObservabilityAgent},
		{Name: "Rancher", CVEs: r.Rancher},
		{Name: "RKE2", CVEs: r.RKE2},
	}

	data := ReportData{MinSeverity: minSeverityDisplay}

	for _, p := range allProjects {
		var matched []CVE
		for _, cve := range p.CVEs {
			// skip any CVE that does not affect the current project
			if cve.Status == "not_affected" {
				continue
			}
			if severityScore(cve.Severity) >= minScore {
				matched = append(matched, cve)
			}
		}

		sort.Slice(matched, func(i, j int) bool {
			si, sj := severityScore(matched[i].Severity), severityScore(matched[j].Severity)
			if si != sj {
				return si > sj
			}
			return matched[i].VulnerabilityID < matched[j].VulnerabilityID
		})

		counts := countSeverities(matched)

		data.Projects = append(data.Projects, ProjectReport{
			Name:   p.Name,
			CVEs:   matched,
			Counts: counts,
		})

		data.Totals.Critical += counts.Critical
		data.Totals.High += counts.High
		data.Totals.Medium += counts.Medium
		data.Totals.Low += counts.Low
		data.Totals.Other += counts.Other
	}

	return data
}

// ── Slack renderer ───────────────────────────────────────────────────────────

func renderSlack(data ReportData) error {
	payload := buildSlackPayload(data)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

type slackPayload struct {
	Blocks []slackBlock `json:"blocks"`
}

type slackBlock struct {
	Type string     `json:"type"`
	Text *slackText `json:"text,omitempty"`
}

type slackText struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Emoji bool   `json:"emoji,omitempty"`
}

func buildSlackPayload(data ReportData) slackPayload {
	var blocks []slackBlock

	blocks = append(blocks,
		headerBlock(fmt.Sprintf("🔍 Vulnerability Report · Min Severity: %s", data.MinSeverity)),
		sectionBlock(fmt.Sprintf("*SUMMARY*\n%s", buildSummaryTable(data))),
		dividerBlock(),
		sectionBlock("*DETAILS*"),
	)

	for _, p := range data.Projects {
		if p.Counts.Total() == 0 {
			blocks = append(blocks, sectionBlock(
				fmt.Sprintf("✅  *%s* — No findings at or above this severity", p.Name),
			))
			continue
		}

		plural := "s"
		if p.Counts.Total() == 1 {
			plural = ""
		}
		blocks = append(blocks,
			sectionBlock(fmt.Sprintf("*%s* — %d CVE%s", p.Name, p.Counts.Total(), plural)),
			sectionBlock(buildCVELines(p.CVEs)),
			dividerBlock(),
		)
	}

	return slackPayload{Blocks: blocks}
}

func buildSummaryTable(data ReportData) string {
	const header = "Project                  CRIT   HIGH    MED    LOW   TOTAL"
	sep := strings.Repeat("─", len(header))

	var sb strings.Builder
	sb.WriteString("```\n")
	sb.WriteString(header + "\n")
	sb.WriteString(sep + "\n")

	for _, p := range data.Projects {
		fmt.Fprintf(&sb, "%-24s %5d  %5d  %5d  %5d  %6d\n",
			p.Name,
			p.Counts.Critical, p.Counts.High, p.Counts.Medium, p.Counts.Low,
			p.Counts.Total(),
		)
	}

	sb.WriteString(sep + "\n")
	fmt.Fprintf(&sb, "%-24s %5d  %5d  %5d  %5d  %6d\n",
		"Total",
		data.Totals.Critical, data.Totals.High, data.Totals.Medium, data.Totals.Low,
		data.Totals.Total(),
	)
	sb.WriteString("```")

	return sb.String()
}

func buildCVELines(cves []CVE) string {
	var lines []string
	for _, cve := range cves {
		line := fmt.Sprintf("%s  `%s`  %s %s  _%s_",
			severityEmoji(cve.Severity),
			cve.VulnerabilityID,
			cve.PackageName,
			cve.PackageVersion,
			cve.Release,
		)
		if cve.Status != "" {
			line += "  ·  " + cve.Status
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func headerBlock(text string) slackBlock {
	return slackBlock{Type: "header", Text: &slackText{Type: "plain_text", Text: text, Emoji: true}}
}

func sectionBlock(text string) slackBlock {
	return slackBlock{Type: "section", Text: &slackText{Type: "mrkdwn", Text: text}}
}

func dividerBlock() slackBlock {
	return slackBlock{Type: "divider"}
}

// ── Shared helpers ───────────────────────────────────────────────────────────

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

func countSeverities(cves []CVE) SeverityCounts {
	var c SeverityCounts
	for _, cve := range cves {
		switch strings.ToLower(strings.TrimSpace(cve.Severity)) {
		case "critical":
			c.Critical++
		case "high":
			c.High++
		case "medium":
			c.Medium++
		case "low":
			c.Low++
		default:
			c.Other++
		}
	}
	return c
}

func severityEmoji(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical":
		return emojiCritical
	case "high":
		return emojiHigh
	case "medium":
		return emojiMedium
	case "low":
		return emojiLow
	default:
		return "⚪"
	}
}

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

// ── Data fetching ────────────────────────────────────────────────────────────

func CVEsMetrics(ctx context.Context, client *github.Client) (*Reports, error) {
	opts := &github.RepositoryContentGetOptions{Ref: "main"}

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

	cveFileRegex := regexp.MustCompile(`^report-(harvester|k3s|longhorn|observability-agent|observability|rancher|rke2)-.*-cves\.csv$`)
	reports := &Reports{}

	for _, item := range directoryContent {
		if item.GetType() != "file" {
			continue
		}

		matches := cveFileRegex.FindStringSubmatch(item.GetName())
		if len(matches) < 2 {
			continue
		}

		cves, err := downloadAndParseCSV(item.GetDownloadURL())
		if err != nil {
			return nil, fmt.Errorf("failed to process %s: %w", item.GetName(), err)
		}

		switch matches[1] {
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

func downloadAndParseCSV(downloadURL string) ([]CVE, error) {
	resp, err := http.Get(downloadURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download csv: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d when downloading csv", resp.StatusCode)
	}

	records, err := csv.NewReader(resp.Body).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read csv records: %w", err)
	}

	var cves []CVE
	for i, row := range records {
		if i == 0 {
			continue
		}
		for len(row) < 13 {
			row = append(row, "")
		}
		mirrored, _ := strconv.ParseBool(row[10])
		cves = append(cves, CVE{
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
		})
	}

	return cves, nil
}
