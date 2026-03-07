package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/ramtinj/colosseum/internal/config"
)

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
	return KeyMapFromConfig(config.Default().Keys)
}

func KeyMapFromConfig(kc config.KeysConfig) KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys(kc.Up),
			key.WithHelp(kc.Up, "up"),
		),
		Down: key.NewBinding(
			key.WithKeys(kc.Down),
			key.WithHelp(kc.Down, "down"),
		),
		Enter: key.NewBinding(
			key.WithKeys(kc.Enter),
			key.WithHelp(kc.Enter, "attach"),
		),
		New: key.NewBinding(
			key.WithKeys(kc.New),
			key.WithHelp(kc.New, "new workspace"),
		),
		Delete: key.NewBinding(
			key.WithKeys(kc.Delete),
			key.WithHelp(kc.Delete, "delete"),
		),
		PaneLeft: key.NewBinding(
			key.WithKeys(kc.PaneLeft),
			key.WithHelp(kc.PaneLeft, "prev pane"),
		),
		PaneRight: key.NewBinding(
			key.WithKeys(kc.PaneRight),
			key.WithHelp(kc.PaneRight, "next pane"),
		),
		Broadcast: key.NewBinding(
			key.WithKeys(kc.Broadcast),
			key.WithHelp(kc.Broadcast, "broadcast"),
		),
		Diff: key.NewBinding(
			key.WithKeys(kc.Diff),
			key.WithHelp(kc.Diff, "diff"),
		),
		Rename: key.NewBinding(
			key.WithKeys(kc.Rename),
			key.WithHelp(kc.Rename, "rename"),
		),
		Filter: key.NewBinding(
			key.WithKeys(kc.Filter),
			key.WithHelp(kc.Filter, "filter"),
		),
		Tab: key.NewBinding(
			key.WithKeys(kc.Tab),
			key.WithHelp(kc.Tab, "cycle section"),
		),
		MarkRead: key.NewBinding(
			key.WithKeys(kc.MarkRead),
			key.WithHelp(kc.MarkRead, "mark read"),
		),
		JumpNext: key.NewBinding(
			key.WithKeys(kc.JumpNext),
			key.WithHelp(kc.JumpNext, "jump to next"),
		),
		Restart: key.NewBinding(
			key.WithKeys(kc.Restart),
			key.WithHelp(kc.Restart, "restart"),
		),
		Stop: key.NewBinding(
			key.WithKeys(kc.Stop),
			key.WithHelp(kc.Stop, "stop"),
		),
		Help: key.NewBinding(
			key.WithKeys(kc.Help),
			key.WithHelp(kc.Help, "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys(kc.Quit, "ctrl+c"),
			key.WithHelp(kc.Quit, "quit"),
		),
	}
}
