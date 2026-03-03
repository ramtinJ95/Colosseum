package tmux

import (
	"strconv"
	"strings"
)

const (
	FormatSessionName = "#{session_name}"
	FormatPaneID      = "#{pane_id}"
	FormatPaneWidth   = "#{pane_width}"
	FormatPaneHeight  = "#{pane_height}"
	FormatWindowID    = "#{window_id}"
	FormatWindowIndex = "#{window_index}"

	fieldSeparator = "\t"
)

func BuildFormat(vars ...string) string {
	return strings.Join(vars, fieldSeparator)
}

func ParseFields(output string) []string {
	return strings.Split(output, fieldSeparator)
}

func ParseInt(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
}

func ParseBool(s string) bool {
	return strings.TrimSpace(s) == "1"
}
