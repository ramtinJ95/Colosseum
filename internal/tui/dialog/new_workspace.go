package dialog

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/config"
	"github.com/ramtinj/colosseum/internal/tui/theme"
	"github.com/ramtinj/colosseum/internal/workspace"
)

type NewWorkspaceMsg struct {
	Name           string
	Path           string
	AgentType      agent.AgentType
	Branch         string
	BaseBranch     string
	Prompt         string
	CandidateCount int
	Layout         workspace.LayoutType
	Mode           workspace.CreateMode
	AgentStrategy  workspace.ExperimentAgentStrategy
}

type NewWorkspaceCancelMsg struct{}

type newWorkspaceField int

const (
	fieldName newWorkspaceField = iota
	fieldPath
	fieldBranch
	fieldBaseBranch
	fieldPrompt
	fieldCount
	fieldTotal
)

const (
	selectorMode     = int(fieldTotal)
	selectorStrategy = int(fieldTotal) + 1
	selectorAgent    = int(fieldTotal) + 2
	selectorLayout   = int(fieldTotal) + 3
	totalFields      = int(fieldTotal) + 4
)

type NewWorkspaceModel struct {
	inputs        [fieldTotal]textinput.Model
	focusIndex    int
	modeIndex     int
	strategyIndex int
	agentIndex    int
	layoutIndex   int
	pathCycle     pathCycleState
	modes         []workspace.CreateMode
	strategies    []workspace.ExperimentAgentStrategy
	agents        []agent.AgentType
	layouts       []workspace.LayoutType
	Width         int
	Height        int
	keys          NewWorkspaceKeyMap
	theme         theme.Theme
}

type pathCycleState struct {
	baseValue string
	matches   []string
	index     int
}

func NewNewWorkspace() NewWorkspaceModel {
	nameInput := textinput.New()
	nameInput.Placeholder = "workspace or experiment title"
	nameInput.Focus()
	nameInput.CharLimit = 60
	nameInput.Width = 36

	pathInput := textinput.New()
	pathInput.Placeholder = "checkout path or repo root"
	pathInput.CharLimit = 200
	pathInput.Width = 36
	pathInput.ShowSuggestions = true

	branchInput := textinput.New()
	branchInput.Placeholder = "branch (blank = generated for new worktree)"
	branchInput.CharLimit = 100
	branchInput.Width = 36

	baseBranchInput := textinput.New()
	baseBranchInput.Placeholder = "base branch (optional)"
	baseBranchInput.CharLimit = 100
	baseBranchInput.Width = 36

	promptInput := textinput.New()
	promptInput.Placeholder = "broadcast prompt for experiment (optional)"
	promptInput.CharLimit = 400
	promptInput.Width = 36

	countInput := textinput.New()
	countInput.Placeholder = "candidate count"
	countInput.CharLimit = 3
	countInput.Width = 8
	countInput.SetValue("2")

	return NewWorkspaceModel{
		inputs: [fieldTotal]textinput.Model{
			nameInput,
			pathInput,
			branchInput,
			baseBranchInput,
			promptInput,
			countInput,
		},
		modes:      []workspace.CreateMode{workspace.CreateModeExistingCheckout, workspace.CreateModeNewWorktree, workspace.CreateModeExperimentRun},
		strategies: []workspace.ExperimentAgentStrategy{workspace.ExperimentAgentSelected, workspace.ExperimentAgentAllSupported},
		agents:     agent.Supported(),
		layouts:    workspace.ValidLayouts(),
		keys:       NewWorkspaceKeyMapFromConfig(config.Default().Keys),
		theme:      theme.DefaultTheme(),
	}
}

func (m NewWorkspaceModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m NewWorkspaceModel) Update(msg tea.Msg) (NewWorkspaceModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Cancel):
			return m, func() tea.Msg { return NewWorkspaceCancelMsg{} }

		case key.Matches(msg, m.keys.Tab):
			if m.focusIndex == int(fieldPath) {
				m.completePathShellStyle()
				return m, nil
			}
			m.focusIndex = (m.focusIndex + 1) % totalFields
			m.updateFocus()
			return m, nil

		case key.Matches(msg, m.keys.BackTab):
			m.focusIndex = (m.focusIndex - 1 + totalFields) % totalFields
			m.updateFocus()
			return m, nil

		case key.Matches(msg, m.keys.Enter):
			if m.focusIndex < totalFields-1 {
				m.focusIndex++
				m.updateFocus()
				return m, nil
			}
			msg, ok := m.buildSubmitMessage()
			if !ok {
				return m, nil
			}
			return m, func() tea.Msg { return msg }

		case key.Matches(msg, m.keys.SelectPrev):
			switch m.focusIndex {
			case selectorMode:
				m.modeIndex = (m.modeIndex - 1 + len(m.modes)) % len(m.modes)
				return m, nil
			case selectorStrategy:
				m.strategyIndex = (m.strategyIndex - 1 + len(m.strategies)) % len(m.strategies)
				return m, nil
			case selectorAgent:
				m.agentIndex = (m.agentIndex - 1 + len(m.agents)) % len(m.agents)
				return m, nil
			case selectorLayout:
				m.layoutIndex = (m.layoutIndex - 1 + len(m.layouts)) % len(m.layouts)
				return m, nil
			}

		case key.Matches(msg, m.keys.SelectNext):
			switch m.focusIndex {
			case selectorMode:
				m.modeIndex = (m.modeIndex + 1) % len(m.modes)
				return m, nil
			case selectorStrategy:
				m.strategyIndex = (m.strategyIndex + 1) % len(m.strategies)
				return m, nil
			case selectorAgent:
				m.agentIndex = (m.agentIndex + 1) % len(m.agents)
				return m, nil
			case selectorLayout:
				m.layoutIndex = (m.layoutIndex + 1) % len(m.layouts)
				return m, nil
			}
		}
	}

	if m.focusIndex < int(fieldTotal) {
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

func (m NewWorkspaceModel) buildSubmitMessage() (NewWorkspaceMsg, bool) {
	name := strings.TrimSpace(m.inputs[fieldName].Value())
	if name == "" {
		return NewWorkspaceMsg{}, false
	}
	path := strings.TrimSpace(m.inputs[fieldPath].Value())
	if path == "" {
		path = "."
	}
	path = expandPathValue(path)

	candidateCount := 2
	if raw := strings.TrimSpace(m.inputs[fieldCount].Value()); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			return NewWorkspaceMsg{}, false
		}
		candidateCount = value
	}

	return NewWorkspaceMsg{
		Name:           name,
		Path:           path,
		AgentType:      m.agents[m.agentIndex],
		Branch:         strings.TrimSpace(m.inputs[fieldBranch].Value()),
		BaseBranch:     strings.TrimSpace(m.inputs[fieldBaseBranch].Value()),
		Prompt:         strings.TrimSpace(m.inputs[fieldPrompt].Value()),
		CandidateCount: candidateCount,
		Layout:         m.layouts[m.layoutIndex],
		Mode:           m.modes[m.modeIndex],
		AgentStrategy:  m.strategies[m.strategyIndex],
	}, true
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
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
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

	m.pathCycle = pathCycleState{baseValue: current, matches: matches, index: 0}
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
	labels := []string{"  Name:", "  Path:", "Branch:", "  Base:", "Prompt:", " Count:"}

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

	rows = append(rows, fmt.Sprintf("%s %s", focusLabel(t, "  Mode:", m.focusIndex == selectorMode), renderChoices(t, modeStrings(m.modes), m.modeIndex, m.focusIndex == selectorMode)))
	rows = append(rows, fmt.Sprintf("%s %s", focusLabel(t, "Strategy:", m.focusIndex == selectorStrategy), renderChoices(t, strategyStrings(m.strategies), m.strategyIndex, m.focusIndex == selectorStrategy)))
	rows = append(rows, fmt.Sprintf("%s %s", focusLabel(t, " Agent:", m.focusIndex == selectorAgent), renderChoices(t, agentStrings(m.agents), m.agentIndex, m.focusIndex == selectorAgent)))
	rows = append(rows, fmt.Sprintf("%s %s", focusLabel(t, "Layout:", m.focusIndex == selectorLayout), renderChoices(t, layoutStrings(m.layouts), m.layoutIndex, m.focusIndex == selectorLayout)))

	help := t.Dim.Render(fmt.Sprintf(
		"  %s: complete path  %s: next/create  %s/%s: select  %s: cancel",
		BindingLabel(m.keys.Tab),
		BindingLabel(m.keys.Enter),
		BindingLabel(m.keys.SelectPrev),
		BindingLabel(m.keys.SelectNext),
		BindingLabel(m.keys.Cancel),
	))

	content := title + "\n\n" + strings.Join(rows, "\n") + "\n\n" + help
	return t.DialogBorder.Padding(1, 2).Width(68).Render(content)
}

func focusLabel(t theme.Theme, label string, focused bool) string {
	if focused {
		return t.StatusWaiting.Render(label)
	}
	return t.Dim.Render(label)
}

func renderChoices(t theme.Theme, items []string, selected int, focused bool) string {
	var parts []string
	for i, item := range items {
		switch {
		case i == selected && focused:
			item = t.StatusWaiting.Bold(true).Render(fmt.Sprintf("[%s]", item))
		case i == selected:
			item = t.NormalItem.Bold(true).Render(fmt.Sprintf("[%s]", item))
		default:
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

func (m NewWorkspaceModel) WithKeyMap(keys NewWorkspaceKeyMap) NewWorkspaceModel {
	m.keys = keys
	return m
}

func agentStrings(agents []agent.AgentType) []string {
	values := make([]string, len(agents))
	for i, agentType := range agents {
		values[i] = string(agentType)
	}
	return values
}

func layoutStrings(layouts []workspace.LayoutType) []string {
	values := make([]string, len(layouts))
	for i, layout := range layouts {
		values[i] = string(layout)
	}
	return values
}

func modeStrings(modes []workspace.CreateMode) []string {
	values := make([]string, len(modes))
	for i, mode := range modes {
		values[i] = string(mode)
	}
	return values
}

func strategyStrings(strategies []workspace.ExperimentAgentStrategy) []string {
	values := make([]string, len(strategies))
	for i, strategy := range strategies {
		values[i] = string(strategy)
	}
	return values
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
		index := 0
		for index < limit && prefix[index] == runes[index] {
			index++
		}
		prefix = prefix[:index]
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
