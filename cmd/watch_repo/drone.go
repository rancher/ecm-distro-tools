package main

import (
	"context"
	"strconv"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/drone/drone-go/drone"
	"golang.org/x/oauth2"
)

var (
	droneRancherPrServer  = "https://drone-pr.rancher.io"
	droneRancherPubServer = "https://drone-publish.rancher.io"
	droneK3sPrServer      = "https://drone-pr.k3s.io"
	droneK3sPubServer     = "https://drone-publish.k3s.io"
)

type droneBuild struct {
	number string
	org    string
	repo   string
	server string
	build  drone.Build
	err    error
}

func newDroneBuild(server, org, repo, number string) droneBuild {
	return droneBuild{
		number: number,
		org:    org,
		repo:   repo,
		server: server,
	}
}

func (b droneBuild) Title() string {

	switch b.server {
	case droneRancherPrServer:
		return "[Drone PR] " + b.number
	case droneRancherPubServer:
		return "[Drone Publish] " + b.number
	case droneK3sPrServer:
		return "[k3s PR] " + b.number
	case droneK3sPubServer:
		return "[k3s Publish] " + b.number
	default:
		return "[Drone] " + b.number
	}
}

func (b droneBuild) Description() string {
	var status string
	switch b.build.Status {
	case drone.StatusPassing:
		status = greenBlock.Width(12).Align(lipgloss.Center).Render("passing")
	case drone.StatusFailing:
		status = redBlock.Width(12).Align(lipgloss.Center).Render("failing")
	case drone.StatusRunning:
		status = yellowBlock.Width(12).Align(lipgloss.Center).Render("running")
	case "":
		status = whiteBlock.Width(12).Align(lipgloss.Center).Render("unknown")
	default:
		status = whiteBlock.Width(12).Align(lipgloss.Center).Render(b.build.Status)
	}

	if b.err != nil {
		status = redBlock.Width(12).Align(lipgloss.Center).Render("error")
		return status + "  " + subtle.Render(b.org+"/"+b.repo) + "  " + b.err.Error()
	}

	var elapsed string
	if b.build.Started != 0 {
		started := time.Unix(b.build.Started, 0)
		elapsed = formatDuration(time.Since(started))
	}
	return status + "  " + subtle.Render(b.org+"/"+b.repo) + "  " + elapsed
}

func (b droneBuild) FilterValue() string {
	return "drone " + b.number + " " + b.build.Title + " " + b.org + "/" + b.repo + " " + b.build.Status
}

func (b droneBuild) Key() string {
	return "drone_build_" + b.server + "_" + b.org + "_" + b.repo + b.number
}

func (b droneBuild) Record() record {
	return record{
		Type:   "drone_build",
		Id:     b.number,
		Org:    b.org,
		Repo:   b.repo,
		Server: b.server,
	}
}

func (b droneBuild) Completed() bool {
	switch b.build.Status {
	case drone.StatusPassing, drone.StatusFailing,
		drone.StatusKilled, drone.StatusError,
		drone.StatusBlocked, drone.StatusDeclined,
		drone.StatusSkipped:
		return true
	default:
		return false
	}
}

func (b droneBuild) Refresh(ctx context.Context, c config) listItem {
	b.err = nil
	if err := b.update(ctx, c); err != nil {
		b.err = err
	}
	return b
}

func (b *droneBuild) update(ctx context.Context, config config) error {
	client := newDroneClient(ctx, b.server, config)
	number, err := strconv.Atoi(b.number)
	if err != nil {
		return err
	}
	build, err := client.Build(b.org, b.repo, number)
	if err != nil {
		return err
	}
	b.build = *build
	return nil
}

func newDroneClient(ctx context.Context, server string, config config) drone.Client {
	conf := new(oauth2.Config)
	var token string
	switch server {
	case droneRancherPrServer:
		token = config.DroneRancherPrToken
	case droneRancherPubServer:
		token = config.DroneRancherPublishToken
	case droneK3sPrServer:
		token = config.DroneK3sPrToken
	case droneK3sPubServer:
		token = config.DroneK3sPublishToken
	}
	httpClient := conf.Client(ctx, &oauth2.Token{AccessToken: token})
	return drone.NewClient(server, httpClient)
}
