package ui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the application keybindings; it also drives the help footer.
type KeyMap struct {
	Left, Right, Up, Down  key.Binding
	Detail                 key.Binding
	Add, Delete            key.Binding
	MoveEarlier, MoveLater key.Binding
	Sort                   key.Binding
	Refresh                key.Binding
	Help                   key.Binding
	Quit                   key.Binding
	Confirm, Cancel        key.Binding
}

// DefaultKeyMap returns the default keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Left:        key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "left")),
		Right:       key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "right")),
		Up:          key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:        key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Detail:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "details")),
		Add:         key.NewBinding(key.WithKeys("a", "+"), key.WithHelp("a", "add ticker")),
		Delete:      key.NewBinding(key.WithKeys("d", "x", "delete"), key.WithHelp("d", "remove")),
		MoveEarlier: key.NewBinding(key.WithKeys("["), key.WithHelp("[", "move earlier")),
		MoveLater:   key.NewBinding(key.WithKeys("]"), key.WithHelp("]", "move later")),
		Sort:        key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort by 24h")),
		Refresh:     key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Help:        key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "more")),
		Quit:        key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Confirm:     key.NewBinding(key.WithKeys("y", "Y", "enter"), key.WithHelp("y", "confirm")),
		Cancel:      key.NewBinding(key.WithKeys("n", "N", "esc", "q"), key.WithHelp("esc", "cancel")),
	}
}

// ShortHelp implements help.KeyMap.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Detail, k.Add, k.Delete, k.Refresh, k.Help, k.Quit}
}

// FullHelp implements help.KeyMap.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Left, k.Right, k.Up, k.Down},
		{k.Detail, k.Add, k.Delete, k.Refresh},
		{k.MoveEarlier, k.MoveLater, k.Sort},
		{k.Help, k.Quit},
	}
}
