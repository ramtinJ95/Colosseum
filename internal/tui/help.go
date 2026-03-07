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
		{Key: bindingLabel(keys.Enter), Desc: "attach"},
		{Key: bindingLabel(keys.New), Desc: "new"},
		{Key: bindingLabel(keys.Broadcast), Desc: "broadcast"},
		{Key: bindingLabel(keys.Help), Desc: "help"},
		{Key: bindingLabel(keys.Quit), Desc: "quit"},
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
		{Key: bindingLabel(keys.Enter), Desc: "Attach to selected workspace"},
		{Key: bindingLabel(keys.New), Desc: "New workspace"},
		{Key: bindingLabel(keys.Delete), Desc: "Delete workspace"},
		{Key: bindingLabel(keys.Broadcast), Desc: "Broadcast prompt"},
		{Key: bindingLabel(keys.JumpNext), Desc: "Jump to next needing attention"},
		{Key: bindingLabel(keys.Help), Desc: "Toggle this help"},
		{Key: bindingLabel(keys.Quit), Desc: "Quit"},
		{Key: "", Desc: ""},
		{Key: "prefix+" + returnKey, Desc: "Return to dashboard from workspace"},
	}
}

func unavailableHelpItems(keys KeyMap) []dialog.HelpItem {
	return []dialog.HelpItem{
		{Key: bindingLabel(keys.Diff), Desc: "Diff viewer (unavailable)"},
		{Key: bindingLabel(keys.Rename), Desc: "Rename workspace (unavailable)"},
		{Key: bindingLabel(keys.Filter), Desc: "Filter workspaces (unavailable)"},
		{Key: bindingLabel(keys.MarkRead), Desc: "Mark read (unavailable)"},
		{Key: bindingLabel(keys.Restart), Desc: "Restart agent (unavailable)"},
		{Key: bindingLabel(keys.Stop), Desc: "Stop agent (unavailable)"},
	}
}

func bindingLabel(binding key.Binding) string {
	if label := binding.Help().Key; label != "" {
		return label
	}
	keys := binding.Keys()
	return strings.Join(keys, "/")
}

func combineBindingLabels(bindings ...key.Binding) string {
	labels := make([]string, 0, len(bindings))
	seen := make(map[string]struct{}, len(bindings))
	for _, binding := range bindings {
		label := bindingLabel(binding)
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
