package dialog

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestEnterAcceptsPathSuggestionBeforeAdvancing(t *testing.T) {
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

	if updated.focusIndex != int(fieldPath) {
		t.Fatalf("focus moved to %d, want path field", updated.focusIndex)
	}
	if got := updated.inputs[fieldPath].Value(); got != expected {
		t.Fatalf("path value = %q, want %q", got, expected)
	}
}
