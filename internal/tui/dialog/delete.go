package dialog

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ramtinj/colosseum/internal/tui/theme"
)

type DeleteConfirmMsg struct {
	WorkspaceID string
}

type DeleteCancelMsg struct{}

type DeleteModel struct {
	WorkspaceID    string
	WorkspaceTitle string
	confirmed      bool
}

func NewDelete(id, title string) DeleteModel {
	return DeleteModel{
		WorkspaceID:    id,
		WorkspaceTitle: title,
	}
}

func (m DeleteModel) Update(msg tea.Msg) (DeleteModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y", "enter":
			return m, func() tea.Msg {
				return DeleteConfirmMsg{WorkspaceID: m.WorkspaceID}
			}
		case "n", "N", "esc":
			return m, func() tea.Msg { return DeleteCancelMsg{} }
		}
	}
	return m, nil
}

func (m DeleteModel) View() string {
	t := theme.DefaultTheme()

	title := t.StatusError.Bold(true).Render(" Delete Workspace")
	prompt := fmt.Sprintf("\n  Delete %q?\n  This will kill the tmux session.\n", m.WorkspaceTitle)
	help := t.Dim.Render("  y/enter: confirm  n/esc: cancel")

	content := title + prompt + "\n" + help

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(1, 2).
		Width(45)

	return border.Render(content)
}
