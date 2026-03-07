package dialog

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/ramtinj/colosseum/internal/config"
)

type DeleteKeyMap struct {
	Confirm key.Binding
	Cancel  key.Binding
}

func DeleteKeyMapFromConfig(keys config.KeysConfig) DeleteKeyMap {
	return DeleteKeyMap{
		Confirm: key.NewBinding(
			key.WithKeys("y", "Y", keys.Enter),
			key.WithHelp(joinKeyLabels("y", keys.Enter), "confirm"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("n", "N", "esc"),
			key.WithHelp("n/esc", "cancel"),
		),
	}
}

type NewWorkspaceKeyMap struct {
	Up         key.Binding
	Down       key.Binding
	Enter      key.Binding
	Tab        key.Binding
	BackTab    key.Binding
	Cancel     key.Binding
	SelectPrev key.Binding
	SelectNext key.Binding
}

func NewWorkspaceKeyMapFromConfig(keys config.KeysConfig) NewWorkspaceKeyMap {
	return NewWorkspaceKeyMap{
		Up:         newBinding(keys.Up, "move up"),
		Down:       newBinding(keys.Down, "move down"),
		Enter:      newBinding(keys.Enter, "next/create"),
		Tab:        newBinding(keys.Tab, "next/complete"),
		BackTab:    newBinding("shift+tab", "previous"),
		Cancel:     newBinding("esc", "cancel"),
		SelectPrev: newBinding(keys.PaneLeft, "previous option"),
		SelectNext: newBinding(keys.PaneRight, "next option"),
	}
}

type BroadcastKeyMap struct {
	Up           key.Binding
	Down         key.Binding
	Tab          key.Binding
	BackTab      key.Binding
	Enter        key.Binding
	Submit       key.Binding
	ToggleTarget key.Binding
	ToggleAll    key.Binding
	Cancel       key.Binding
}

func BroadcastKeyMapFromConfig(keys config.KeysConfig) BroadcastKeyMap {
	return BroadcastKeyMap{
		Up:           newBinding(keys.Up, "move up"),
		Down:         newBinding(keys.Down, "move down"),
		Tab:          newBinding(keys.Tab, "switch focus"),
		BackTab:      newBinding("shift+tab", "switch focus"),
		Enter:        newBinding(keys.Enter, "toggle target"),
		Submit:       newBinding("ctrl+s", "send"),
		ToggleTarget: key.NewBinding(key.WithKeys(" ", "x"), key.WithHelp("space/x", "toggle target")),
		ToggleAll:    newBinding("a", "toggle all"),
		Cancel:       newBinding("esc", "cancel"),
	}
}

func newBinding(keyName, desc string) key.Binding {
	return key.NewBinding(
		key.WithKeys(keyName),
		key.WithHelp(keyName, desc),
	)
}

func bindingLabel(binding key.Binding) string {
	keys := binding.Keys()
	if len(keys) > 0 {
		return strings.Join(keys, "/")
	}
	return binding.Help().Key
}

func joinKeyLabels(labels ...string) string {
	filtered := make([]string, 0, len(labels))
	seen := make(map[string]struct{}, len(labels))
	for _, label := range labels {
		label = strings.TrimSpace(label)
		if label == "" {
			continue
		}
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		filtered = append(filtered, label)
	}
	return strings.Join(filtered, "/")
}
