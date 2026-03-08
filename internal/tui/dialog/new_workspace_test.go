package dialog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ramtinj/colosseum/internal/workspace"
)

func TestEnterAdvancesPastPathWithoutAcceptingSuggestion(t *testing.T) {
	root := t.TempDir()
	expected := filepath.Join(root, "project-alpha") + "/"
	if err := os.Mkdir(expected, 0o755); err != nil {
		t.Fatalf("mkdir suggestion dir: %v", err)
	}

	model := NewNewWorkspace()
	model.focusIndex = int(fieldPath)
	model.updateFocus()
	model.inputs[fieldPath].SetValue(filepath.Join(root, "pro"))
	model.refreshPathSuggestions()

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if updated.focusIndex != selectorAgent {
		t.Fatalf("focus moved to %d, want agent selector", updated.focusIndex)
	}
	if got := updated.inputs[fieldPath].Value(); got != filepath.Join(root, "pro") {
		t.Fatalf("path value = %q, want %q", got, filepath.Join(root, "pro"))
	}
}

func TestTabExpandsLongestSharedPathPrefix(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"project-alpha", "project-beta"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
	}

	model := newFocusedPathModel()
	model.inputs[fieldPath].SetValue(filepath.Join(root, "pro"))
	model.refreshPathSuggestions()

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})

	if got := updated.inputs[fieldPath].Value(); got != filepath.Join(root, "project-") {
		t.Fatalf("path value = %q, want %q", got, filepath.Join(root, "project-"))
	}
}

func TestRepeatedTabCyclesPathMatches(t *testing.T) {
	root := t.TempDir()
	alpha := filepath.Join(root, "project-alpha") + "/"
	beta := filepath.Join(root, "project-beta") + "/"
	for _, dir := range []string{alpha, beta} {
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	model := newFocusedPathModel()
	model.inputs[fieldPath].SetValue(filepath.Join(root, "project-"))
	model.refreshPathSuggestions()

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	if got := updated.inputs[fieldPath].Value(); got != alpha {
		t.Fatalf("first tab value = %q, want %q", got, alpha)
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyTab})
	if got := updated.inputs[fieldPath].Value(); got != beta {
		t.Fatalf("second tab value = %q, want %q", got, beta)
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyTab})
	if got := updated.inputs[fieldPath].Value(); got != alpha {
		t.Fatalf("third tab value = %q, want %q", got, alpha)
	}
}

func TestEnterEmitsExpandedHomePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	model := NewNewWorkspace()
	model.inputs[fieldName].SetValue("demo")
	model.inputs[fieldPath].SetValue("~/project")
	model.focusIndex = selectorLayout
	model.updateFocus()

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if updated.focusIndex != selectorLayout {
		t.Fatalf("focus changed to %d, want %d", updated.focusIndex, selectorLayout)
	}
	if cmd == nil {
		t.Fatal("expected create command")
	}

	msg := cmd()
	createMsg, ok := msg.(NewWorkspaceMsg)
	if !ok {
		t.Fatalf("message type = %T, want NewWorkspaceMsg", msg)
	}
	if got := createMsg.Path; got != filepath.Join(home, "project") {
		t.Fatalf("path = %q, want %q", got, filepath.Join(home, "project"))
	}
}

func TestConfiguredNavigationKeysStillTypeInPathField(t *testing.T) {
	model := newFocusedPathModel().WithKeyMap(NewWorkspaceKeyMap{
		Up:         key.NewBinding(key.WithKeys("k")),
		Down:       key.NewBinding(key.WithKeys("j")),
		Enter:      key.NewBinding(key.WithKeys("enter")),
		Tab:        key.NewBinding(key.WithKeys("tab")),
		BackTab:    key.NewBinding(key.WithKeys("shift+tab")),
		Cancel:     key.NewBinding(key.WithKeys("esc")),
		SelectPrev: key.NewBinding(key.WithKeys("a")),
		SelectNext: key.NewBinding(key.WithKeys("d")),
	})

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if got := updated.inputs[fieldPath].Value(); got != "j" {
		t.Fatalf("path value after j = %q, want %q", got, "j")
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if got := updated.inputs[fieldPath].Value(); got != "jk" {
		t.Fatalf("path value after k = %q, want %q", got, "jk")
	}
}

func TestEnterEmitsExperimentFields(t *testing.T) {
	model := NewNewWorkspace()
	model.inputs[fieldName].SetValue("Auth comparison")
	model.inputs[fieldPath].SetValue("/repo")
	model.inputs[fieldBaseBranch].SetValue("main")
	model.inputs[fieldPrompt].SetValue("Fix the auth flow")
	model.inputs[fieldCount].SetValue("3")
	model.modeIndex = 2
	model.strategyIndex = 1
	model.focusIndex = selectorLayout
	model.updateFocus()

	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected create command")
	}

	msg := cmd()
	createMsg, ok := msg.(NewWorkspaceMsg)
	if !ok {
		t.Fatalf("message type = %T, want NewWorkspaceMsg", msg)
	}
	if createMsg.Mode != workspace.CreateModeExperimentRun {
		t.Fatalf("mode = %q, want %q", createMsg.Mode, workspace.CreateModeExperimentRun)
	}
	if createMsg.CandidateCount != 3 {
		t.Fatalf("candidate count = %d, want 3", createMsg.CandidateCount)
	}
	if createMsg.AgentStrategy != workspace.ExperimentAgentSelected {
		t.Fatalf("agent strategy = %q, want %q", createMsg.AgentStrategy, workspace.ExperimentAgentSelected)
	}
}

func TestExistingCheckoutSubmitIgnoresHiddenExperimentCount(t *testing.T) {
	model := NewNewWorkspace()
	model.inputs[fieldName].SetValue("demo")
	model.inputs[fieldPath].SetValue("/repo")
	model.inputs[fieldCount].SetValue("not-a-number")
	model.focusIndex = selectorLayout
	model.updateFocus()

	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected create command")
	}

	msg := cmd()
	createMsg, ok := msg.(NewWorkspaceMsg)
	if !ok {
		t.Fatalf("message type = %T, want NewWorkspaceMsg", msg)
	}
	if createMsg.Mode != workspace.CreateModeExistingCheckout {
		t.Fatalf("mode = %q, want %q", createMsg.Mode, workspace.CreateModeExistingCheckout)
	}
}

func TestViewHidesExperimentOnlyControlsOutsideExperimentMode(t *testing.T) {
	model := NewNewWorkspace()

	view := model.View()

	for _, hidden := range []string{"Prompt:", "Strategy:", "Count:"} {
		if containsNormalized(view, hidden) {
			t.Fatalf("view unexpectedly contains %q:\n%s", hidden, view)
		}
	}
}

func TestExperimentAllSupportedHidesAgentAndCount(t *testing.T) {
	model := NewNewWorkspace()
	model.modeIndex = 2
	model.focusIndex = selectorMode
	model.updateFocus()

	view := model.View()

	for _, hidden := range []string{" Agent:", " Count:"} {
		if containsNormalized(view, hidden) {
			t.Fatalf("view unexpectedly contains %q:\n%s", hidden, view)
		}
	}
	if !containsNormalized(view, "Strategy:") {
		t.Fatalf("view should contain strategy selector:\n%s", view)
	}
}

func containsNormalized(view, needle string) bool {
	return strings.Contains(stripANSI(view), needle)
}

func stripANSI(value string) string {
	var b strings.Builder
	inEscape := false
	for i := 0; i < len(value); i++ {
		ch := value[i]
		switch {
		case inEscape && ch == 'm':
			inEscape = false
		case inEscape:
			continue
		case ch == 0x1b:
			inEscape = true
		default:
			b.WriteByte(ch)
		}
	}
	return b.String()
}

func newFocusedPathModel() NewWorkspaceModel {
	model := NewNewWorkspace()
	model.focusIndex = int(fieldPath)
	model.updateFocus()
	return model
}
