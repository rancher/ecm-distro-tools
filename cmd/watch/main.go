package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type view int

const (
	listView view = iota
	addItemView
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

var dot = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(" • ")

type config struct {
	drone_pr_token      string
	drone_publish_token string
	github_token        string
}

type model struct {
	filename string
	items    []listItem
	// builds  chan *drone.Build
	refreshing bool
	config     config
	visible    view
	list       list.Model
	addItem    addItem
}

func subtle(s string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(s)
}

func listenForActivity(sub chan struct{}) tea.Cmd {
	return func() tea.Msg {
		for {
			time.Sleep(time.Millisecond * time.Duration(rand.Int63n(900)+100)) // nolint:gosec
			sub <- struct{}{}
		}
	}
}

func waitForActivity(sub chan *droneBuild) tea.Cmd {
	return func() tea.Msg {
		return <-sub
	}
}

type refreshCountdownCompleteMsg struct{}

func startRefreshCountdown(t time.Duration) tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		time.Sleep(t)
		return refreshCountdownCompleteMsg{}
	})
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		// startRefreshCountdown(0),
		m.refresh(),
	)
}

// Insert saves the item record to the local file
// and inserts it into the list. It returns a command
// to update the filtered list.
func (m model) save() error {
	return errors.New("failed to save")
}

// delay executes a command, and returns its message after a minimum duration.
func delayCmd(d time.Duration, cmd tea.Cmd) tea.Cmd {
	return func() tea.Msg {
		ch := make(chan tea.Msg)
		go func() {
			ch <- cmd()
		}()

		<-time.After(d)
		return <-ch
	}
}

func (m model) refresh() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		msg := updateItemsMsg{
			items: make([]listItem, len(m.items)),
		}
		mu := sync.Mutex{}
		wg := sync.WaitGroup{}
		for i, item := range m.items {
			wg.Add(1)
			go func(i int, item listItem) {
				defer wg.Done()
				item.Refresh(ctx)
				mu.Lock()
				msg.items[i] = item
				mu.Unlock()
			}(i, item)
		}

		wg.Wait()
		return msg
	}
}

type updateItemsMsg struct {
	items []listItem
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case refreshCountdownCompleteMsg:
		m.refreshing = true
		m.list.Title = "Refreshing"
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, tea.Batch(cmd, m.list.StartSpinner(), delayCmd(time.Second*2, m.refresh()))
	case updateItemsMsg:
		m.refreshing = false
		m.list.StopSpinner()
		m.list.Title = "Watch list"
		for _, msgItem := range msg.items {
			for i, item := range m.items {
				if item.ID() == msgItem.ID() {
					m.items[i] = msgItem
					m.list.SetItem(i, msgItem)
				}
			}
		}
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, tea.Batch(cmd, startRefreshCountdown(time.Second*15))
		// return m, cmd
	case addItemExitMsg:
		m.visible = listView
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)

	case addItemMsg:
		// add the build to the top of the list
		m.items = append([]listItem{msg.item}, m.items...)
		ins := m.list.InsertItem(0, msg.item)
		// save the updated list to the local file
		err := m.save()
		if err != nil {
			m.addItem = newAddItem(m.config)
			m.visible = listView
			return m, m.list.NewStatusMessage(err.Error())
		}
		// reset the add item view model
		m.addItem = newAddItem(m.config)
		// return to the list view
		m.visible = listView
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, tea.Batch(ins, cmd)
	case kubernetesUpstreamRelease:
		// kubernetes releases are static placeholders.
		m.visible = listView
		ins := m.list.InsertItem(0, msg)
		m.addItem = newAddItem(m.config)
		return m, ins
	}

	switch m.visible {
	case listView:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "a":
				m.visible = addItemView
				m.addItem = newAddItem(m.config)
				return m, nil
			case "r":
				if m.refreshing || len(m.list.Items()) == 0 {
					return m, nil
				}
				m.refreshing = true
				list, cmd := m.list.Update(msg)
				m.list = list
				return m, tea.Batch(cmd, m.refresh())
			}
		}
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	case addItemView:
		var cmd tea.Cmd
		m.addItem, cmd = m.addItem.Update(msg)
		// if m.addItem.validated {
		// 	// print("Add Item view is validated, appending and changing to the list view")
		// 	m.visible = listView
		// 	m.addItem = newAddItem(m.config)
		// 	ins := m.list.InsertItem(0, m.addItem.listItem())
		// 	return m, tea.Batch(ins, cmd)
		// }
		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	switch m.visible {
	case listView:
		return docStyle.Render(m.list.View())
	case addItemView:
		return docStyle.Render(m.addItem.View())
	}

	panic("unreachable")
}

type listItem interface {
	Title() string
	Description() string
	FilterValue() string
	ID() string
	BuildNo() int
	Org() string
	Repo() string
	Type() string
	Refresh(ctx context.Context) error
}

func main() {
	config := config{
		drone_pr_token:      os.Getenv("DRONE_PR_TOKEN"),
		drone_publish_token: os.Getenv("DRONE_PUBLISH_TOKEN"),
		github_token:        os.Getenv("GITHUB_TOKEN"),
	}

	var items []listItem
	if _, err := os.Stat("data.json"); err == nil {
		f, err := os.Open("data.json")
		if err != nil {
			fmt.Println("Error opening JSON file:", err)
			os.Exit(1)
		}
		defer f.Close()

		records, err := deserializeRecords(f)
		if err != nil {
			fmt.Println("Error deserializing items:", err)
			os.Exit(1)
		}
		for _, r := range records {
			switch r.Type {
			case "github_pull_request":
				items = append(items, &pullRequest{
					id:   r.Id,
					org:  r.Org,
					repo: r.Repo,
				})
			case "drone_pr_build":
				buildNo, err := strconv.Atoi(r.Id)
				if err != nil {
					fmt.Println("Error converting build number to int:", err)
					os.Exit(1)
				}
				items = append(items, &droneBuild{
					buildNo: buildNo,
					org:     r.Org,
					repo:    r.Repo,
					server:  "https://drone-pr.rancher.io",
					token:   config.drone_pr_token,
				})
			case "drone_publish_build":
				buildNo, err := strconv.Atoi(r.Id)
				if err != nil {
					fmt.Println("Error converting build number to int:", err)
					os.Exit(1)
				}
				items = append(items, &droneBuild{
					buildNo: buildNo,
					org:     r.Org,
					repo:    r.Repo,
					server:  "https://drone-publish.rancher.io",
					token:   config.drone_publish_token,
				})
			default:
				fmt.Printf("Unknown item type: %s\n", r.Type)
				os.Exit(1)
			}
		}
	} else if os.IsNotExist(err) {
		items = []listItem{}
	} else {
		fmt.Println("Error checking if JSON file exists:", err)
		os.Exit(1)
	}

	// for _, item := range items {

	// listDelegate := list.NewDefaultDelegate()

	// listKeys := newListKeys()
	delegateKeys := newDelegateKeyMap()
	listDelegate := newItemDelegate(delegateKeys)

	var itemList []list.Item
	for _, item := range items {
		itemList = append(itemList, item)
	}

	m := model{
		items:   items,
		config:  config,
		visible: listView,
		list:    list.New(itemList, listDelegate, 0, 0),
	}
	m.list.Title = "Watch list"
	m.list.SetStatusBarItemName("resource", "resources")
	m.list.SetShowStatusBar(false)
	// m.list.AdditionalShortHelpKeys = func() []key.Binding {
	// 	return []key.Binding{
	// 		listKeys.addItem,
	// 	}
	// }
	// p := tea.NewProgram(m, tea.WithAltScreen())
	p := tea.NewProgram(m)

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

}
