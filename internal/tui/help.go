package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/ramtinj/colosseum/internal/tui/dialog"
	"github.com/ramtinj/colosseum/internal/tui/theme"
)

func renderShortHelp(t theme.Theme, keys KeyMap) string {
	items := []dialog.HelpItem{
		{Key: combineBindingLabels(keys.Down, keys.Up), Desc: "navigate"},
		{Key: combineBindingLabels(keys.PaneLeft, keys.PaneRight), Desc: "pane"},
		{Key: dialog.BindingLabel(keys.Enter), Desc: "attach"},
		{Key: dialog.BindingLabel(keys.New), Desc: "new"},
		{Key: dialog.BindingLabel(keys.Broadcast), Desc: "broadcast"},
		{Key: dialog.BindingLabel(keys.Help), Desc: "help"},
		{Key: dialog.BindingLabel(keys.Quit), Desc: "quit"},
	}

	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, t.HelpKey.Render(item.Key)+t.HelpDesc.Render(" "+item.Desc))
	}
	return strings.Join(parts, "  ")
}

func availableHelpItems(keys KeyMap, returnKey string) []dialog.HelpItem {
	return []dialog.HelpItem{
		{Key: combineBindingLabels(keys.Down, keys.Up), Desc: "Navigate workspace list"},
		{Key: combineBindingLabels(keys.PaneLeft, keys.PaneRight), Desc: "Switch preview pane tab"},
		{Key: dialog.BindingLabel(keys.Enter), Desc: "Attach to selected workspace"},
		{Key: dialog.BindingLabel(keys.New), Desc: "New workspace"},
		{Key: dialog.BindingLabel(keys.Delete), Desc: "Delete workspace"},
		{Key: dialog.BindingLabel(keys.Broadcast), Desc: "Broadcast prompt"},
		{Key: dialog.BindingLabel(keys.JumpNext), Desc: "Jump to next needing attention"},
		{Key: dialog.BindingLabel(keys.Help), Desc: "Toggle this help"},
		{Key: dialog.BindingLabel(keys.Quit), Desc: "Quit"},
		{Key: "", Desc: ""},
		{Key: "prefix+" + returnKey, Desc: "Return to dashboard from workspace"},
	}
}

func unavailableHelpItems(keys KeyMap) []dialog.HelpItem {
	return []dialog.HelpItem{
		{Key: dialog.BindingLabel(keys.Diff), Desc: "Diff viewer (unavailable)"},
		{Key: dialog.BindingLabel(keys.Rename), Desc: "Rename workspace (unavailable)"},
		{Key: dialog.BindingLabel(keys.Filter), Desc: "Filter workspaces (unavailable)"},
		{Key: dialog.BindingLabel(keys.MarkRead), Desc: "Mark read (unavailable)"},
		{Key: dialog.BindingLabel(keys.Restart), Desc: "Restart agent (unavailable)"},
		{Key: dialog.BindingLabel(keys.Stop), Desc: "Stop agent (unavailable)"},
	}
}

func combineBindingLabels(bindings ...key.Binding) string {
	labels := make([]string, 0, len(bindings))
	seen := make(map[string]struct{}, len(bindings))
	for _, binding := range bindings {
		label := dialog.BindingLabel(binding)
		if label == "" {
			continue
		}
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		labels = append(labels, label)
	}
	return strings.Join(labels, "/")
}
