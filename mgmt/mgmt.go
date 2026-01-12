package mgmt

import (
	"bufio"
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/google/go-github/v81/github"
	ecmHTTP "github.com/rancher/ecm-distro-tools/http"
	"golang.org/x/crypto/pbkdf2"
)

// isRancherMember determines if the given user is
// part of one of the Rancher organizations.
func isRancherMember(members []*github.User, login string) bool {
	for _, member := range members {
		if member.GetLogin() == login {
			return true
		}
	}

	return false
}

// allMembers retrieves all members from the Rancher and the
// Harvester organziations.
func allMembers(ctx context.Context, client *github.Client) ([]*github.User, error) {
	var rke2K3sMembers []*github.User
	var harvesterMembers []*github.User

	lmo := github.ListMembersOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		users, resp, err := client.Organizations.ListMembers(ctx, "rancher", &lmo)
		if err != nil {
			return nil, err
		}
		rke2K3sMembers = append(rke2K3sMembers, users...)
		if resp.NextPage == 0 {
			break
		}
		lmo.Page = resp.NextPage
	}

	for {
		users, resp, err := client.Organizations.ListMembers(ctx, "harvester", &lmo)
		if err != nil {
			return nil, err
		}
		harvesterMembers = append(harvesterMembers, users...)
		if resp.NextPage == 0 {
			break
		}
		lmo.Page = resp.NextPage
	}

	return append(rke2K3sMembers, harvesterMembers...), nil
}

// WeeklyReport generates the weekly report for RKE2 or K3s.
func WeeklyReport(ctx context.Context, client *github.Client, repo string) (*bytes.Buffer, error) {
	const templateName = "weekly-report"
	weekAgo := time.Now().AddDate(0, 0, -7)

	orgRepo := strings.Split(repo, "/")
	org, repo := orgRepo[0], orgRepo[1]

	repository, _, err := client.Repositories.Get(ctx, org, repo)
	if err != nil {
		return nil, err
	}
	stars := repository.GetStargazersCount()
	forks := repository.GetForksCount()

	ilro := github.IssueListByRepoOptions{
		State: "all",
	}
	issues, _, err := client.Issues.ListByRepo(ctx, org, repo, &ilro)
	if err != nil {
		return nil, err
	}

	var openIssues, closedIssues []*github.Issue

	for _, issue := range issues {
		if issue.GetClosedAt().Before(weekAgo) && issue.GetCreatedAt().Before(weekAgo) {
			continue
		}

		switch issue.GetState() {
		case "open":
			openIssues = append(openIssues, issue)
		case "closed":
			closedIssues = append(closedIssues, issue)
		}
	}

	prlo := github.PullRequestListOptions{
		State: "all",
	}
	prs, _, err := client.PullRequests.List(ctx, org, repo, &prlo)
	if err != nil {
		return nil, err
	}

	members, err := allMembers(ctx, client)
	if err != nil {
		return nil, err
	}

	var openCommunityPRs,
		closedCommunityPRs,
		openMemberPRs,
		closedMemberPRs []*github.PullRequest

	for _, pr := range prs {
		if pr.GetClosedAt().Before(weekAgo) && pr.GetCreatedAt().Before(weekAgo) {
			continue
		}

		switch pr.GetState() {
		case "open":
			if isRancherMember(members, pr.GetUser().GetLogin()) {
				openMemberPRs = append(openMemberPRs, pr)
			} else {
				openCommunityPRs = append(openCommunityPRs, pr)
			}
		case "closed":
			if isRancherMember(members, pr.GetUser().GetLogin()) {
				closedMemberPRs = append(closedMemberPRs, pr)
				continue
			} else {
				closedCommunityPRs = append(closedCommunityPRs, pr)
			}
		}
	}

	tmpl := template.Must(template.New(templateName).Parse(weeklyReportTemplate))

	now := time.Now()
	year, week := now.ISOWeek()

	buf := bytes.NewBuffer(nil)

	if err := tmpl.Execute(buf, map[string]interface{}{
		"year":               year,
		"week":               week,
		"prsOpenedCount":     len(openCommunityPRs) + len(openMemberPRs),
		"prsClosedCount":     len(closedCommunityPRs) + len(closedMemberPRs),
		"openCommunityPRs":   openCommunityPRs,
		"openMemberPRs":      openMemberPRs,
		"openPRs":            append(openCommunityPRs, openMemberPRs...),
		"closedCommunityPRs": closedCommunityPRs,
		"closedMemberPRs":    closedMemberPRs,
		"closedPRs":          append(closedCommunityPRs, closedMemberPRs...),
		"stars":              stars,
		"forks":              forks,
		"openIssues":         openIssues,
		"closedIssues":       closedIssues,
	}); err != nil {
		return nil, err
	}

	return buf, nil
}

const weeklyReportTemplate = `# Weekly Report
Weekly status report for {{.year}} Week #{{.week}}

## Weekly Stats
| | Opened this week| Closed this week|
|--|---|-----|
|Issues| {{len .openIssues}} | {{len .closedIssues}} |
|PR's| {{.prsOpenedCount}} | {{.prsClosedCount}} |
|  |  |
|--|--|
| Stars | {{.stars}} |
| Forks | {{.forks}} |

## PR's Closed
{{$length := len .closedPRs}}{{if ne $length 0}}
{{- range $pr := .closedPRs}}
#[{{$pr.GetNumber}}]({{$pr.GetHTMLURL}}) {{$pr.GetTitle}}
{{- end}}
{{else}}
None
{{- end}}
## PR's Opened
{{$length := len .openPRs}}{{if ne $length 0}}
{{- range $pr := .openPRs}}
#[{{$pr.GetNumber}}]({{$pr.GetHTMLURL}}) {{$pr.GetTitle}}
{{- end}}
{{else}}
None
{{- end}}

## Issues Opened
{{$length := len .openIssues}}{{if ne $length 0}}
{{- range $issue := .openIssues}}
#[{{$issue.GetNumber}}]({{$issue.GetHTMLURL}}) {{$issue.GetTitle}}
{{- end}}
{{else}}
None
{{- end}}

## Issues Closed
{{$length := len .closedIssues}}{{if ne $length 0}}
{{- range $issue := .closedIssues}}
#[{{$issue.GetNumber}}]({{$issue.GetHTMLURL}}) {{$issue.GetTitle}}
{{- end}}
{{else}}
None
{{- end}}

## Community PRs Closed
{{$length := len .closedCommunityPRs}}{{if ne $length 0}}
{{- range $pr := .closedCommunityPRs}}
#[{{$pr.GetNumber}}]({{$pr.GetHTMLURL}}) {{$pr.GetTitle}}
{{- end}}
{{- else}}
None
{{- end}}

## Community PRs Opened
{{$length := len .openCommunityPRs}}{{if ne $length 0}}
{{- range $pr := .openCommunityPRs}}
#[{{$pr.GetNumber}}]({{$pr.GetHTMLURL}}) {{$pr.GetTitle}}
{{- end}}
{{- else}}
None
{{- end}}
`

type ReportStats struct {
	OpenedIssues       map[time.Time]int
	ClosedIssues       map[time.Time]int
	OpenedMemberPRs    map[time.Time]int
	ClosedMemberPRs    map[time.Time]int
	OpenedCommunityPRs map[time.Time]int
	ClosedCommunityPRs map[time.Time]int
}

// RepoReportStats returns back weekly issues closed and opened and pr closed and opened
func RepoReportStats(ctx context.Context, client *github.Client, repo string, weeks int) (*ReportStats, error) {
	orgRepo := strings.Split(repo, "/")
	org, repo := orgRepo[0], orgRepo[1]

	ilro := github.IssueListByRepoOptions{
		State: "all",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}
	issues, _, err := client.Issues.ListByRepo(ctx, org, repo, &ilro)
	if err != nil {
		return nil, err
	}
	prlo := github.PullRequestListOptions{
		State: "all",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}
	prs, _, err := client.PullRequests.List(ctx, org, repo, &prlo)
	if err != nil {
		return nil, err
	}
	report := ReportStats{
		OpenedIssues:       make(map[time.Time]int),
		ClosedIssues:       make(map[time.Time]int),
		OpenedMemberPRs:    make(map[time.Time]int),
		ClosedMemberPRs:    make(map[time.Time]int),
		OpenedCommunityPRs: make(map[time.Time]int),
		ClosedCommunityPRs: make(map[time.Time]int),
	}
	for i := 1; i <= weeks; i++ {
		week := time.Now().AddDate(0, 0, -7*i)
		var openedIssues,
			closedIssues,
			openedMemberPRs,
			closedMemberPRs,
			openedCommunityPRs,
			closedCommunityPRs int
		for _, issue := range issues {
			if (issue.GetClosedAt().Before(week) &&
				issue.GetCreatedAt().Before(week)) ||
				(issue.GetCreatedAt().After(week.AddDate(0, 0, 7)) &&
					issue.GetClosedAt().After(week.AddDate(0, 0, 7))) {
				continue
			}
			switch issue.GetState() {
			case "open":
				openedIssues++
			case "closed":
				closedIssues++
			}
		}
		report.OpenedIssues[week] = openedIssues
		report.ClosedIssues[week] = closedIssues

		members, err := allMembers(ctx, client)
		if err != nil {
			return nil, err
		}

		for _, pr := range prs {
			if (pr.GetClosedAt().Before(week) &&
				pr.GetCreatedAt().Before(week)) ||
				(pr.GetCreatedAt().After(week.AddDate(0, 0, 7)) &&
					pr.GetClosedAt().After(week.AddDate(0, 0, 7))) {
				continue
			}

			switch pr.GetState() {
			case "open":
				if isRancherMember(members, pr.GetUser().GetLogin()) {
					openedMemberPRs++
				} else {
					openedCommunityPRs++
				}
			case "closed":
				if isRancherMember(members, pr.GetUser().GetLogin()) {
					closedMemberPRs++
					continue
				} else {
					closedCommunityPRs++
				}
			}
			report.OpenedMemberPRs[week] = openedMemberPRs
			report.ClosedMemberPRs[week] = closedMemberPRs
			report.OpenedCommunityPRs[week] = openedCommunityPRs
			report.ClosedCommunityPRs[week] = closedCommunityPRs
		}
	}
	return &report, nil
}

func ParseBootstrapData(ctx context.Context, token, dataEncoded string) (string, error) {
	parts := strings.SplitN(dataEncoded, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid cipher text, not : delimited")
	}

	clearKey := pbkdf2.Key([]byte(token), []byte(parts[0]), 4096, 32, sha1.New)
	key, err := aes.NewCipher(clearKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(key)
	if err != nil {
		return "", err
	}

	data, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", err
	}

	decryptedBootstrap, err := gcm.Open(nil, data[:gcm.NonceSize()], data[gcm.NonceSize():], nil)
	if err != nil {
		return "", err
	}

	return string(decryptedBootstrap), nil
}

const (
	imageSourcesURL = "https://raw.githubusercontent.com/rancher/rke2/master/developer-docs/image_sources.md"
	chartIndecURL   = "https://raw.githubusercontent.com/rancher/rke2-charts/main/index.yaml"
)

func imageSourcesProc(ctx context.Context, client *http.Client, columnIndex int) ([]string, error) {
	if client == nil {
		httpClient := ecmHTTP.NewClient(time.Second * 30)
		client = &httpClient
	}

	req, err := http.NewRequest(http.MethodGet, imageSourcesURL, nil)
	if err != nil {
		return nil, err
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var images []string

	scanner := bufio.NewScanner(res.Body)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "rancher/hardened-build-base") || strings.Contains(scanner.Text(), "rancher/rke2") {
			line := strings.Split(scanner.Text(), "|")
			if len(line) >= columnIndex {
				images = append(images, line[columnIndex])
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return images, nil
}

// RKE2ImageRepos
func RKE2ImageRepos(ctx context.Context, client *http.Client) ([]string, error) {
	return imageSourcesProc(ctx, client, 3)
}

// RKE2Images
func RKE2Images(ctx context.Context, client *http.Client) ([]string, error) {
	return imageSourcesProc(ctx, client, 2)
}

// RKE2ChartsIndex
func RKE2ChartsIndex(ctx context.Context, client *http.Client) (string, error) {
	if client == nil {
		httpClient := ecmHTTP.NewClient(time.Second * 30)
		client = &httpClient
	}

	req, err := http.NewRequest(http.MethodGet, chartIndecURL, nil)
	if err != nil {
		return "", err
	}

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	b := bytes.NewBuffer(nil)
	io.Copy(b, res.Body)

	return b.String(), nil
}
