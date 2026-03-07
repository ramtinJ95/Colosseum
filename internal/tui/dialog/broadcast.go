package dialog

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ramtinj/colosseum/internal/config"
	"github.com/ramtinj/colosseum/internal/tui/theme"
	"github.com/ramtinj/colosseum/internal/workspace"
)

type BroadcastSubmitMsg struct {
	Prompt       string
	WorkspaceIDs []string
}

type BroadcastCancelMsg struct{}

type broadcastFocus int

const (
	broadcastFocusTargets broadcastFocus = iota
	broadcastFocusPrompt
)

type BroadcastModel struct {
	targets []broadcastTarget
	cursor  int
	focus   broadcastFocus
	prompt  textarea.Model
	keys    BroadcastKeyMap
	width   int
	height  int
	theme   theme.Theme
}

type broadcastTarget struct {
	ID        string
	Title     string
	AgentType string
	Selected  bool
}

func NewBroadcast(workspaces []workspace.Workspace, selectedID string) BroadcastModel {
	prompt := textarea.New()
	prompt.Placeholder = "Describe the task to send to the selected workspaces..."
	prompt.Prompt = ""
	prompt.ShowLineNumbers = false
	prompt.SetHeight(6)
	prompt.SetWidth(62)
	prompt.Blur()

	targets := make([]broadcastTarget, len(workspaces))
	hasSelected := false
	for i, ws := range workspaces {
		selected := ws.ID == selectedID
		if selected {
			hasSelected = true
		}
		targets[i] = broadcastTarget{
			ID:        ws.ID,
			Title:     ws.Title,
			AgentType: string(ws.AgentType),
			Selected:  selected,
		}
	}
	if !hasSelected && len(targets) > 0 {
		targets[0].Selected = true
	}

	return BroadcastModel{
		targets: targets,
		prompt:  prompt,
		keys:    DefaultBroadcastKeyMap(),
		theme:   theme.DefaultTheme(),
		width:   72,
		height:  20,
	}
}

func (m BroadcastModel) Init() tea.Cmd {
	return nil
}

func (m BroadcastModel) Update(msg tea.Msg) (BroadcastModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Cancel):
			return m, func() tea.Msg { return BroadcastCancelMsg{} }
		case key.Matches(msg, m.keys.Submit):
			prompt := strings.TrimSpace(m.prompt.Value())
			if prompt == "" || len(m.selectedWorkspaceIDs()) == 0 {
				return m, nil
			}
			return m, func() tea.Msg {
				return BroadcastSubmitMsg{
					Prompt:       m.prompt.Value(),
					WorkspaceIDs: m.selectedWorkspaceIDs(),
				}
			}
		}

		if key.Matches(msg, m.keys.Tab, m.keys.BackTab) {
			if m.focus == broadcastFocusTargets {
				m.focus = broadcastFocusPrompt
			} else {
				m.focus = broadcastFocusTargets
			}
			return m, m.applyFocus()
		}

		if m.focus == broadcastFocusTargets {
			switch {
			case key.Matches(msg, m.keys.Up):
				if m.cursor > 0 {
					m.cursor--
				}
				return m, nil
			case key.Matches(msg, m.keys.Down):
				if m.cursor < len(m.targets)-1 {
					m.cursor++
				}
				return m, nil
			case key.Matches(msg, m.keys.Enter, m.keys.ToggleTarget):
				if len(m.targets) > 0 {
					m.targets[m.cursor].Selected = !m.targets[m.cursor].Selected
				}
				return m, nil
			case key.Matches(msg, m.keys.ToggleAll):
				m.toggleAll()
				return m, nil
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.prompt, cmd = m.prompt.Update(msg)
	return m, cmd
}

func (m *BroadcastModel) SetSize(width, height int) {
	if width > 0 {
		m.width = width
	}
	if height > 0 {
		m.height = height
	}

	dialogWidth := min(max(m.width-8, 54), 88)
	promptWidth := max(dialogWidth-8, 20)
	promptHeight := min(max((m.height/4)+1, 5), 10)

	m.prompt.SetWidth(promptWidth)
	m.prompt.SetHeight(promptHeight)
}

func (m *BroadcastModel) applyFocus() tea.Cmd {
	if m.focus == broadcastFocusPrompt {
		return m.prompt.Focus()
	}
	m.prompt.Blur()
	return nil
}

func (m *BroadcastModel) toggleAll() {
	if len(m.targets) == 0 {
		return
	}

	selectAll := false
	for _, target := range m.targets {
		if !target.Selected {
			selectAll = true
			break
		}
	}

	for i := range m.targets {
		m.targets[i].Selected = selectAll
	}
}

func (m BroadcastModel) selectedWorkspaceIDs() []string {
	ids := make([]string, 0, len(m.targets))
	for _, target := range m.targets {
		if target.Selected {
			ids = append(ids, target.ID)
		}
	}
	return ids
}

func (m BroadcastModel) View() string {
	t := m.theme

	title := t.AppTitle.Render(" Broadcast Prompt")
	targetLabel := t.Dim.Render(" Targets")
	promptLabel := t.Dim.Render(" Prompt")
	if m.focus == broadcastFocusTargets {
		targetLabel = t.StatusWaiting.Render(" Targets")
	} else {
		promptLabel = t.StatusWaiting.Render(" Prompt")
	}

	var targetRows []string
	if len(m.targets) == 0 {
		targetRows = append(targetRows, "  "+t.Dim.Render("No workspaces available"))
	} else {
		for i, target := range m.targets {
			mark := " "
			if target.Selected {
				mark = "x"
			}
			line := fmt.Sprintf("  [%s] %s (%s)", mark, target.Title, target.AgentType)
			if i == m.cursor && m.focus == broadcastFocusTargets {
				line = t.SelectedItem.Width(max(m.prompt.Width()+4, 32)).Render(line)
			} else if target.Selected {
				line = t.NormalItem.Bold(true).Render(line)
			} else {
				line = t.Dim.Render(line)
			}
			targetRows = append(targetRows, line)
		}
	}

	countLine := fmt.Sprintf("  %d selected", len(m.selectedWorkspaceIDs()))
	if len(m.selectedWorkspaceIDs()) == 0 {
		countLine = "  " + t.StatusError.Render("Select at least one workspace")
	} else {
		countLine = t.Dim.Render(countLine)
	}

	promptView := m.prompt.View()
	if strings.TrimSpace(m.prompt.Value()) == "" {
		promptView = m.prompt.View()
	}

	moveKeys := fmt.Sprintf("%s/%s", BindingLabel(m.keys.Up), BindingLabel(m.keys.Down))
	toggleKey := BindingLabel(m.keys.Enter)

	help := t.Dim.Render(fmt.Sprintf(
		"  %s: switch focus  %s: move  %s/%s: toggle target  %s: toggle all  %s: send  %s: cancel",
		BindingLabel(m.keys.Tab),
		moveKeys,
		BindingLabel(m.keys.ToggleTarget),
		toggleKey,
		BindingLabel(m.keys.ToggleAll),
		BindingLabel(m.keys.Submit),
		BindingLabel(m.keys.Cancel),
	))
	if m.focus == broadcastFocusPrompt {
		help = t.Dim.Render(fmt.Sprintf(
			"  %s: switch focus  enter: newline  %s: send  %s: cancel",
			BindingLabel(m.keys.Tab),
			BindingLabel(m.keys.Submit),
			BindingLabel(m.keys.Cancel),
		))
	}

	content := strings.Join([]string{
		title,
		"",
		targetLabel,
		strings.Join(targetRows, "\n"),
		countLine,
		"",
		promptLabel,
		promptView,
		"",
		help,
	}, "\n")

	border := t.DialogBorder.
		Padding(1, 2).
		Width(m.prompt.Width() + 8)

	return border.Render(content)
}

func (m BroadcastModel) WithTheme(t theme.Theme) BroadcastModel {
	m.theme = t
	return m
}

func (m BroadcastModel) WithKeyMap(keys BroadcastKeyMap) BroadcastModel {
	m.keys = keys
	return m
}

func DefaultBroadcastKeyMap() BroadcastKeyMap {
	return BroadcastKeyMapFromConfig(config.Default().Keys)
}
