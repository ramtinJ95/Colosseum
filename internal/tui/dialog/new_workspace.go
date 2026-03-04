package dialog

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/tui/theme"
	"github.com/ramtinj/colosseum/internal/workspace"
)

type NewWorkspaceMsg struct {
	Name      string
	Path      string
	AgentType agent.AgentType
	Branch    string
	Layout    workspace.LayoutType
}

type NewWorkspaceCancelMsg struct{}

type field int

const (
	fieldName field = iota
	fieldPath
	fieldBranch
	fieldCount
)

const (
	selectorAgent  = int(fieldCount)
	selectorLayout = int(fieldCount) + 1
	totalFields    = int(fieldCount) + 2
)

type NewWorkspaceModel struct {
	inputs      [fieldCount]textinput.Model
	focusIndex  int
	agentIndex  int
	layoutIndex int
	agents      []agent.AgentType
	layouts     []workspace.LayoutType
	Width       int
	Height      int
}

func NewNewWorkspace() NewWorkspaceModel {
	nameInput := textinput.New()
	nameInput.Placeholder = "workspace name"
	nameInput.Focus()
	nameInput.CharLimit = 40
	nameInput.Width = 30

	pathInput := textinput.New()
	pathInput.Placeholder = "project path (default: .)"
	pathInput.CharLimit = 200
	pathInput.Width = 30

	branchInput := textinput.New()
	branchInput.Placeholder = "branch (optional)"
	branchInput.CharLimit = 100
	branchInput.Width = 30

	return NewWorkspaceModel{
		inputs:  [fieldCount]textinput.Model{nameInput, pathInput, branchInput},
		agents:  agent.Available(),
		layouts: workspace.ValidLayouts(),
	}
}

func (m NewWorkspaceModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m NewWorkspaceModel) Update(msg tea.Msg) (NewWorkspaceModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return NewWorkspaceCancelMsg{} }

		case "tab", "j":
			m.focusIndex = (m.focusIndex + 1) % totalFields
			m.updateFocus()
			return m, nil

		case "shift+tab", "k":
			m.focusIndex = (m.focusIndex - 1 + totalFields) % totalFields
			m.updateFocus()
			return m, nil

		case "h":
			if m.focusIndex == selectorAgent {
				m.agentIndex = (m.agentIndex - 1 + len(m.agents)) % len(m.agents)
			} else if m.focusIndex == selectorLayout {
				m.layoutIndex = (m.layoutIndex - 1 + len(m.layouts)) % len(m.layouts)
			}
			return m, nil

		case "l":
			if m.focusIndex == selectorAgent {
				m.agentIndex = (m.agentIndex + 1) % len(m.agents)
			} else if m.focusIndex == selectorLayout {
				m.layoutIndex = (m.layoutIndex + 1) % len(m.layouts)
			}
			return m, nil

		case "enter":
			name := strings.TrimSpace(m.inputs[fieldName].Value())
			if name == "" {
				return m, nil
			}
			path := strings.TrimSpace(m.inputs[fieldPath].Value())
			if path == "" {
				path = "."
			}
			return m, func() tea.Msg {
				return NewWorkspaceMsg{
					Name:      name,
					Path:      path,
					AgentType: m.agents[m.agentIndex],
					Branch:    strings.TrimSpace(m.inputs[fieldBranch].Value()),
					Layout:    m.layouts[m.layoutIndex],
				}
			}
		}
	}

	if m.focusIndex < int(fieldCount) {
		var cmd tea.Cmd
		m.inputs[m.focusIndex], cmd = m.inputs[m.focusIndex].Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *NewWorkspaceModel) updateFocus() {
	for i := range m.inputs {
		if i == m.focusIndex {
			m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
}

func (m NewWorkspaceModel) View() string {
	t := theme.DefaultTheme()

	title := t.AppTitle.Render(" New Workspace")

	labels := [fieldCount]string{"  Name:", "  Path:", "Branch:"}
	var rows []string
	for i := range m.inputs {
		label := labels[i]
		if i == m.focusIndex {
			label = t.StatusWaiting.Render(label)
		} else {
			label = t.Dim.Render(label)
		}
		rows = append(rows, fmt.Sprintf("%s %s", label, m.inputs[i].View()))
	}

	agentLabel := t.Dim.Render(" Agent:")
	if m.focusIndex == selectorAgent {
		agentLabel = t.StatusWaiting.Render(" Agent:")
	}
	rows = append(rows, fmt.Sprintf("%s %s", agentLabel, renderChoices(agentStrings(m.agents), m.agentIndex, m.focusIndex == selectorAgent)))

	layoutLabel := t.Dim.Render("Layout:")
	if m.focusIndex == selectorLayout {
		layoutLabel = t.StatusWaiting.Render("Layout:")
	}
	rows = append(rows, fmt.Sprintf("%s %s", layoutLabel, renderChoices(layoutStrings(m.layouts), m.layoutIndex, m.focusIndex == selectorLayout)))

	help := t.Dim.Render("  j/k: navigate  h/l: select  enter: create  esc: cancel")

	content := title + "\n\n" + strings.Join(rows, "\n") + "\n\n" + help

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("220")).
		Padding(1, 2).
		Width(50)

	return border.Render(content)
}

func renderChoices(items []string, selected int, focused bool) string {
	t := theme.DefaultTheme()
	var parts []string
	for i, item := range items {
		if i == selected {
			if focused {
				item = t.StatusWaiting.Bold(true).Render(fmt.Sprintf("[%s]", item))
			} else {
				item = t.NormalItem.Bold(true).Render(fmt.Sprintf("[%s]", item))
			}
		} else {
			item = t.Dim.Render(item)
		}
		parts = append(parts, item)
	}
	return strings.Join(parts, " ")
}

func agentStrings(agents []agent.AgentType) []string {
	s := make([]string, len(agents))
	for i, a := range agents {
		s[i] = string(a)
	}
	return s
}

func layoutStrings(layouts []workspace.LayoutType) []string {
	s := make([]string, len(layouts))
	for i, l := range layouts {
		s[i] = string(l)
	}
	return s
}
