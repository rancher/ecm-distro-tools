package watch

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/rancher/ecm-distro-tools/cmd/release/watch/style"
)

func newList(items []item) list.Model {
	listItems := make([]list.Item, 0, len(items))
	for _, item := range items {
		listItems = append(listItems, item)
	}

	m := list.New(listItems, newItemDelegate(), 0, 0)
	m.Title = "Watch list"
	m.SetStatusBarItemName("resource", "resources")
	m.SetShowStatusBar(false)
	m.SetFilteringEnabled(false)
	m.DisableQuitKeybindings()
	m.Styles.Title = m.Styles.Title.Copy().Background(style.Highlight)

	return m
}

type listKeyMap struct {
	quit   key.Binding
	add    key.Binding
	delete key.Binding
	clear  key.Binding
}

var listKeys = listKeyMap{
	quit: key.NewBinding(
		key.WithKeys("ctrl+c", "q"),
		key.WithHelp("q", "quit"),
	),
	add: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "add"),
	),
	delete: key.NewBinding(
		key.WithKeys("d", "backspace"),
		key.WithHelp("d", "delete"),
	),
	clear: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "clear completed"),
	),
}

func newItemDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.Copy().BorderForeground(style.Highlight).Foreground(style.Highlight)
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.Copy().BorderForeground(style.Highlight)

	d.ShortHelpFunc = func() []key.Binding {
		return []key.Binding{
			listKeys.add,
			listKeys.delete,
			listKeys.clear,
			listKeys.quit,
		}
	}

	d.FullHelpFunc = func() [][]key.Binding {
		return [][]key.Binding{{
			listKeys.add,
			listKeys.delete,
			listKeys.clear,
			listKeys.quit,
		}}
	}

	return d
}
