package main

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// type listKeyMap struct {
// 	addItem key.Binding
// }

// func newListKeys() *listKeyMap {
// 	return &listKeyMap{
// 		addItem: key.NewBinding(
// 			key.WithKeys("a"),
// 			key.WithHelp("a", "add item"),
// 		),
// 	}
// }

// type delegateKeyMap struct {
// 	add    key.Binding
// 	remove key.Binding
// }

// type item interface {
// 	Title() string
// 	Description() string
// }

// func newDelegateKeyMap() *delegateKeyMap {
// 	return &delegateKeyMap{
// 		add: key.NewBinding(
// 			key.WithKeys("a"),
// 			key.WithHelp("a", "choose"),
// 		),
// 		remove: key.NewBinding(
// 			key.WithKeys("x", "backspace"),
// 			key.WithHelp("x", "delete"),
// 		),
// 	}
// }

// delegate

func newItemDelegate(keys *delegateKeyMap) list.DefaultDelegate {
	d := list.NewDefaultDelegate()

	d.UpdateFunc = func(msg tea.Msg, m *list.Model) tea.Cmd {

		// if i, ok := m.SelectedItem().(item); ok {
		// 	title = i.Title()
		// } else {
		// 	return nil
		// }

		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.add):
				return m.NewStatusMessage("You chose ADD")

			case key.Matches(msg, keys.remove):
				x := m.SelectedItem()
				if x == nil {
					return nil
				}
				index := m.Index()
				m.RemoveItem(index)
				if len(m.Items()) == 0 {
					keys.remove.SetEnabled(false)
				}
				return m.NewStatusMessage("Deleted build")
			}
		}

		return nil
	}

	help := []key.Binding{keys.add, keys.remove}

	d.ShortHelpFunc = func() []key.Binding {
		return help
	}

	d.FullHelpFunc = func() [][]key.Binding {
		return [][]key.Binding{help}
	}

	return d
}

type delegateKeyMap struct {
	add     key.Binding
	refresh key.Binding
	remove  key.Binding
}

// Additional short help entries. This satisfies the help.KeyMap interface and
// is entirely optional.
func (d delegateKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		d.add,
		d.refresh,
		d.remove,
	}
}

// Additional full help entries. This satisfies the help.KeyMap interface and
// is entirely optional.
func (d delegateKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{
			d.add,
			d.refresh,
			d.remove,
		},
	}
}

func newDelegateKeyMap() *delegateKeyMap {
	return &delegateKeyMap{
		add: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add"),
		),
		refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		remove: key.NewBinding(
			key.WithKeys("x", "backspace"),
			key.WithHelp("x", "delete"),
		),
	}
}
