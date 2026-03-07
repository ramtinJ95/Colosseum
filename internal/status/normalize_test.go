package status

import "testing"

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no escapes",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "color codes",
			input: "\x1b[31mred text\x1b[0m",
			want:  "red text",
		},
		{
			name:  "bold and reset",
			input: "\x1b[1mbold\x1b[22m normal",
			want:  "bold normal",
		},
		{
			name:  "cursor movement",
			input: "before\x1b[2Jafter",
			want:  "beforeafter",
		},
		{
			name:  "OSC with BEL",
			input: "text\x1b]0;window title\x07more",
			want:  "textmore",
		},
		{
			name:  "OSC with ST",
			input: "text\x1b]0;title\x1b\\more",
			want:  "textmore",
		},
		{
			name:  "mixed color around spinner",
			input: "\x1b[36m⠹\x1b[0m Working",
			want:  "⠹ Working",
		},
		{
			name:  "utf8 preserved",
			input: "\x1b[32m❯\x1b[0m hello 世界",
			want:  "❯ hello 世界",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "trailing ESC",
			input: "text\x1b",
			want:  "text",
		},
		{
			name:  "complex CSI with params",
			input: "\x1b[38;5;196mcolored\x1b[0m",
			want:  "colored",
		},
		{
			name:  "simple two-byte escape",
			input: "a\x1bMb",
			want:  "ab",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripANSI(tt.input)
			if got != tt.want {
				t.Errorf("StripANSI(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
