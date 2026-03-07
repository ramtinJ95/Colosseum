package dialog

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ramtinj/colosseum/internal/tui/theme"
)

type HelpCloseMsg struct{}

type HelpModel struct {
	Width  int
	Height int
	theme  theme.Theme
}

func NewHelp() HelpModel {
	return HelpModel{theme: theme.DefaultTheme()}
}

func (m HelpModel) Update(msg tea.Msg) (HelpModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "?", "q":
			return m, func() tea.Msg { return HelpCloseMsg{} }
		}
	}
	return m, nil
}

func (m HelpModel) View() string {
	t := m.theme

	title := t.AppTitle.Render(" Keybindings")

	availableBindings := [][2]string{
		{"j/k", "Navigate workspace list"},
		{"h/l", "Switch preview pane tab"},
		{"enter", "Attach to selected workspace"},
		{"n", "New workspace"},
		{"d", "Delete workspace"},
		{"J", "Jump to next needing attention"},
		{"?", "Toggle this help"},
		{"q", "Quit"},
		{"", ""},
		{"prefix+e", "Return to dashboard from workspace"},
	}

	unavailableBindings := [][2]string{
		{"b", "Broadcast prompt (unavailable)"},
		{"D", "Diff viewer (unavailable)"},
		{"r", "Rename workspace (unavailable)"},
		{"/", "Filter workspaces (unavailable)"},
		{"m", "Mark read (unavailable)"},
		{"R", "Restart agent (unavailable)"},
		{"s", "Stop agent (unavailable)"},
	}

	var rows []string
	rows = append(rows, "  "+t.Dim.Render("Available"))
	for _, b := range availableBindings {
		key := t.HelpKey.Width(10).Render(b[0])
		desc := t.HelpDesc.Render(b[1])
		rows = append(rows, "  "+key+" "+desc)
	}

	rows = append(rows, "", "  "+t.Dim.Render("Unavailable"))
	for _, b := range unavailableBindings {
		key := t.HelpKey.Width(10).Render(b[0])
		desc := t.Dim.Render(b[1])
		rows = append(rows, "  "+key+" "+desc)
	}

	content := title + "\n\n" + strings.Join(rows, "\n") + "\n\n" + t.Dim.Render("  Press esc or ? to close")

	border := t.DialogBorder.
		Padding(1, 2).
		Width(56)

	return border.Render(content)
}

func (m HelpModel) WithTheme(t theme.Theme) HelpModel {
	m.theme = t
	return m
}
