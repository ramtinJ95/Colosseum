package agent

import "regexp"

func init() {
	Register(&AgentDef{
		Name:        OpenCode,
		Binary:      "opencode",
		LaunchFlags: []string{},
		YoloFlags:   []string{},
		WorkingPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)esc\s*(to\s+)?interrupt`),
			BrailleSpinner,
		},
		WaitingPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)enter to select`),
			regexp.MustCompile(`(?i)esc to cancel`),
			regexp.MustCompile(`(?i)\(y/n\)`),
			regexp.MustCompile(`(?i)\[y/n\]`),
			regexp.MustCompile(`(?i)(continue|proceed)\?`),
			regexp.MustCompile(`(?i)(approve|allow)`),
			regexp.MustCompile(`❯\s*[123]\.`),
			regexp.MustCompile(`^\s*>{1,2}\s*$`),
		},
		ErrorPatterns: []*regexp.Regexp{
			RateLimitPattern,
			PanicPattern,
		},
	})
}
