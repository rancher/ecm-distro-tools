package drone

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/drone/drone-go/drone"
	"github.com/rancher/ecm-distro-tools/cmd/release/config"
	"github.com/rancher/ecm-distro-tools/cmd/release/watch/style"
)

var (
	BuildKind        = "drone_build"
	RancherPrServer  = "https://drone-pr.rancher.io"
	RancherPubServer = "https://drone-publish.rancher.io"
	K3sPrServer      = "https://drone-pr.k3s.io"
	K3sPubServer     = "https://drone-publish.k3s.io"
)

type Build struct {
	number string
	org    string
	repo   string
	server string
	build  *drone.Build
	err    error
}

func New(ctx context.Context, c config.Drone, server, org, repo, id string) *Build {
	build := &Build{
		number: id,
		org:    org,
		repo:   repo,
		server: server,
	}
	client := newClient(ctx, server, c)
	number, err := strconv.Atoi(id)
	if err != nil {
		build.err = err
		return build
	}
	b, err := client.Build(org, repo, number)
	if err != nil {
		build.err = err
		return build
	}
	build.build = b
	return build
}

func (b Build) Title() string {
	var prefix string
	switch b.server {
	case RancherPrServer, K3sPrServer:
		prefix = "[drone-pr]"
	case RancherPubServer, K3sPubServer:
		prefix = "[drone-publish]"
	default:
		prefix = "[drone]"
	}

	if b.build == nil {
		return fmt.Sprintf("%s %s", prefix, b.number)
	}

	return fmt.Sprintf("%s %s %s", prefix, b.number, b.build.Title)
}

func (b Build) Status() string {
	if b.err != nil {
		return style.Badge("error", style.Red)
	}
	if b.build == nil {
		return style.Badge("unavailable", style.Contrast)
	}

	var s string
	switch b.build.Status {
	case drone.StatusPassing:
		s = style.Badge("passing", style.Green)
	case drone.StatusFailing:
		s = style.Badge("failing", style.Red)
	case drone.StatusRunning:
		s = style.Badge("running", style.Yellow)
	default:
		s = style.Badge(b.build.Status, style.Yellow)
	}

	if b.build.Started != 0 {
		return s + " " + style.FormatDuration(time.Since(time.Unix(b.build.Started, 0)))
	}

	return s
}

func (b Build) Error() error {
	return b.err
}

func (b Build) FilterValue() string {
	var title string
	if b.build != nil {
		title = b.build.Title
	}
	return fmt.Sprintf("drone %s %s %s %s", title, b.org, b.repo, b.number)
}

func (b Build) Completed() bool {
	if b.build == nil {
		return false
	}
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
