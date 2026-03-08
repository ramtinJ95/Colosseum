package agent

import "regexp"

func init() {
	Register(&AgentDef{
		Name:            Codex,
		Binary:          "codex",
		LaunchFlags:     []string{},
		YoloFlags:       []string{"--approval-mode", "full-auto"},
		PasteSingleLine: true,
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
			ChoiceMenuPattern,
			ChoicePromptPattern,
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
