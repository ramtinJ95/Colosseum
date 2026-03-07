package dialog

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ramtinj/colosseum/internal/config"
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
	keys           DeleteKeyMap
	theme          theme.Theme
}

func NewDelete(id, title string) DeleteModel {
	return DeleteModel{
		WorkspaceID:    id,
		WorkspaceTitle: title,
		keys:           DeleteKeyMapFromConfig(config.Default().Keys),
		theme:          theme.DefaultTheme(),
	}
}

func (m DeleteModel) Update(msg tea.Msg) (DeleteModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Confirm):
			return m, func() tea.Msg {
				return DeleteConfirmMsg{WorkspaceID: m.WorkspaceID}
			}
		case key.Matches(msg, m.keys.Cancel):
			return m, func() tea.Msg { return DeleteCancelMsg{} }
		}
	}
	return m, nil
}

func (m DeleteModel) View() string {
	t := m.theme

	title := t.StatusError.Bold(true).Render(" Delete Workspace")
	prompt := fmt.Sprintf("\n  Delete %q?\n  This will kill the tmux session.\n", m.WorkspaceTitle)
	help := t.Dim.Render(fmt.Sprintf("  %s: confirm  %s: cancel", bindingLabel(m.keys.Confirm), bindingLabel(m.keys.Cancel)))

	content := title + prompt + "\n" + help

	border := t.DialogBorder.
		Padding(1, 2).
		Width(45)

	return border.Render(content)
}

func (m DeleteModel) WithTheme(t theme.Theme) DeleteModel {
	m.theme = t
	return m
}

func (m DeleteModel) WithKeyMap(keys DeleteKeyMap) DeleteModel {
	m.keys = keys
	return m
}
