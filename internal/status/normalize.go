package status

import "strings"

// StripANSI removes ANSI escape sequences from terminal output using a
// single-pass O(n) parser. This prevents escape codes embedded in pane
// content from interfering with regex-based status detection.
func StripANSI(s string) string {
	var buf strings.Builder
	buf.Grow(len(s))

	i := 0
	for i < len(s) {
		if s[i] != '\x1b' {
			buf.WriteByte(s[i])
			i++
			continue
		}

		// ESC found — determine sequence type
		i++
		if i >= len(s) {
			break
		}

		switch s[i] {
		case '[': // CSI: ESC [ <params> <final byte 0x40-0x7E>
			i++
			for i < len(s) && (s[i] < 0x40 || s[i] > 0x7E) {
				i++
			}
			if i < len(s) {
				i++ // skip final byte
			}

		case ']': // OSC: ESC ] <text> (BEL | ST)
			i++
			for i < len(s) {
				if s[i] == '\x07' {
					i++
					break
				}
				if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '\\' {
					i += 2
					break
				}
				i++
			}

		default: // Simple two-byte escape (e.g. ESC M, ESC 7)
			if s[i] >= 0x40 && s[i] <= 0x5F {
				i++
			}
		}
	}

	return buf.String()
}
