package workspace

import "testing"

func TestIsValidLayout(t *testing.T) {
	if !IsValidLayout(LayoutAgent) {
		t.Fatal("expected agent layout to be valid")
	}
	if !IsValidLayout(LayoutAgentShell) {
		t.Fatal("expected agent-shell layout to be valid")
	}
	if !IsValidLayout(LayoutAgentShellLogs) {
		t.Fatal("expected agent-shell-logs layout to be valid")
	}
	if IsValidLayout(LayoutType("broken")) {
		t.Fatal("expected broken layout to be invalid")
	}
}
