package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/drone/drone-go/drone"
	"golang.org/x/oauth2"
)

type droneBuild struct {
	id      string
	buildNo int
	title   string
	desc    string
	org     string
	repo    string
	server  string
	token   string
	running bool
	passing bool
	failing bool
	elapsed string
}

func formatDurationSince(timestamp int64) string {
	t := time.Unix(timestamp, 0)
	duration := time.Since(t)

	seconds := int64(duration.Seconds())
	minutes := seconds / 60
	hours := minutes / 60
	days := hours / 24
	weeks := days / 7
	months := days / 30 // Approximation
	years := days / 365 // Approximation

	switch {
	case seconds < 60:
		return fmt.Sprintf("%d seconds", seconds)
	case minutes < 60:
		return fmt.Sprintf("%d minutes", minutes)
	case hours < 24:
		return fmt.Sprintf("%d hours", hours)
	case days < 7:
		return fmt.Sprintf("%d days", days)
	case weeks < 4:
		return fmt.Sprintf("%d weeks", weeks)
	case months < 12:
		return fmt.Sprintf("%d months", months)
	default:
		return fmt.Sprintf("%d years", years)
	}
}

func newDroneBuild(server, token, org, repo string, b *drone.Build) droneBuild {
	return droneBuild{
		id:      strconv.Itoa(int(b.ID)),
		buildNo: int(b.Number),
		title:   b.Title,
		desc:    b.Message,
		org:     org,
		repo:    repo,
		server:  server,
		token:   token,
		passing: b.Status == drone.StatusPassing,
		failing: b.Status == drone.StatusFailing,
		running: b.Status == drone.StatusRunning,
		elapsed: formatDurationSince(b.Started),
	}
}

var droneBuildRunningStyle = lipgloss.NewStyle().Background(lipgloss.Color("220")).Foreground(lipgloss.AdaptiveColor{Light: "236", Dark: "248"})
var droneBuildFailingStyle = lipgloss.NewStyle().Background(lipgloss.Color("160")).Foreground(lipgloss.AdaptiveColor{Light: "236", Dark: "248"})
var droneBuildPassingStyle = lipgloss.NewStyle().Background(lipgloss.Color("40")).Foreground(lipgloss.AdaptiveColor{Light: "236", Dark: "248"})
var droneBuildNotStartedStyle = lipgloss.NewStyle().Background(lipgloss.Color("255")).Foreground(lipgloss.AdaptiveColor{Light: "236", Dark: "248"})

func (i droneBuild) BuildNo() int {
	return i.buildNo
}
func (i droneBuild) Title() string {
	return "[Drone] " + i.title
}
func (i droneBuild) Description() string {
	var elapsed string
	if i.elapsed != "" {
		elapsed = i.elapsed + " elapsed"
	}
	if i.running {
		return droneBuildRunningStyle.Render("running") + "    " + elapsed
	}
	if i.passing {
		return droneBuildPassingStyle.Render("passing") + "    " + elapsed
	}
	if i.failing {
		return droneBuildFailingStyle.Render("failing") + "    " + elapsed
	}
	return droneBuildNotStartedStyle.Render("not started")
}
func (i droneBuild) FilterValue() string {
	return i.id + " " + i.title
}
func (i droneBuild) ID() string {
	return i.id
}
func (i droneBuild) Org() string {
	return i.org
}
func (i droneBuild) Repo() string {
	return i.repo
}
func (i droneBuild) Type() string {
	switch i.server {
	case "https://drone-pr.rancher.io":
		return "drone_pr_build"
	case "https://drone-publish.rancher.io":
		return "drone_publish_build"
	}
	return ""
}

func (i droneBuild) completed() bool {
	return i.passing || i.failing
}

func (i *droneBuild) Refresh(ctx context.Context) error {
	if i.completed() {
		return nil
	}
	client := newDroneClient(ctx, i.server, i.token)
	// id, err := strconv.Atoi(i.id)
	// if err != nil {
	// 	return err
	// }
	build, err := client.Build(i.org, i.repo, i.buildNo)
	if err != nil {
		return err
	}
	*i = newDroneBuild(i.server, i.token, i.org, i.repo, build)
	return nil
}

func newDroneClient(ctx context.Context, server, token string) drone.Client {
	conf := new(oauth2.Config)
	httpClient := conf.Client(ctx, &oauth2.Token{AccessToken: token})
	return drone.NewClient(server, httpClient)
}
