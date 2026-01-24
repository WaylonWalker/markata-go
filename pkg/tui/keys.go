package tui

import "github.com/charmbracelet/bubbles/key"

type keyMapType struct {
	Up      key.Binding
	Down    key.Binding
	Quit    key.Binding
	Filter  key.Binding
	Command key.Binding
	Help    key.Binding
	Posts   key.Binding
	Tags    key.Binding
	Enter   key.Binding
	Escape  key.Binding
	Sort    key.Binding
}

var keyMap = keyMapType{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Filter: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "filter"),
	),
	Command: key.NewBinding(
		key.WithKeys(":"),
		key.WithHelp(":", "command"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Posts: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "posts"),
	),
	Tags: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "tags"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
	Sort: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "sort"),
	),
}
