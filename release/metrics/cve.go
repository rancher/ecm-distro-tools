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

// ReleaseReport holds all filtered CVEs for a single project+release pair.
type ReleaseReport struct {
	ProjectName string
	Release     string
	CVEs        []CVE
	Counts      SeverityCounts
}

type ReportData struct {
	MinSeverity string
	Releases    []ReleaseReport
	Totals      SeverityCounts
}

// CVEsBySeverity filters and renders a CVE report, sending one Slack message per release.
func (r *Reports) CVEsBySeverity(minSeverity, webhookURL string) error {
	data := r.buildReportData(minSeverity)

	for i, release := range data.Releases {
		if err := notifySlackRelease(release, data.MinSeverity, webhookURL); err != nil {
			return fmt.Errorf("failed to send message for %s · %s: %w", release.ProjectName, release.Release, err)
		}

		// Slack has a rate limit for Incoming Webhooks, of 1 req / sec.
		if i < len(data.Releases)-1 {
			time.Sleep(2 * time.Second)
		}
	}

	fmt.Println("Successfully sent all release reports to Slack!")
	return nil
}

// buildReportData filters, sorts, and groups CVEs by project and release.
func (r *Reports) buildReportData(minSeverity string) ReportData {
	minScore := severityScore(minSeverity)
	minSeverityDisplay := strings.ToUpper(minSeverity[:1]) + strings.ToLower(minSeverity[1:])

	allProjects := []struct {
		name string
		cves []CVE
	}{
		{"Harvester", r.Harvester},
		{"K3s", r.K3s},
		{"Longhorn", r.Longhorn},
		{"Observability", r.Observability},
		{"Observability Agent", r.ObservabilityAgent},
		{"Rancher", r.Rancher},
		{"RKE2", r.RKE2},
	}

	data := ReportData{MinSeverity: minSeverityDisplay}

	for _, p := range allProjects {
		releaseMap := make(map[string][]CVE)
		var releaseOrder []string

		for _, cve := range p.cves {
			if cve.Status != "affected" && cve.Status != "under_investigation" {
				continue
			}
			if severityScore(cve.Severity) < minScore {
				continue
			}
			if _, seen := releaseMap[cve.Release]; !seen {
				releaseOrder = append(releaseOrder, cve.Release)
			}
			releaseMap[cve.Release] = append(releaseMap[cve.Release], cve)
		}

		for _, release := range releaseOrder {
			cves := releaseMap[release]

			sort.Slice(cves, func(i, j int) bool {
				si, sj := severityScore(cves[i].Severity), severityScore(cves[j].Severity)
				if si != sj {
					return si > sj
				}
				return cves[i].VulnerabilityID < cves[j].VulnerabilityID
			})

			counts := countSeverities(cves)
			data.Releases = append(data.Releases, ReleaseReport{
				ProjectName: p.name,
				Release:     release,
				CVEs:        cves,
				Counts:      counts,
			})

			data.Totals.Critical += counts.Critical
			data.Totals.High += counts.High
			data.Totals.Medium += counts.Medium
			data.Totals.Low += counts.Low
			data.Totals.Other += counts.Other
		}
	}

	return data
}

func notifySlackRelease(release ReleaseReport, minSeverity, webhookURL string) error {
	if len(release.CVEs) == 0 {
		fmt.Printf("Skipping notification for '%s · %s', no CVE of severity '%s' or higher found\n",
			release.ProjectName, release.Release, minSeverity)
		return nil
	}

	payload := buildReleaseSlackPayload(release, minSeverity)

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

// buildReleaseSlackPayload builds a Slack message for a single project release.
//
// Block budget: Slack enforces a hard limit of 50 blocks per payload. With releases
// that have hundreds of images and thousands of CVEs, emitting one block per image
// header and per status label would easily blow past that limit.
//
// To stay well under 50 blocks we stream all content (image headers, status labels,
// and CVE lines) into a single text builder and flush it into section blocks only
// when the 2900-char mrkdwn limit is approached. The result is:
//
//	3 fixed blocks (header + summary + divider)
//	+ ceil(total_content_chars / 2900) content blocks
//
// For a release with 1000 CVEs (~60 chars/line ≈ 60 KB), that yields ~24 blocks total.
func buildReleaseSlackPayload(release ReleaseReport, minSeverity string) slackPayload {
	var blocks []slackBlock

	blocks = append(blocks,
		headerBlock(fmt.Sprintf("🔍 %s · %s  |  Min: %s", release.ProjectName, release.Release, minSeverity)),
	)

	if release.Counts.Total() == 0 {
		blocks = append(blocks, sectionBlock("✅  No findings at or above this severity"))
		return slackPayload{Blocks: blocks}
	}

	plural := "s"
	if release.Counts.Total() == 1 {
		plural = ""
	}

	summaryLine := fmt.Sprintf("*%d CVE%s*  |  🔴 %d CRIT  |  🟠 %d HIGH  |  🟡 %d MED  |  🔵 %d LOW",
		release.Counts.Total(), plural,
		release.Counts.Critical, release.Counts.High, release.Counts.Medium, release.Counts.Low)

	blocks = append(blocks,
		sectionBlock(summaryLine),
		dividerBlock(),
	)

	// Group CVEs by image, sorted alphabetically for deterministic output.
	imageMap := make(map[string][]CVE)
	var imageOrder []string
	for _, cve := range release.CVEs {
		if _, seen := imageMap[cve.Image]; !seen {
			imageOrder = append(imageOrder, cve.Image)
		}
		imageMap[cve.Image] = append(imageMap[cve.Image], cve)
	}
	sort.Strings(imageOrder)

	statusLabel := map[string]string{
		"affected":            "⚠️ *Affected*",
		"under_investigation": "🔎 *Under Investigation*",
	}

	// stream accumulates all CVE content into a single rolling text buffer.
	// writeLine flushes it into a new section block whenever the mrkdwn limit
	// is approached, keeping structural labels and CVE lines in the same flow.
	var stream strings.Builder

	flush := func() {
		if stream.Len() > 0 {
			blocks = append(blocks, sectionBlock(stream.String()))
			stream.Reset()
		}
	}

	writeLine := func(line string) {
		if stream.Len()+len(line) > 2900 {
			flush()
		}
		stream.WriteString(line)
	}

	for i, image := range imageOrder {
		writeLine(fmt.Sprintf("*📦 %s*\n", image))

		// Always render affected before under_investigation.
		for _, status := range []string{"affected", "under_investigation"} {
			var group []CVE
			for _, cve := range imageMap[image] {
				if cve.Status == status {
					group = append(group, cve)
				}
			}
			if len(group) == 0 {
				continue
			}

			writeLine(statusLabel[status] + "\n")
			for _, cve := range group {
				writeLine(fmt.Sprintf("%s `%s` %s %s _%s_\n",
					severityEmoji(cve.Severity),
					cve.VulnerabilityID,
					cve.PackageName,
					cve.PackageVersion,
					cve.Type,
				))
			}
		}

		// Blank line between images for visual separation, except after the last one.
		if i < len(imageOrder)-1 {
			writeLine("\n")
		}
	}

	flush()

	return slackPayload{Blocks: blocks}
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
