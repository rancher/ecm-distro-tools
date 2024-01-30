package add

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rancher/ecm-distro-tools/cmd/release/watch/drone"
	"github.com/rancher/ecm-distro-tools/cmd/release/watch/github"
)

var (
	cursorUp = key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	)
	cursorDown = key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	)
	ExitKey = key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	)
	returnKey = key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "continue"),
	)
	kinds = []string{
		drone.BuildKind,
		github.PullRequestKind,
		github.ReleaseKind,
	}
)

type view int

const (
	selectKind view = iota
	selectDroneServer
	inputRepo
	inputId
)

type Model struct {
	view        view
	height      int
	help        help.Model
	kind        int
	droneServer int
	itemRepo    textinput.Model
	itemID      textinput.Model
	Org         string
	Repo        string
	ID          string
	Server      string
	Kind        string
	Error       error
	Completed   bool
}

var droneServers = []string{
	drone.RancherPrServer,
	drone.RancherPubServer,
	drone.K3sPrServer,
	drone.K3sPubServer,
}

func New() Model {
	m := Model{
		help:     help.New(),
		view:     selectKind,
		kind:     0,
		itemRepo: textinput.New(),
		itemID:   textinput.New(),
	}

	m.itemRepo.Placeholder = "rancher/rancher"
	m.itemRepo.CharLimit = 48
	m.itemRepo.Width = 48

	m.itemID.CharLimit = 24
	m.itemID.Width = 24

	return m
}

// Reset sets the form to its initial state
func (m *Model) Reset() {
	m.view = selectKind
	m.itemRepo.Reset()
	m.itemID.Reset()
	m.kind = 0
	m.droneServer = 0
	m.Org = ""
	m.Repo = ""
	m.ID = ""
	m.Server = ""
	m.Kind = ""
	m.Completed = false
}

func (m *Model) SetHeight(height int) {
	m.height = height
}

func checkbox(label string, checked bool) string {
	if checked {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("(•) " + label)
	}
	return fmt.Sprintf("( ) %s", label)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, cursorDown):
			switch m.view {
			case selectKind:
				if m.kind < len(kinds)-1 {
					m.kind++
				}
				return m, nil
			case selectDroneServer:
				if m.droneServer < len(droneServers)-1 {
					m.droneServer++
				}
				return m, nil
			}
		case key.Matches(msg, cursorUp):
			switch m.view {
			case selectKind:
				if m.kind > 0 {
					m.kind--
				}
				return m, nil
			case selectDroneServer:
				if m.droneServer > 0 {
					m.droneServer--
				}
				return m, nil
			}
		case key.Matches(msg, returnKey):
			switch m.view {
			case selectKind:
				m.Kind = kinds[m.kind]
				switch kinds[m.kind] {
				case github.PullRequestKind, github.ReleaseKind:
					m.view = inputRepo
					return m, tea.Batch(m.itemRepo.Focus(), textinput.Blink)
				case drone.BuildKind:
					m.droneServer = 0
					m.view = selectDroneServer
				}
				return m, nil
			case selectDroneServer:
				m.Server = droneServers[m.droneServer]
				m.view = inputRepo
				return m, tea.Batch(m.itemRepo.Focus(), textinput.Blink)
			case inputRepo:
				parts := strings.Split(m.itemRepo.Value(), "/")
				if len(parts) == 2 {
					m.Org, m.Repo = parts[0], parts[1]
				}
				m.view = inputId
				return m, tea.Batch(m.itemID.Focus(), textinput.Blink)
			case inputId:
				m.ID = m.itemID.Value()
				m.Completed = true
				return m, nil
			}
			return m, nil
		}
	}

	if m.view == inputRepo {
		var cmd tea.Cmd
		m.itemRepo, cmd = m.itemRepo.Update(msg)
		return m, cmd
	}
	if m.view == inputId {
		var cmd tea.Cmd
		m.itemID, cmd = m.itemID.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) pageContent() string {
	switch m.view {
	case selectKind:
		return m.selectKindView()
	case selectDroneServer:
		return m.selectDroneServerView()
	case inputRepo:
		return m.repoInputView()
	case inputId:
		return m.idInputView()
	default:
		return ""
	}
}

func (m Model) ShortHelp() []key.Binding {
	return []key.Binding{cursorUp, cursorDown, ExitKey, returnKey}
}

func (m Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{{cursorUp, cursorDown, ExitKey, returnKey}}

}

func (m Model) View() string {
	help := m.help.View(m)
	height := m.height - lipgloss.Height(help)
	content := lipgloss.NewStyle().Height(height).Render(m.pageContent())

	return lipgloss.JoinVertical(lipgloss.Left, content, help)
}

func (m Model) selectKindView() string {
	tpl := "What kind of resource do you want to watch?\n"
	for i, t := range kinds {
		tpl += "\n"
		switch t {
		case drone.BuildKind:
			tpl += checkbox("Drone Build", i == m.kind)
		case github.PullRequestKind:
			tpl += checkbox("GitHub Pull Request", i == m.kind)
		case github.ReleaseKind:
			tpl += checkbox("GitHub Release", i == m.kind)
		}
	}

	return tpl
}

func (m Model) selectDroneServerView() string {
	tpl := "Choose Drone server:\n"

	for i, server := range droneServers {
		tpl += "\n"
		switch server {
		case drone.RancherPrServer:
			tpl += checkbox("drone-pr.rancher.io", i == m.droneServer)
		case drone.RancherPubServer:
			tpl += checkbox("drone-publish.rancher.io", i == m.droneServer)
		case drone.K3sPrServer:
			tpl += checkbox("drone-pr.k3s.io", i == m.droneServer)
		case drone.K3sPubServer:
			tpl += checkbox("drone-publish.k3s.io", i == m.droneServer)
		}
	}
	return tpl
}

func (m Model) repoInputView() string {
	tpl := "What is the repo?\n\n"
	tpl += m.itemRepo.View()

	return tpl
}

func (m Model) idInputView() string {
	var tpl string
	switch kinds[m.kind] {
	case drone.BuildKind:
		tpl += "What is the drone build number?\n\n"
	case github.PullRequestKind:
		tpl += "What is the github pull request number?\n\n"
	case github.ReleaseKind:
		tpl += "What is the release tag?\n\n"
	}
	tpl += m.itemID.View()

	return tpl
}
