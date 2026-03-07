package dialog

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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

	if updated.focusIndex != int(fieldBranch) {
		t.Fatalf("focus moved to %d, want branch field", updated.focusIndex)
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

func newFocusedPathModel() NewWorkspaceModel {
	model := NewNewWorkspace()
	model.focusIndex = int(fieldPath)
	model.updateFocus()
	return model
}
