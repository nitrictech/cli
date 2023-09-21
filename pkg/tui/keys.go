package tui

import "github.com/charmbracelet/bubbles/key"

type DefaultKeyMap struct {
	Enter key.Binding
	Quit  key.Binding
	Up    key.Binding
	Down  key.Binding
}

var KeyMap = DefaultKeyMap{
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "submit input"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "exit"),
	),
	Up: key.NewBinding(
		key.WithKeys("up"),
		key.WithHelp("↑", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down"),
		key.WithHelp("↓", "down"),
	),
}
