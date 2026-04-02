package dialog

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ramtinj/colosseum/internal/config"
	"github.com/ramtinj/colosseum/internal/tui/theme"
)

type RenameConfirmMsg struct {
	WorkspaceID string
	NewTitle    string
}

type RenameCancelMsg struct{}

type RenameKeyMap struct {
	Confirm key.Binding
	Cancel  key.Binding
}

func RenameKeyMapFromConfig(keys config.KeysConfig) RenameKeyMap {
	return RenameKeyMap{
		Confirm: key.NewBinding(
			key.WithKeys(keys.Enter),
			key.WithHelp(keys.Enter, "confirm"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
	}
}

type RenameModel struct {
	WorkspaceID string
	input       textinput.Model
	keys        RenameKeyMap
	theme       theme.Theme
}

func NewRename(id, currentTitle string) RenameModel {
	ti := textinput.New()
	ti.Placeholder = "new workspace title"
	ti.SetValue(currentTitle)
	ti.Focus()
	ti.CharLimit = 64

	return RenameModel{
		WorkspaceID: id,
		input:       ti,
		keys:        RenameKeyMapFromConfig(config.Default().Keys),
		theme:       theme.DefaultTheme(),
	}
}

func (m RenameModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m RenameModel) Update(msg tea.Msg) (RenameModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Confirm):
			return m, func() tea.Msg {
				return RenameConfirmMsg{
					WorkspaceID: m.WorkspaceID,
					NewTitle:    m.input.Value(),
				}
			}
		case key.Matches(msg, m.keys.Cancel):
			return m, func() tea.Msg { return RenameCancelMsg{} }
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m RenameModel) View() string {
	t := m.theme

	title := t.StatusWorking.Bold(true).Render(" Rename Workspace")
	prompt := "\n  Enter a new title:\n\n"
	inputView := "  " + m.input.View() + "\n"
	help := t.Dim.Render(fmt.Sprintf("  %s: confirm  %s: cancel", BindingLabel(m.keys.Confirm), BindingLabel(m.keys.Cancel)))

	content := title + prompt + inputView + "\n" + help

	border := t.DialogBorder.
		Padding(1, 2).
		Width(50)

	return border.Render(content)
}

func (m RenameModel) WithTheme(t theme.Theme) RenameModel {
	m.theme = t
	return m
}

func (m RenameModel) WithKeyMap(keys RenameKeyMap) RenameModel {
	m.keys = keys
	return m
}
