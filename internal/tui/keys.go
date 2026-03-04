package tui

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Enter     key.Binding
	New       key.Binding
	Delete    key.Binding
	PaneLeft  key.Binding
	PaneRight key.Binding
	Broadcast key.Binding
	Diff      key.Binding
	Rename    key.Binding
	Filter    key.Binding
	Tab       key.Binding
	MarkRead  key.Binding
	JumpNext  key.Binding
	Restart   key.Binding
	Stop      key.Binding
	Help      key.Binding
	Quit      key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("k"),
			key.WithHelp("k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j"),
			key.WithHelp("j", "down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "attach"),
		),
		New: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new workspace"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		PaneLeft: key.NewBinding(
			key.WithKeys("h"),
			key.WithHelp("h", "prev pane"),
		),
		PaneRight: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "next pane"),
		),
		Broadcast: key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "broadcast"),
		),
		Diff: key.NewBinding(
			key.WithKeys("D"),
			key.WithHelp("D", "diff"),
		),
		Rename: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "rename"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "cycle section"),
		),
		MarkRead: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "mark read"),
		),
		JumpNext: key.NewBinding(
			key.WithKeys("J"),
			key.WithHelp("J", "jump to next"),
		),
		Restart: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "restart"),
		),
		Stop: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "stop"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}
