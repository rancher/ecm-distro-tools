package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("failed to get home dir:", err)
		os.Exit(1)
	}
	dir := home + "/.ecm-distro-tools/watch"
	configPath := dir + "/config.json"
	listPath := dir + "/watch_list.json"
	config, err := newConfig(configPath)
	if err != nil {
		fmt.Println("failed to load config:", err)
		os.Exit(1)
	}
	items, err := deserialize(listPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("failed to load config:", err)
			os.Exit(1)
		}
	}

	m := model{
		filename: dir + "/watch_list.json",
		config:   *config,
		interval: time.Second * 30,
		visible:  listView,
		items:    items,
		list:     newList(items),
		addItem:  newAddItem(*config),
	}

	p := tea.NewProgram(m)

	if _, err := p.Run(); err != nil {
		fmt.Println("failed to run program:", err)
		os.Exit(1)
	}
}

type listItem interface {
	Title() string
	Description() string
	FilterValue() string
	Completed() bool
	Key() string
	Record() record
	Refresh(context.Context, config) listItem
}

type modelView int

const (
	listView modelView = iota
	addItemView
	configView
)

type model struct {
	filename   string
	items      []listItem
	interval   time.Duration
	refreshing bool
	config     config
	visible    modelView
	list       list.Model
	addItem    addItem
}

func (m model) Init() tea.Cmd {
	return startRefreshInterval(0)
}

func (m model) View() string {
	switch m.visible {
	case listView:
		return docStyle.Render(m.list.View())
	case addItemView:
		return docStyle.Render(m.addItem.View())
	case configView:
		return docStyle.Render(m.config.View())
	default:
		return docStyle.Render("view missing")
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}
	case removeItemMsg:
		return m.deleteSelected(msg)
	case clearItemsMsg:
		return m.clearCompleted()
	case refreshMsg:
		var incomplete []listItem
		for _, item := range m.items {
			if !item.Completed() {
				incomplete = append(incomplete, item)
			}
		}
		if len(incomplete) == 0 {
			return m, startRefreshInterval(m.interval)
		}
		m.refreshing = true
		m.list.Title = "Refreshing"
		return m, tea.Batch(
			m.list.StartSpinner(),
			delayCmd(time.Second*2, m.refreshCmd(incomplete)),
		)
	case refreshCompleteMsg:
		return m.updateList(msg)
	case viewConfigMsg:
		m.visible = configView
	case exitConfigMsg:
		m.visible = listView
	case viewAddItemMsg:
		m.visible = addItemView
		m.addItem = newAddItem(m.config)
	case exitAddItemMsg:
		m.visible = listView
	case newItemMsg:
		m.visible = listView
		if msg.err != nil {
			m.list.NewStatusMessage(msg.err.Error())
			return m, nil
		}
		m.items = append([]listItem{msg.item}, m.items...)

		return m, tea.Batch(m.list.InsertItem(0, msg.item), m.saveCmd())
	}

	switch m.visible {
	case listView:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	case addItemView:
		var cmd tea.Cmd
		m.addItem, cmd = m.addItem.Update(msg)
		return m, cmd
	case configView:
		var cmd tea.Cmd
		m.config, cmd = m.config.Update(msg)
		return m, cmd
	}

	return m, nil
}

type refreshMsg struct{}

func startRefreshInterval(t time.Duration) tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		time.Sleep(t)
		return refreshMsg{}
	})
}

func (m model) saveCmd() tea.Cmd {
	return func() tea.Msg {
		err := serialize(m.filename, m.items)
		if err != nil {
			return m.list.NewStatusMessage(err.Error())
		}
		return m.list.NewStatusMessage("Saved")
	}
}

// refresh calls refresh on each incomplete item concurrently
func (m model) refresh(ctx context.Context, items []listItem) []listItem {
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}
	for i, item := range items {
		wg.Add(1)
		go func(i int, item listItem) {
			defer wg.Done()
			result := item.Refresh(ctx, m.config)
			mu.Lock()
			defer mu.Unlock()
			items[i] = result
		}(i, item)
	}
	wg.Wait()
	return items
}

type refreshCompleteMsg struct {
	items []listItem
}

func (m model) refreshCmd(items []listItem) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
		defer cancel()
		return refreshCompleteMsg{
			items: m.refresh(ctx, items),
		}
	}
}

func (m model) updateList(msg refreshCompleteMsg) (tea.Model, tea.Cmd) {
	m.refreshing = false
	m.list.Title = "Watch list"
	m.list.StopSpinner()

	for _, msgItem := range msg.items {
		if msgItem.Completed() {
			fmt.Print("\a")
		}
		for i, item := range m.items {
			if item.Key() == msgItem.Key() {
				m.items[i] = msgItem
				m.list.SetItem(i, msgItem)
			}
		}
	}

	return m, startRefreshInterval(time.Second * 30)
}

func (m model) deleteSelected(msg removeItemMsg) (tea.Model, tea.Cmd) {
	i := msg.index
	m.items = append(m.items[:i], m.items[i+1:]...)
	m.list.RemoveItem(i)

	return m, tea.Batch(m.list.NewStatusMessage("Deleted"), m.saveCmd())
}

func (m model) clearCompleted() (tea.Model, tea.Cmd) {
	var items []listItem
	for _, item := range m.items {
		if item.Completed() {
			continue
		}
		items = append(items, item)
	}

	count := len(m.items) - len(items)
	if count == 0 {
		return m, m.list.NewStatusMessage("Nothing to clear")
	}
	m.items = items
	m.list = newList(m.items)

	return m, tea.Batch(
		m.list.NewStatusMessage("Cleared "+strconv.Itoa(count)+" completed builds"),
		m.saveCmd(),
	)
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
