package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type addItemContentView int

const (
	addItemType addItemContentView = iota
	addDroneServer
	addItemRepo
	addItemId
)

type itemType int

const (
	droneBuildItem itemType = iota
	githubPrItem
	githubReleaseItem
)

type addItem struct {
	config   config
	view     addItemContentView
	itemType itemType
	server   string
	itemRepo textinput.Model
	itemID   textinput.Model
}

var droneServers = []string{
	droneRancherPrServer,
	droneRancherPubServer,
	droneK3sPrServer,
	droneK3sPubServer,
}

func newAddItem(config config) addItem {
	a := addItem{
		config:   config,
		view:     addItemType,
		itemType: droneBuildItem,
		itemRepo: textinput.New(),
		itemID:   textinput.New(),
	}

	a.itemRepo.Placeholder = "rancher/rancher"
	a.itemRepo.CharLimit = 48
	a.itemRepo.Width = 48

	a.itemID.CharLimit = 24
	a.itemID.Width = 24

	return a
}

func checkbox(label string, checked bool) string {
	if checked {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("[x] " + label)
	}
	return fmt.Sprintf("[ ] %s", label)
}

func (a addItem) orgRepo() (string, string) {
	parts := strings.Split(a.itemRepo.Value(), "/")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", ""
}

type newItemMsg struct {
	item listItem
	err  error
}

func (a *addItem) validateItem() tea.Cmd {
	return func() tea.Msg {
		org, repo := a.orgRepo()

		switch a.itemType {
		case droneBuildItem:
			build := newDroneBuild(a.server, org, repo, a.itemID.Value())
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := build.update(ctx, a.config); err != nil {
				return newItemMsg{err: err}
			}
			return newItemMsg{item: build}
		case githubPrItem:
			pr := pullRequest{
				number: a.itemID.Value(),
				org:    org,
				repo:   repo,
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := pr.update(ctx, a.config.GitHubToken); err != nil {
				return newItemMsg{err: err}
			}
			return newItemMsg{item: pr}
		case githubReleaseItem:
			release := release{
				tag:  a.itemID.Value(),
				org:  org,
				repo: repo,
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			release.update(ctx, a.config.GitHubToken)
			return newItemMsg{item: release}
		}

		return exitAddItemCmd()
	}
}

type exitAddItemMsg struct{}

func exitAddItemCmd() tea.Cmd {
	return func() tea.Msg {
		return exitAddItemMsg{}
	}
}

func (a addItem) Update(msg tea.Msg) (addItem, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			switch a.view {
			case addItemType:
				if a.itemType < githubReleaseItem {
					a.itemType++
					return a, nil
				}
			case addDroneServer:
				for i, server := range droneServers {
					if server == a.server {
						if i < len(droneServers)-1 {
							a.server = droneServers[i+1]
							return a, nil
						}
					}
				}
			}
		case "k", "up":
			switch a.view {
			case addItemType:
				if a.itemType > 0 {
					a.itemType--
					return a, nil
				}
			case addDroneServer:
				for i, server := range droneServers {
					if server == a.server {
						if i > 0 {
							a.server = droneServers[i-1]
							return a, nil
						}
					}
				}
			}
		case "esc":
			return a, exitAddItemCmd()
		case "enter":
			switch a.view {
			case addItemType:
				switch a.itemType {
				case githubPrItem, githubReleaseItem:
					a.view = addItemRepo
					return a, tea.Batch(a.itemRepo.Focus(), textinput.Blink)
				case droneBuildItem:
					a.server = droneRancherPrServer
					a.view = addDroneServer
				}
			case addDroneServer:
				a.view = addItemRepo
				return a, tea.Batch(a.itemRepo.Focus(), textinput.Blink)
			case addItemRepo:
				a.view = addItemId
				return a, tea.Batch(a.itemID.Focus(), textinput.Blink)
			case addItemId:
				return a, a.validateItem()
			}
			return a, nil
		}
	case error:
		fmt.Println("got an error", msg)
		return a, nil
	}

	if a.view == addItemRepo {
		var cmd tea.Cmd
		a.itemRepo, cmd = a.itemRepo.Update(msg)
		return a, cmd
	}
	if a.view == addItemId {
		var cmd tea.Cmd
		a.itemID, cmd = a.itemID.Update(msg)
		return a, cmd
	}

	return a, nil
}

func (a addItem) View() string {
	switch a.view {
	case addItemType:
		return a.chooseTypeView()
	case addDroneServer:
		return a.chooseDroneView()
	case addItemRepo:
		return a.repoInputView()
	case addItemId:
		return a.idInputView()
	}

	return ""

}

func (a addItem) chooseTypeView() string {
	t := a.itemType
	tpl := "What kind of resource do you want to watch?\n"
	tpl += "\n" + checkbox("Drone Build", t == droneBuildItem)
	tpl += "\n" + checkbox("GitHub Pull Request", t == githubPrItem)
	tpl += "\n" + checkbox("GitHub Release", t == githubReleaseItem)
	tpl += "\n\n"
	tpl += subtle.Render("j/k, up/down: select")
	tpl += dot + subtle.Render("enter: choose")
	tpl += dot + subtle.Render("q, esc: quit")

	return tpl
}

func (a addItem) chooseDroneView() string {
	tpl := "Choose Drone server:\n"

	for _, server := range droneServers {
		tpl += "\n"
		switch server {
		case droneRancherPrServer:
			tpl += checkbox("drone-pr.rancher.io", server == a.server)
		case droneRancherPubServer:
			tpl += checkbox("drone-publish.rancher.io", server == a.server)
		case droneK3sPrServer:
			tpl += checkbox("drone-pr.k3s.io", server == a.server)
		case droneK3sPubServer:
			tpl += checkbox("drone-publish.k3s.io", server == a.server)
		}
	}
	tpl += "\n\n"
	tpl += subtle.Render("j/k, up/down: select")
	tpl += dot + subtle.Render("enter: choose")
	tpl += dot + subtle.Render("q, esc: quit")
	return tpl
}

func (a addItem) repoInputView() string {
	tpl := "What is the repo?\n\n"
	tpl += a.itemRepo.View()
	tpl += "\n\n" + subtle.Render("enter: save") + dot + subtle.Render("q, esc: quit") + "\n"

	return tpl
}

func (a addItem) idInputView() string {
	var tpl string
	switch a.itemType {
	case droneBuildItem:
		tpl += "What is the drone build number?\n\n"
	case githubPrItem:
		tpl += "What is the github pull request number?\n\n"
	case githubReleaseItem:
		tpl += "What is the release tag?\n\n"
	}
	tpl += a.itemID.View()
	tpl += "\n\n" + subtle.Render("enter: submit") + dot + subtle.Render("esc: quit") + "\n"

	return tpl
}

// func (a addItem) listItem() list.Item {
// 	switch a.itemType {
// 	case "drone_pr_build", "drone_publish_build":
// 		return &droneBuild{
// 			id:    a.buildNo.Value(),
// 			title: a.itemRepo.Value(),
// 			desc:  a.itemRepo.Value(),
// 		}
// 	case "github_pull_request":
// 		return &pullRequest{
// 			id:    a.buildNo.Value(),
// 			title: a.itemRepo.Value(),
// 			desc:  a.itemRepo.Value(),
// 		}
// 	}
// 	return nil
// }
