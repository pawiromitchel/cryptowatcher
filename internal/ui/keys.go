package ui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines keybindings for the application.
type KeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Add       key.Binding
	Delete    key.Binding
	Reload    key.Binding
	Candle    key.Binding
	Quit      key.Binding
	Enter     key.Binding
	Esc       key.Binding
}

// DefaultKeyMap returns the default keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Add: key.NewBinding(
			key.WithKeys("a", "+"),
			key.WithHelp("a", "add pair"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d", "x", "delete"),
			key.WithHelp("d", "remove pair"),
		),
		Reload: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Candle: key.NewBinding(
			key.WithKeys("v", "c"),
			key.WithHelp("v", "candlestick chart"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "submit"),
		),
		Esc: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
	}
}
