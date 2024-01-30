package watch

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rancher/ecm-distro-tools/cmd/release/config"
	"github.com/rancher/ecm-distro-tools/cmd/release/watch/add"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type modelView int

const (
	listView modelView = iota
	addItemView
	detailView
)

type model struct {
	listFile        string
	refreshInterval time.Duration
	config          config.Auth
	view            modelView
	list            list.Model
	add             add.Model
}

func New(c config.Auth, listFile string) (*model, error) {
	items, err := load(listFile)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	m := &model{
		listFile:        listFile,
		refreshInterval: time.Second * 30,
		config:          c,
		view:            listView,
		list:            newList(items),
		add:             add.New(),
	}

	return m, nil
}
func (m model) Init() tea.Cmd {
	return func() tea.Msg {
		return refreshIntervalMsg{}
	}
}

func (m model) View() string {
	switch m.view {
	case listView:
		return docStyle.Render(m.list.View())
	case addItemView:
		return docStyle.Render(m.add.View())
	default:
		return ""
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
		m.add.SetHeight(msg.Height - v)
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, listKeys.quit):
			return m, tea.Quit
		}
	case refreshIntervalMsg:
		var items []item
		for _, li := range m.list.Items() {
			if item, ok := li.(item); ok && !item.Completed() {
				items = append(items, item)
			}
		}
		if len(items) == 0 {
			return m, m.refreshIntervalCmd()
		}
		m.list.Title = "Refreshing"
		return m, tea.Batch(
			m.list.StartSpinner(),
			delayCmd(time.Second*2, refreshItems(m.config, items)),
		)
	case refreshCompleteMsg:
		m.list.Title = "Watch list"
		m.list.StopSpinner()
		for _, refreshed := range msg.items {
			for i, li := range m.list.Items() {
				if li.(item).Key() != refreshed.Key() {
					continue
				}
				if refreshed.Completed() {
					fmt.Print("\a")
				}
				m.list.SetItem(i, refreshed)
			}
		}
		return m, m.refreshIntervalCmd()
	case itemRefreshedMsg:
		for i, li := range m.list.Items() {
			if li.(item).Key() == msg.i.Key() {
				return m, m.list.SetItem(i, msg.i)
			}
		}
	}

	switch m.view {
	case listView:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, listKeys.add):
				m.view = addItemView
				m.add.Reset()
				return m, nil
			case key.Matches(msg, listKeys.delete):
				li := m.list.SelectedItem()
				if i, ok := li.(item); ok {
					items := make([]list.Item, 0, len(m.list.Items())-1)
					for _, li := range m.list.Items() {
						if item, ok := li.(item); ok {
							if i.Key() != item.Key() {
								items = append(items, item)
							}
						}
					}
					m.list.SetItems(items)
					return m, m.save()
				}
			case key.Matches(msg, listKeys.clear):
				return m.clear()
			}
		}

		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	case addItemView:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, add.ExitKey):
				m.view = listView
			}
		}
		if m.add.Completed {
			m.view = listView
			i := item{
				Kind:   m.add.Kind,
				Org:    m.add.Org,
				Repo:   m.add.Repo,
				ID:     m.add.ID,
				Server: m.add.Server,
			}
			return m, tea.Batch(
				m.list.InsertItem(0, i),
				refresh(context.Background(), m.config, i),
				m.save(),
			)
		}

		var cmd tea.Cmd
		m.add, cmd = m.add.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) clear() (tea.Model, tea.Cmd) {
	items := make([]list.Item, 0, len(m.list.Items()))
	for _, i := range m.list.Items() {
		if item, ok := i.(item); ok {
			if !item.Completed() {
				items = append(items, i)
			}
		}
	}
	m.list.SetItems(items)
	return m, tea.Batch(m.save(), m.list.NewStatusMessage("Cleared completed"))
}

type refreshIntervalMsg struct{}

func (m model) refreshIntervalCmd() tea.Cmd {
	return tea.Tick(m.refreshInterval, func(time.Time) tea.Msg {
		return refreshIntervalMsg{}
	})
}

func (m model) save() tea.Cmd {
	return func() tea.Msg {
		items := make([]item, 0, len(m.list.Items()))
		for _, li := range m.list.Items() {
			items = append(items, li.(item))
		}

		err := save(m.listFile, items)
		if err != nil {
			return m.list.NewStatusMessage(err.Error())
		}
		return m.list.NewStatusMessage("Saved")
	}
}

type refreshCompleteMsg struct {
	items []item
}

// refresh returns a tea Cmd which refreshes each item
// and returns a message containing every new item.
func refreshItems(c config.Auth, items []item) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
		defer cancel()

		mu := sync.Mutex{}
		wg := sync.WaitGroup{}
		for i := range items {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				next := items[i].refresh(ctx, c)
				mu.Lock()
				defer mu.Unlock()
				items[i] = next
			}(i)
		}
		wg.Wait()
		return refreshCompleteMsg{items}
	}
}

type itemRefreshedMsg struct {
	i item
}

func refresh(ctx context.Context, c config.Auth, i item) tea.Cmd {
	return func() tea.Msg {
		next := i.refresh(ctx, c)
		return itemRefreshedMsg{next}
	}
}

// delay calls the provided command, and returns its message after the duration.
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

func save(listFile string, items []item) error {
	f, err := os.Create(listFile)
	if err != nil {
		return err
	}
	defer f.Close()

	jsonData, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}

	_, err = f.Write(jsonData)
	if err != nil {
		return err
	}

	return nil
}

func load(filename string) ([]item, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var items []item
	if err := json.NewDecoder(f).Decode(&items); err != nil {
		return nil, err
	}

	return items, nil
}
