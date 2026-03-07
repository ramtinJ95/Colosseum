package dialog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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
	pathCycle   pathCycleState
	agents      []agent.AgentType
	layouts     []workspace.LayoutType
	Width       int
	Height      int
	theme       theme.Theme
}

type pathCycleState struct {
	baseValue string
	matches   []string
	index     int
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
	pathInput.ShowSuggestions = true

	branchInput := textinput.New()
	branchInput.Placeholder = "branch (optional)"
	branchInput.CharLimit = 100
	branchInput.Width = 30

	return NewWorkspaceModel{
		inputs:  [fieldCount]textinput.Model{nameInput, pathInput, branchInput},
		agents:  agent.Supported(),
		layouts: workspace.ValidLayouts(),
		theme:   theme.DefaultTheme(),
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

		case "tab":
			if m.focusIndex == int(fieldPath) {
				m.completePathShellStyle()
				return m, nil
			}
			m.focusIndex = (m.focusIndex + 1) % totalFields
			m.updateFocus()
			return m, nil

		case "shift+tab":
			m.focusIndex = (m.focusIndex - 1 + totalFields) % totalFields
			m.updateFocus()
			return m, nil

		case "enter":
			if m.focusIndex < totalFields-1 {
				m.focusIndex++
				m.updateFocus()
				return m, nil
			}
			name := strings.TrimSpace(m.inputs[fieldName].Value())
			if name == "" {
				return m, nil
			}
			path := strings.TrimSpace(m.inputs[fieldPath].Value())
			if path == "" {
				path = "."
			}
			path = expandPathValue(path)
			return m, func() tea.Msg {
				return NewWorkspaceMsg{
					Name:      name,
					Path:      path,
					AgentType: m.agents[m.agentIndex],
					Branch:    strings.TrimSpace(m.inputs[fieldBranch].Value()),
					Layout:    m.layouts[m.layoutIndex],
				}
			}

		case "h":
			if m.focusIndex == selectorAgent {
				m.agentIndex = (m.agentIndex - 1 + len(m.agents)) % len(m.agents)
				return m, nil
			} else if m.focusIndex == selectorLayout {
				m.layoutIndex = (m.layoutIndex - 1 + len(m.layouts)) % len(m.layouts)
				return m, nil
			}

		case "l":
			if m.focusIndex == selectorAgent {
				m.agentIndex = (m.agentIndex + 1) % len(m.agents)
				return m, nil
			} else if m.focusIndex == selectorLayout {
				m.layoutIndex = (m.layoutIndex + 1) % len(m.layouts)
				return m, nil
			}
		}
	}

	if m.focusIndex < int(fieldCount) {
		var cmd tea.Cmd
		m.inputs[m.focusIndex], cmd = m.inputs[m.focusIndex].Update(msg)
		if m.focusIndex == int(fieldPath) {
			if _, ok := msg.(tea.KeyMsg); ok {
				m.resetPathCycle()
			}
			m.refreshPathSuggestions()
		}
		return m, cmd
	}

	return m, nil
}

func (m *NewWorkspaceModel) refreshPathSuggestions() {
	raw := m.inputs[fieldPath].Value()
	if raw == "" {
		m.inputs[fieldPath].SetSuggestions(nil)
		return
	}

	expanded := raw
	if strings.HasPrefix(expanded, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			expanded = filepath.Join(home, expanded[2:])
		}
	} else if expanded == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			expanded = home
		}
	}

	var dir, prefix string
	if strings.HasSuffix(expanded, "/") {
		dir = expanded
		prefix = ""
	} else {
		dir = filepath.Dir(expanded)
		prefix = filepath.Base(expanded)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		m.inputs[fieldPath].SetSuggestions(nil)
		return
	}

	var suggestions []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") && !strings.HasPrefix(prefix, ".") {
			continue
		}
		if prefix != "" && !strings.HasPrefix(name, prefix) {
			continue
		}
		full := filepath.Join(dir, name) + "/"
		if strings.HasPrefix(raw, "~/") || raw == "~" {
			if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(full, home) {
				full = "~" + full[len(home):]
			}
		}
		suggestions = append(suggestions, full)
	}

	m.inputs[fieldPath].SetSuggestions(suggestions)
}

func (m *NewWorkspaceModel) completePathShellStyle() {
	current := m.inputs[fieldPath].Value()
	if m.pathCycle.canContinue(current) {
		m.pathCycle.index = (m.pathCycle.index + 1) % len(m.pathCycle.matches)
		m.setPathValue(m.pathCycle.matches[m.pathCycle.index])
		return
	}

	m.refreshPathSuggestions()
	matches := append([]string(nil), m.inputs[fieldPath].MatchedSuggestions()...)
	if len(matches) == 0 {
		m.resetPathCycle()
		return
	}

	commonPrefix := longestSharedPrefix(matches)
	if len(commonPrefix) > len(current) {
		m.setPathValue(commonPrefix)
		m.resetPathCycle()
		return
	}

	if len(matches) == 1 {
		m.setPathValue(matches[0])
		m.resetPathCycle()
		return
	}

	m.pathCycle = pathCycleState{
		baseValue: current,
		matches:   matches,
		index:     0,
	}
	m.setPathValue(matches[0])
}

func (m *NewWorkspaceModel) setPathValue(value string) {
	m.inputs[fieldPath].SetValue(value)
	m.inputs[fieldPath].CursorEnd()
	m.refreshPathSuggestions()
}

func (m *NewWorkspaceModel) resetPathCycle() {
	m.pathCycle = pathCycleState{}
}

func (s pathCycleState) canContinue(current string) bool {
	if len(s.matches) < 2 {
		return false
	}
	if current == s.baseValue {
		return true
	}
	for _, match := range s.matches {
		if current == match {
			return true
		}
	}
	return false
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
	t := m.theme

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
		if i == int(fieldPath) {
			if hint := renderPathSuggestionHint(t, m.inputs[fieldPath]); hint != "" {
				rows = append(rows, hint)
			}
		}
	}

	agentLabel := t.Dim.Render(" Agent:")
	if m.focusIndex == selectorAgent {
		agentLabel = t.StatusWaiting.Render(" Agent:")
	}
	rows = append(rows, fmt.Sprintf("%s %s", agentLabel, renderChoices(t, agentStrings(m.agents), m.agentIndex, m.focusIndex == selectorAgent)))

	layoutLabel := t.Dim.Render("Layout:")
	if m.focusIndex == selectorLayout {
		layoutLabel = t.StatusWaiting.Render("Layout:")
	}
	rows = append(rows, fmt.Sprintf("%s %s", layoutLabel, renderChoices(t, layoutStrings(m.layouts), m.layoutIndex, m.focusIndex == selectorLayout)))

	help := t.Dim.Render("  tab: complete path  enter: next/create  up/down: cycle matches  h/l: select  esc: cancel")

	content := title + "\n\n" + strings.Join(rows, "\n") + "\n\n" + help

	border := t.DialogBorder.
		Padding(1, 2).
		Width(50)

	return border.Render(content)
}

func renderChoices(t theme.Theme, items []string, selected int, focused bool) string {
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

func (m NewWorkspaceModel) WithTheme(t theme.Theme) NewWorkspaceModel {
	m.theme = t
	return m
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

func renderPathSuggestionHint(t theme.Theme, input textinput.Model) string {
	matches := input.MatchedSuggestions()
	if len(matches) == 0 {
		return ""
	}

	current := input.CurrentSuggestion()
	if current == "" {
		return ""
	}

	hint := fmt.Sprintf("        %s %s", t.StatusWaiting.Render("suggestion:"), t.Dim.Render(current))
	if len(matches) > 1 {
		hint += t.Dim.Render(fmt.Sprintf("  (%d matches)", len(matches)))
	}
	return hint
}

func longestSharedPrefix(values []string) string {
	if len(values) == 0 {
		return ""
	}

	prefix := []rune(values[0])
	for _, value := range values[1:] {
		runes := []rune(value)
		limit := min(len(prefix), len(runes))
		i := 0
		for i < limit && prefix[i] == runes[i] {
			i++
		}
		prefix = prefix[:i]
		if len(prefix) == 0 {
			return ""
		}
	}
	return string(prefix)
}

func expandPathValue(value string) string {
	if value == "~" || strings.HasPrefix(value, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			if value == "~" {
				return home
			}
			return filepath.Join(home, value[2:])
		}
	}
	return value
}
