package dialog

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ramtinj/colosseum/internal/tui/theme"
)

type HelpCloseMsg struct{}

type HelpItem struct {
	Key  string
	Desc string
}

type HelpModel struct {
	Width       int
	Height      int
	available   []HelpItem
	unavailable []HelpItem
	closeKeys   []key.Binding
	theme       theme.Theme
}

func NewHelp() HelpModel {
	return HelpModel{theme: theme.DefaultTheme()}
}

func (m HelpModel) Update(msg tea.Msg) (HelpModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.closeKeys...):
			return m, func() tea.Msg { return HelpCloseMsg{} }
		}
	}
	return m, nil
}

func (m HelpModel) View() string {
	t := m.theme

	title := t.AppTitle.Render(" Keybindings")

	var rows []string
	rows = append(rows, "  "+t.Dim.Render("Available"))
	for _, item := range m.available {
		key := t.HelpKey.Width(10).Render(item.Key)
		desc := t.HelpDesc.Render(item.Desc)
		rows = append(rows, "  "+key+" "+desc)
	}

	rows = append(rows, "", "  "+t.Dim.Render("Unavailable"))
	for _, item := range m.unavailable {
		key := t.HelpKey.Width(10).Render(item.Key)
		desc := t.Dim.Render(item.Desc)
		rows = append(rows, "  "+key+" "+desc)
	}

	content := title + "\n\n" + strings.Join(rows, "\n") + "\n\n" + t.Dim.Render("  Press "+joinCloseKeyLabels(m.closeKeys...)+" to close")

	border := t.DialogBorder.
		Padding(1, 2).
		Width(56)

	return border.Render(content)
}

func (m HelpModel) WithTheme(t theme.Theme) HelpModel {
	m.theme = t
	return m
}

func (m HelpModel) WithItems(available, unavailable []HelpItem, closeKeys ...key.Binding) HelpModel {
	m.available = available
	m.unavailable = unavailable
	m.closeKeys = closeKeys
	return m
}

func joinCloseKeyLabels(bindings ...key.Binding) string {
	labels := make([]string, 0, len(bindings))
	seen := make(map[string]struct{}, len(bindings))
	for _, binding := range bindings {
		label := binding.Help().Key
		if label == "" {
			keys := binding.Keys()
			label = strings.Join(keys, "/")
		}
		if label == "" {
			continue
		}
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		labels = append(labels, label)
	}
	return strings.Join(labels, " or ")
}
