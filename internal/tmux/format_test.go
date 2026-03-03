package tmux

import (
	"testing"
)

func TestBuildFormat(t *testing.T) {
	result := BuildFormat(FormatPaneID, FormatPaneWidth, FormatPaneHeight)
	expected := "#{pane_id}\t#{pane_width}\t#{pane_height}"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestBuildFormatSingle(t *testing.T) {
	result := BuildFormat(FormatSessionName)
	if result != "#{session_name}" {
		t.Errorf("expected #{session_name}, got %q", result)
	}
}

func TestParseFields(t *testing.T) {
	fields := ParseFields("%1\t120\t40")
	if len(fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(fields))
	}
	if fields[0] != "%1" {
		t.Errorf("field[0]: expected %%1, got %q", fields[0])
	}
	if fields[1] != "120" {
		t.Errorf("field[1]: expected 120, got %q", fields[1])
	}
	if fields[2] != "40" {
		t.Errorf("field[2]: expected 40, got %q", fields[2])
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"42", 42},
		{"0", 0},
		{"-1", -1},
		{"", 0},
		{"abc", 0},
		{" 10 ", 10},
	}

	for _, tc := range tests {
		got := ParseInt(tc.input)
		if got != tc.expected {
			t.Errorf("ParseInt(%q): expected %d, got %d", tc.input, tc.expected, got)
		}
	}
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"1", true},
		{"0", false},
		{"", false},
		{"true", false},
		{" 1 ", true},
	}

	for _, tc := range tests {
		got := ParseBool(tc.input)
		if got != tc.expected {
			t.Errorf("ParseBool(%q): expected %v, got %v", tc.input, tc.expected, got)
		}
	}
}
