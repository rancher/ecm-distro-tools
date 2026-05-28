package metrics

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v85/github"
	"github.com/rancher/ecm-distro-tools/cmd/release/config"
)

const reportsFolder = "reports"

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

// CVEsBySeverity filters and renders a CVE report, sending one Slack message per project.
func (r *Reports) CVEsBySeverity(minSeverity, webhookURL string) error {
	data := r.buildReportData(minSeverity)

	for i, project := range data.Projects {
		if err := notifySlackProject(project, data.MinSeverity, webhookURL); err != nil {
			return fmt.Errorf("failed to send message for project %s: %w", project.Name, err)
		}

		// Slack has a rate limit for Incoming Webhooks, of 1 req / sec.
		if i < len(data.Projects)-1 {
			time.Sleep(2 * time.Second)
		}
	}

	fmt.Println("Successfully sent all project reports to Slack!")
	return nil
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
			if cve.Status == "not_affected" || cve.Status == "under_investigation" {
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

func notifySlackProject(project ProjectReport, minSeverity, webhookURL string) error {
	if len(project.CVEs) == 0 {
		fmt.Printf("Skipping notification for '%s', no CVE of severity '%s' or higher found\n", project.Name, minSeverity)
		return nil
	}
	payload := buildProjectSlackPayload(project, minSeverity)

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal slack payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to slack: %w", err)
	}
	defer resp.Body.Close()

	respB, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response's body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("slack API rejected the request with status %d: %s", resp.StatusCode, string(respB))
	}

	return nil
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

func buildProjectSlackPayload(project ProjectReport, minSeverity string) slackPayload {
	var blocks []slackBlock

	blocks = append(blocks,
		headerBlock(fmt.Sprintf("🔍 %s Vulnerabilities · Min: %s", project.Name, minSeverity)),
	)

	if project.Counts.Total() == 0 {
		blocks = append(blocks, sectionBlock("✅  No findings at or above this severity"))
		return slackPayload{Blocks: blocks}
	}

	plural := "s"
	if project.Counts.Total() == 1 {
		plural = ""
	}

	summaryLine := fmt.Sprintf("*%d CVE%s* |  🔴 %d CRIT  |  🟠 %d HIGH  |  🟡 %d MED  |  🔵 %d LOW",
		project.Counts.Total(), plural,
		project.Counts.Critical, project.Counts.High, project.Counts.Medium, project.Counts.Low)

	blocks = append(blocks,
		sectionBlock(summaryLine),
		dividerBlock(),
	)

	// Append chunked CVE blocks to avoid Slack's 3000 character limit
	blocks = append(blocks, buildCVEBlocks(project.CVEs)...)

	return slackPayload{Blocks: blocks}
}

func buildCVEBlocks(cves []CVE) []slackBlock {
	var blocks []slackBlock
	var currentChunk strings.Builder

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
		line += "\n"

		// Slack mrkdwn limit is 3000 chars. We chunk at 2900 to be perfectly safe.
		if currentChunk.Len()+len(line) > 2900 {
			blocks = append(blocks, sectionBlock(currentChunk.String()))
			currentChunk.Reset()
		}
		currentChunk.WriteString(line)
	}

	// Flush the remaining text as the final block
	if currentChunk.Len() > 0 {
		blocks = append(blocks, sectionBlock(currentChunk.String()))
	}

	return blocks
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

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	reports := &Reports{}
	for _, item := range directoryContent {
		if item.GetType() != "file" {
			continue
		}

		matches := cveFileRegex.FindStringSubmatch(item.GetName())
		if len(matches) < 2 {
			continue
		}

		cves, err := downloadAndParseCSV(httpClient, item.GetDownloadURL())
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

func downloadAndParseCSV(client *http.Client, downloadURL string) ([]CVE, error) {

	req, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
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
