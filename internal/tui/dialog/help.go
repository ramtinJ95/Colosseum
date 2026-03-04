package dialog

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ramtinj/colosseum/internal/tui/theme"
)

type HelpCloseMsg struct{}

type HelpModel struct {
	Width  int
	Height int
}

func NewHelp() HelpModel {
	return HelpModel{}
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
	t := theme.DefaultTheme()

	title := t.AppTitle.Render(" Keybindings")

	bindings := [][2]string{
		{"j/k", "Navigate workspace list"},
		{"enter", "Attach to selected workspace"},
		{"n", "New workspace"},
		{"d", "Delete workspace"},
		{"J", "Jump to next needing attention"},
		{"b", "Broadcast prompt"},
		{"D", "Diff viewer"},
		{"r", "Rename workspace"},
		{"/", "Filter workspaces"},
		{"R", "Restart agent"},
		{"s", "Stop agent"},
		{"?", "Toggle this help"},
		{"q", "Quit"},
		{"", ""},
		{"prefix+L", "Return to dashboard from workspace"},
	}

	var rows []string
	for _, b := range bindings {
		key := t.HelpKey.Width(10).Render(b[0])
		desc := t.HelpDesc.Render(b[1])
		rows = append(rows, "  "+key+" "+desc)
	}

	content := title + "\n\n" + strings.Join(rows, "\n") + "\n\n" + t.Dim.Render("  Press esc or ? to close")

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(50)

	return border.Render(content)
}
