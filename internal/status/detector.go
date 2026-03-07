package status

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/ramtinj/colosseum/internal/agent"
)

var (
	brailleRange      = regexp.MustCompile(`[\x{2800}-\x{28FF}]`)
	questionEndsLine  = regexp.MustCompile(`\?\s*$`)
)

type PaneCapturer interface {
	CapturePane(ctx context.Context, target string, lines int) (string, error)
	CapturePaneTitle(ctx context.Context, target string) (string, error)
}

type Detector struct {
	capturer     PaneCapturer
	captureLines int
}

func NewDetector(capturer PaneCapturer, captureLines int) *Detector {
	if captureLines <= 0 {
		captureLines = 50
	}
	return &Detector{
		capturer:     capturer,
		captureLines: captureLines,
	}
}

func (d *Detector) Detect(ctx context.Context, paneTarget string, agentType agent.AgentType) (agent.Status, string, error) {
	content, err := d.capturer.CapturePane(ctx, paneTarget, d.captureLines)
	if err != nil {
		return agent.StatusStopped, "", fmt.Errorf("capture pane %q: %w", paneTarget, err)
	}

	def, ok := agent.Get(agentType)
	if !ok {
		return agent.StatusUnknown, content, fmt.Errorf("unknown agent type: %s", agentType)
	}

	status := DetectFromContent(content, def)

	// Pane title is a supplementary working signal: Claude Code sets braille
	// spinner characters in the tmux pane title while actively processing.
	// Only upgrade Unknown — when content definitively says Idle we trust it
	// because the pane title is sticky and may not be cleared after a crash.
	if status == agent.StatusUnknown {
		if title, err := d.capturer.CapturePaneTitle(ctx, paneTarget); err == nil && titleIndicatesWorking(title) {
			status = agent.StatusWorking
		}
	}

	return status, content, nil
}

func DetectFromContent(content string, def *agent.AgentDef) agent.Status {
	content = StripANSI(content)
	lines := strings.Split(content, "\n")

	// Filter out lines matching agent-specific ignore patterns (e.g. status bars)
	// before running detection, so UI chrome doesn't trigger false matches.
	filtered := filterIgnored(lines, def.IgnorePatterns)

	lastNonEmpty := lastNonEmptyLines(filtered, 30)

	// Check only the very last non-empty line for idle indicators.
	// A prompt at the bottom means the agent is idle, regardless of
	// keywords in earlier output. Using only the last line avoids
	// false idle detection from selection arrows (❯ Yes, allow once)
	// which appear above other menu items in waiting prompts.
	var bottom []string
	if len(lastNonEmpty) > 0 {
		bottom = lastNonEmpty[len(lastNonEmpty)-1:]
	}
	if matchesAnyLine(bottom, def.IdlePatterns) {
		recent := linesBeforeBottom(lastNonEmpty, 3)

		// If the agent is visibly still running on the line right above the
		// prompt/helper line, keep it in Working rather than flickering back.
		if matchesAnyLine(linesBeforeBottom(lastNonEmpty, 1), def.WorkingPatterns) {
			return agent.StatusWorking
		}

		// A visible prompt with a recent explicit question/choice means the
		// agent is waiting on the user, not merely idle at a fresh prompt.
		if isPromptOnly(bottom[0]) {
			if matchesAny(recent, def.WaitingPatterns) {
				return agent.StatusWaiting
			}
			// Catch natural-language questions ("Do you have...?") that
			// aren't covered by the specific WaitingPatterns. Safe here
			// because the window is only the 3 lines above the prompt.
			if matchesAnyLine(recent, []*regexp.Regexp{questionEndsLine}) {
				return agent.StatusWaiting
			}
		}

		return agent.StatusIdle
	}

	if matchesAny(lastNonEmpty, def.WorkingPatterns) {
		return agent.StatusWorking
	}
	// Waiting patterns are more prone to false positives from old output,
	// so restrict to the last 10 non-empty lines in the non-idle path.
	recentForWaiting := lastNonEmpty
	if len(recentForWaiting) > 10 {
		recentForWaiting = recentForWaiting[len(recentForWaiting)-10:]
	}
	if matchesAny(recentForWaiting, def.WaitingPatterns) {
		return agent.StatusWaiting
	}
	if matchesAny(lastNonEmpty, def.ErrorPatterns) {
		return agent.StatusError
	}

	return agent.StatusUnknown
}

func lastNonEmptyLines(lines []string, n int) []string {
	var result []string
	for i := len(lines) - 1; i >= 0 && len(result) < n; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed != "" {
			result = append(result, lines[i])
		}
	}
	// Reverse to maintain order
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result
}

func linesBeforeBottom(lines []string, n int) []string {
	if len(lines) <= 1 || n <= 0 {
		return nil
	}

	end := len(lines) - 1
	start := max(0, end-n)
	return lines[start:end]
}

func isPromptOnly(line string) bool {
	return regexp.MustCompile(`^\s*[❯›>$]\s*$`).MatchString(line)
}

func matchesAny(lines []string, patterns []*regexp.Regexp) bool {
	joined := strings.Join(lines, "\n")
	for _, p := range patterns {
		if p.MatchString(joined) {
			return true
		}
	}
	return false
}

func matchesAnyLine(lines []string, patterns []*regexp.Regexp) bool {
	for _, line := range lines {
		for _, p := range patterns {
			if p.MatchString(line) {
				return true
			}
		}
	}
	return false
}

func titleIndicatesWorking(title string) bool {
	return brailleRange.MatchString(title)
}

func filterIgnored(lines []string, patterns []*regexp.Regexp) []string {
	if len(patterns) == 0 {
		return lines
	}
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		ignored := false
		for _, p := range patterns {
			if p.MatchString(line) {
				ignored = true
				break
			}
		}
		if !ignored {
			result = append(result, line)
		}
	}
	return result
}
