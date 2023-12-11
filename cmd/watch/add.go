package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/drone/drone-go/drone"
)

type addItemContentView int

type addItemExitMsg struct{}

// type addItemValidatedMsg struct {
// 	id string
// }

const (
	addItemType addItemContentView = iota
	addItemRepo
	addBuildNo
)

//	type droneBuildClient interface {
//		Build(string, string, int) (*drone.Build, error)
//	}
type addItemMsg struct {
	item listItem
}

type addItem struct {
	config    config
	view      addItemContentView
	itemType  string
	itemRepo  textinput.Model
	buildNo   textinput.Model
	validated bool
}

var itemTypes = []string{
	"drone_pr_build",
	"drone_publish_build",
	"github_pull_request",
	"kubernetes_upstream_release",
}

func newAddItem(config config) addItem {
	a := addItem{
		config:    config,
		view:      addItemType,
		itemType:  "drone_pr_build",
		itemRepo:  textinput.New(),
		buildNo:   textinput.New(),
		validated: false,
	}

	a.itemRepo.Placeholder = "rancher/rke2"
	a.itemRepo.CharLimit = 28
	a.itemRepo.Width = 30

	a.buildNo.Placeholder = "1234"
	a.buildNo.CharLimit = 10
	a.buildNo.Width = 30

	return a
}

func checkbox(label string, checked bool) string {
	if checked {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("[x] " + label)
	}
	return fmt.Sprintf("[ ] %s", label)
}

func (a addItem) validateKubernetesUpstreamRelease() tea.Cmd {
	return func() tea.Msg {
		return kubernetesUpstreamRelease{
			version:   a.buildNo.Value(),
			published: false,
		}
	}
}

func (a *addItem) validateDroneBuild() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var server string
		var token string
		switch a.itemType {
		case "drone_pr_build":
			server = "https://drone-pr.rancher.io"
			token = a.config.drone_pr_token
		case "drone_publish_build":
			server = "https://drone-publish.rancher.io"
			token = a.config.drone_publish_token
		}

		client := newDroneClient(ctx, server, token)

		var org, repo string
		parts := strings.Split(a.itemRepo.Value(), "/")
		if len(parts) == 2 {
			org = parts[0]
			repo = parts[1]
		} else {
			return errors.New("invalid repo")
		}
		id, err := strconv.Atoi(a.buildNo.Value())
		if err != nil {
			return err
		}
		build, err := client.Build(org, repo, id)
		if err != nil {
			return err
		}
		return addItemMsg{
			item: &droneBuild{
				id:      strconv.Itoa(int(build.ID)),
				title:   build.Title,
				desc:    build.Message,
				org:     org,
				repo:    repo,
				passing: build.Status == drone.StatusPassing,
				failing: build.Status == drone.StatusFailing,
				running: build.Status == drone.StatusRunning,
				elapsed: formatDurationSince(build.Started),
				server:  server,
				token:   token,
			},
		}
	}
}

func exitAddItem() tea.Cmd {
	return func() tea.Msg {
		return addItemExitMsg{}
	}
}

// func validatedAddItem(*drone.Build) tea.Cmd {
// 	return func() tea.Msg {
// 		return addItemValidatedMsg{}
// 	}
// }

func (v addItem) Update(msg tea.Msg) (addItem, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if v.view == addItemType {
				for i, itemType := range itemTypes {
					if itemType == v.itemType {
						if i < len(itemTypes)-1 {
							v.itemType = itemTypes[i+1]
							break
						}
					}
				}
			}
		case "k", "up":
			if v.view == addItemType {
				for i, itemType := range itemTypes {
					if itemType == v.itemType {
						if i > 0 {
							v.itemType = itemTypes[i-1]
							break
						}
					}
				}
			}
		case "esc":
			return v, exitAddItem()

		case "enter":
			if v.view == addItemType {
				if v.itemType == "kubernetes_upstream_release" {
					v.view = addBuildNo
					v.buildNo.Placeholder = "v1.29.0"
					return v, tea.Batch(v.buildNo.Focus(), textinput.Blink)

				} else {
					v.view = addItemRepo
					return v, tea.Batch(v.itemRepo.Focus(), textinput.Blink)
				}
			}
			if v.view == addItemRepo {
				v.view = addBuildNo
				return v, tea.Batch(v.buildNo.Focus(), textinput.Blink)
			}
			if v.view == addBuildNo {
				if v.itemType == "drone_pr_build" || v.itemType == "drone_publish_build" {
					return v, v.validateDroneBuild()
				}
				if v.itemType == "kubernetes_upstream_release" {
					return v, v.validateKubernetesUpstreamRelease()
				}
			}
			return v, nil
		}
	case error:
		fmt.Println("got an error", msg)
		return v, nil
		// case *drone.Build:
		// 	fmt.Println("got the drone build down in add item", msg)
		// 	v.validated = true
		// 	return v, nil

	}

	if v.view == addItemRepo {
		var cmd tea.Cmd
		v.itemRepo, cmd = v.itemRepo.Update(msg)
		return v, cmd
	}
	if v.view == addBuildNo {
		var cmd tea.Cmd
		v.buildNo, cmd = v.buildNo.Update(msg)
		return v, cmd
	}

	return v, nil
}

func (v addItem) View() string {
	switch v.view {
	case addItemType:
		return v.chooseTypeView()
	case addItemRepo:
		return v.repoInputView()
	case addBuildNo:
		return v.idInputView()
	}

	return ""

}

func (v addItem) chooseTypeView() string {
	tpl := "What kind of resource do you want to watch?\n\n"
	for _, it := range itemTypes {
		tpl += "\n" + checkbox(it, it == v.itemType)
	}
	tpl += "\n\n"
	tpl += subtle("j/k, up/down: select") + dot + subtle("enter: choose") + dot + subtle("q, esc: quit")

	return tpl
}

func (v addItem) repoInputView() string {
	tpl := "What is the repo?\n\n"
	tpl += v.itemRepo.View()
	tpl += "\n\n" + subtle("enter: save") + dot + subtle("q, esc: quit") + "\n"

	return tpl
}

func (v addItem) idInputView() string {
	var tpl string
	switch v.itemType {
	case "drone_pr_build", "drone_publish_build":
		tpl += "What is the drone build id?\n\n"
	case "github_pull_request":
		tpl += "What is the github pull request id?\n\n"
	case "kubernetes_upstream_release":
		tpl += "This option will watch for a new Kubernetes release"
		tpl += "\nand then tag a new image-build-kubernetes release"
		tpl += "\n\nWhat is the Kubernetes version?\n\n"
	}
	tpl += v.buildNo.View()
	tpl += "\n\n" + subtle("enter: submit") + dot + subtle("q, esc: quit") + "\n"

	return tpl
}

// func (v addItem) listItem() list.Item {
// 	switch v.itemType {
// 	case "drone_pr_build", "drone_publish_build":
// 		return &droneBuild{
// 			id:    v.buildNo.Value(),
// 			title: v.itemRepo.Value(),
// 			desc:  v.itemRepo.Value(),
// 		}
// 	case "github_pull_request":
// 		return &pullRequest{
// 			id:    v.buildNo.Value(),
// 			title: v.itemRepo.Value(),
// 			desc:  v.itemRepo.Value(),
// 		}
// 	}
// 	return nil
// }
