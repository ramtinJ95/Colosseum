package sidebar

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/tui/theme"
	"github.com/ramtinj/colosseum/internal/workspace"
)

type Model struct {
	Workspaces []workspace.Workspace
	Cursor     int
	Width      int
	Height     int
	Focused    bool
	theme      theme.Theme
	upKey      key.Binding
	downKey    key.Binding
}

func New() Model {
	return Model{
		Focused: true,
		theme:   theme.DefaultTheme(),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.Focused {
			return m, nil
		}
		switch {
		case key.Matches(msg, m.upKey):
			if m.Cursor > 0 {
				m.Cursor--
			}
		case key.Matches(msg, m.downKey):
			if m.Cursor < len(m.Workspaces)-1 {
				m.Cursor++
			}
		}
	}
	return m, nil
}

func (m Model) WithNavigationKeys(up, down key.Binding) Model {
	m.upKey = up
	m.downKey = down
	return m
}

func (m Model) WithTheme(t theme.Theme) Model {
	m.theme = t
	return m
}

func (m Model) SelectedWorkspace() *workspace.Workspace {
	if len(m.Workspaces) == 0 || m.Cursor >= len(m.Workspaces) {
		return nil
	}
	return &m.Workspaces[m.Cursor]
}

func (m *Model) SetWorkspaces(ws []workspace.Workspace) {
	m.Workspaces = ws
	if m.Cursor >= len(ws) {
		m.Cursor = max(0, len(ws)-1)
	}
}

func (m *Model) UpdateWorkspaceStatus(id string, s agent.Status) {
	for i := range m.Workspaces {
		if m.Workspaces[i].ID == id {
			m.Workspaces[i].Status = s
			return
		}
	}
}
