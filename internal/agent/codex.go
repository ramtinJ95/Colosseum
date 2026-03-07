package agent

import "regexp"

func init() {
	Register(&AgentDef{
		Name:        Codex,
		Binary:      "codex",
		LaunchFlags: []string{},
		YoloFlags:   []string{"--approval-mode", "full-auto"},
		IgnorePatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)^\s*(~?/|/).*[·•].*\d+% left.*used\s*$`),
		},
		WorkingPatterns: []*regexp.Regexp{
			BrailleSpinner,
			regexp.MustCompile(`(?i)(thinking|generating|processing|working)`),
			regexp.MustCompile(`(?i)\(.*esc to interrupt.*\)`),
		},
		WaitingPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(approve|deny|explain)`),
			regexp.MustCompile(`(?i)(yes|no|edit)\s*$`),
			regexp.MustCompile(`(?i)(do you want|would you like|should i|shall i)`),
			regexp.MustCompile(`(?i)(which .* prefer|what .* want to tackle|how would you like)`),
			regexp.MustCompile(`\?\s*$`),
		},
		IdlePatterns: []*regexp.Regexp{
			CommonPromptChars,
			regexp.MustCompile(`^\s*[❯›>].*$`),
		},
		ErrorPatterns: []*regexp.Regexp{
			RateLimitPattern,
			PanicPattern,
			AuthErrorPattern,
		},
	})
}
