package status

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ramtinj/colosseum/internal/agent"
)

func TestDetectFromContent_Fixtures(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "testdata", "fixtures")

	tests := []struct {
		agentType agent.AgentType
		subdir    string
		expected  agent.Status
	}{
		{agent.Claude, "claude/working", agent.StatusWorking},
		{agent.Claude, "claude/waiting", agent.StatusWaiting},
		{agent.Claude, "claude/idle", agent.StatusIdle},
		{agent.Claude, "claude/error", agent.StatusError},
		{agent.Codex, "codex/working", agent.StatusWorking},
		{agent.Codex, "codex/waiting", agent.StatusWaiting},
		{agent.Codex, "codex/idle", agent.StatusIdle},
		{agent.Codex, "codex/error", agent.StatusError},
		{agent.Gemini, "gemini/working", agent.StatusWorking},
		{agent.Gemini, "gemini/waiting", agent.StatusWaiting},
		{agent.Gemini, "gemini/idle", agent.StatusIdle},
		{agent.Gemini, "gemini/error", agent.StatusError},
	}

	for _, tt := range tests {
		dir := filepath.Join(fixtureRoot, tt.subdir)
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read fixture dir %s: %v", dir, err)
		}

		def, ok := agent.Get(tt.agentType)
		if !ok {
			t.Fatalf("agent %s not registered", tt.agentType)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := filepath.Join(tt.subdir, entry.Name())
			t.Run(name, func(t *testing.T) {
				content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
				if err != nil {
					t.Fatalf("read fixture %s: %v", name, err)
				}

				got := DetectFromContent(string(content), def)
				if got != tt.expected {
					t.Errorf("DetectFromContent(%s) = %s, want %s", name, got, tt.expected)
				}
			})
		}
	}
}

func TestDetectFromContent_EmptyContent(t *testing.T) {
	def, _ := agent.Get(agent.Claude)
	got := DetectFromContent("", def)
	if got != agent.StatusUnknown {
		t.Errorf("empty content: got %s, want Unknown", got)
	}
}

func TestDetectFromContent_PriorityOrder(t *testing.T) {
	def, _ := agent.Get(agent.Claude)

	content := "Error: Rate limit exceeded 429\n⠹ Working on task (esc to interrupt)"
	got := DetectFromContent(content, def)
	if got != agent.StatusWorking {
		t.Errorf("working should take priority over error, got %s", got)
	}
}

func TestDetectFromContent_IdleOverridesOlderContent(t *testing.T) {
	def, _ := agent.Get(agent.Claude)

	// Simulates Claude idle at prompt with typed text, after a conversation
	// containing words like "running". Status bar lines are filtered out,
	// leaving "❯ nice" as the last non-empty line.
	content := "● It's running Arch Linux with kernel 6.18.13-arch1-1.\n\n❯ nice\n[Opus 4.6] Tokens: 32,171/200,000 | Remaining: 167,829 | Used: 16.0%\n-- INSERT -- >> bypass permissions on (shift+tab to cycle)"
	got := DetectFromContent(content, def)
	if got != agent.StatusIdle {
		t.Errorf("idle prompt should override older conversation content, got %s", got)
	}
}

func TestDetectFromContent_IdleBottomPriority(t *testing.T) {
	def, _ := agent.Get(agent.Claude)

	// Working keywords in old output, but prompt at the bottom.
	content := "⠹ Reading file (esc to interrupt)\n\nDone! Here are the results.\n\n>"
	got := DetectFromContent(content, def)
	if got != agent.StatusIdle {
		t.Errorf("idle prompt at bottom should win over working in older lines, got %s", got)
	}
}

func TestDetectFromContent_WorkingBeforePromptBottomWins(t *testing.T) {
	def, _ := agent.Get(agent.Codex)

	content := "• Working (26s • esc to interrupt)\n\n› Use /skills to list available skills\n\n~/workspace/prefect-data-flows · gpt-5.4 high · 86% left · 249K used"
	got := DetectFromContent(content, def)
	if got != agent.StatusWorking {
		t.Errorf("working line directly before prompt-like bottom line should stay Working, got %s", got)
	}
}

func TestDetectFromContent_PromptWithRecentQuestionMeansWaitingForClaude(t *testing.T) {
	def, _ := agent.Get(agent.Claude)

	content := "● I didn't do an independent analysis of what would matter most to you as a user.\nDo you have a different sense of what's most valuable?\n\n❯"
	got := DetectFromContent(content, def)
	if got != agent.StatusWaiting {
		t.Errorf("prompt with recent question should be Waiting, got %s", got)
	}
}

func TestDetectFromContent_PromptWithRecentQuestionMeansWaitingForCodex(t *testing.T) {
	def, _ := agent.Get(agent.Codex)

	content := "• I have enough now.\nWould you like me to do a deeper code-level audit of a specific package?\n\n›"
	got := DetectFromContent(content, def)
	if got != agent.StatusWaiting {
		t.Errorf("prompt with recent question should be Waiting, got %s", got)
	}
}

func TestDetectFromContent_StatusBarFiltered(t *testing.T) {
	def, _ := agent.Get(agent.Claude)

	// "bypass permissions" in the status bar was triggering false Waiting
	// via the (?i)permission pattern. Status bar should be filtered out.
	content := "· Fluttering.. (thought for 2s)\n└ Tip: Double-tap esc to rewind\n\n❯\n[Opus 4.6] Tokens: 32,863/200,000 | Remaining: 167,137 | Used: 16.0%\n-- INSERT -- >> bypass permissions on (shift+tab to cycle)"
	got := DetectFromContent(content, def)
	if got == agent.StatusWaiting {
		t.Errorf("status bar 'bypass permissions' should not trigger Waiting, got %s", got)
	}
	if got != agent.StatusIdle {
		t.Errorf("expected Idle (prompt visible), got %s", got)
	}
}
