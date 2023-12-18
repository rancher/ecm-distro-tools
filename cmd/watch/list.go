package main

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func newList(items []listItem) list.Model {
	listItems := make([]list.Item, 0, len(items))
	for _, item := range items {
		listItems = append(listItems, item)
	}
	delegate := newItemDelegate()
	list := list.New(listItems, delegate, 0, 0)
	list.Title = "Watch list"
	list.SetStatusBarItemName("resource", "resources")
	list.SetShowStatusBar(false)
	list.Styles.Title = list.Styles.Title.Copy().Background(selected)
	return list
}

type delegateKeyMap struct {
	add    key.Binding
	config key.Binding
	remove key.Binding
	clear  key.Binding
}

func newItemDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.Copy().BorderForeground(selected).Foreground(selected)
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.Copy().BorderForeground(selected)

	keys := delegateKeyMap{
		add: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add"),
		),
		config: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "config"),
		),
		remove: key.NewBinding(
			key.WithKeys("d", "backspace"),
			key.WithHelp("d", "delete"),
		),
		clear: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "clear completed"),
		),
	}

	d.UpdateFunc = func(msg tea.Msg, m *list.Model) tea.Cmd {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if m.FilterState() == list.Filtering {
				break
			}
			switch {
			case key.Matches(msg, keys.add):
				return viewAddItem()
			case key.Matches(msg, keys.remove):
				return removeItem(m.Index())
			case key.Matches(msg, keys.clear):
				return clearItems()
			case key.Matches(msg, keys.config):
				return viewConfig()
			}
		}

		return nil
	}

	d.ShortHelpFunc = func() []key.Binding {
		return []key.Binding{
			keys.add,
			keys.remove,
			keys.clear,
		}
	}

	d.FullHelpFunc = func() [][]key.Binding {
		return [][]key.Binding{{
			keys.add,
			keys.remove,
			keys.clear,
			keys.config,
		}}
	}

	return d
}

type removeItemMsg struct {
	index int
}

func removeItem(index int) tea.Cmd {
	return func() tea.Msg {
		return removeItemMsg{index: index}
	}
}

type clearItemsMsg struct{}

func clearItems() tea.Cmd {
	return func() tea.Msg {
		return clearItemsMsg{}
	}
}

type viewAddItemMsg struct{}

func viewAddItem() tea.Cmd {
	return func() tea.Msg {
		return viewAddItemMsg{}
	}
}

type viewConfigMsg struct{}

func viewConfig() tea.Cmd {
	return func() tea.Msg {
		return viewConfigMsg{}
	}
}
