package main

import (
	"encoding/json"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

type config struct {
	DroneRancherPrToken      string `json:"drone_rancher_pr_token"`
	DroneRancherPublishToken string `json:"drone_rancher_publish_token"`
	DroneK3sPrToken          string `json:"drone_k3s_pr_token"`
	DroneK3sPublishToken     string `json:"drone_k3s_publish_token"`
	GitHubToken              string `json:"github_token"`
}

func newConfig(path string) (*config, error) {
	c := &config{}
	if path == "" {
		return c, nil
	}

	if err := loadConfig(path, c); err != nil {
		return nil, err
	}

	return c, nil
}

func loadConfig(path string, c *config) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(&c)
	if err != nil {
		return err
	}

	return nil
}

func (c config) View() string {
	tpl := "Config"
	tpl += "\n  Drone Rancher PR Token:\n "
	tpl += c.DroneRancherPrToken
	tpl += "\n  Drone Rancher Publish Token:\n "
	tpl += "\n  Drone K3s PR Token:\n "
	tpl += "\n  Drone K3s Publish Token:\n "
	tpl += "\n  GitHub token:\n "

	return tpl
}

type exitConfigMsg struct{}

func exitConfig() tea.Cmd {
	return func() tea.Msg {
		return exitConfigMsg{}
	}
}

func (c config) Update(msg tea.Msg) (config, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return c, exitConfig()
		}
	}
	return c, nil
}
