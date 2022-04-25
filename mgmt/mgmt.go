package mgmt

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/google/go-github/v39/github"
)

// WeeklyReport
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
		Since: weekAgo,
	}
	issues, _, err := client.Issues.ListByRepo(ctx, org, repo, &ilro)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var openIssues []*github.Issue
	var closedIssues []*github.Issue

	for _, issue := range issues {
		switch issue.GetState() {
		case "open":
			openIssues = append(openIssues, issue)
		case "closed":
			closedIssues = append(closedIssues, issue)
		}
	}

	prlo := github.PullRequestListOptions{}
	prs, _, err := client.PullRequests.List(ctx, org, repo, &prlo)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var openPRs []*github.PullRequest
	var closedPRs []*github.PullRequest

	for _, pr := range prs {
		switch pr.GetState() {
		case "open":
			openPRs = append(openPRs, pr)
			if pr.CreatedAt.After(weekAgo) {
				fmt.Printf("%#v", pr)
			}
		case "closed":
			closedPRs = append(closedPRs, pr)
		}
	}

	tmpl := template.Must(template.New(templateName).Parse(weeklyReportTemplate))

	buf := bytes.NewBuffer(nil)

	if err := tmpl.Execute(buf, map[string]interface{}{
		"issuesOpened": len(openIssues),
		"issuesClosed": len(closedIssues),
		"prsOpened":    len(openPRs),
		"prsClosed":    len(closedPRs),
		"stars":        stars,
		"forks":        forks,
	}); err != nil {
		return nil, err
	}

	return buf, nil
}

const weeklyReportTemplate = `# Weekly Report
Weekly status report for %s Week #%s
## Here's what the team has focused on this week:
* 
## Weekly Stats
| | Opened this week| Closed this week|
|--|---|-----|
|Issues| {{.issuesOpened}} | {{.issuesClosed}} |
|PR's| {{.prsOpened}} | {{.prsClosed}} |
|  |  |
|--|--|
| Stars | {{.stars}} |
| Forks | {{.forks}} |
## PR's Closed
%s
## PR's Opened
%s
## Issues Opened
%s
## Issues Closed
%s
## Community PRs Closed
%s
## Community PRs Opened
%s
`
