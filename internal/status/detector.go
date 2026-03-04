package status

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/ramtinj/colosseum/internal/agent"
)

type PaneCapturer interface {
	CapturePane(ctx context.Context, target string, lines int) (string, error)
}

type Detector struct {
	capturer    PaneCapturer
	captureLines int
}

func NewDetector(capturer PaneCapturer, captureLines int) *Detector {
	if captureLines <= 0 {
		captureLines = 50
	}
	return &Detector{
		capturer:    capturer,
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

	return DetectFromContent(content, def), content, nil
}

func DetectFromContent(content string, def *agent.AgentDef) agent.Status {
	lines := strings.Split(content, "\n")

	lastNonEmpty := lastNonEmptyLines(lines, 10)

	// Check the most recent lines for idle indicators first.
	// A prompt at the bottom of the output means the agent is idle,
	// regardless of keywords in earlier conversation output.
	bottom := lastNonEmpty
	if len(bottom) > 3 {
		bottom = bottom[len(bottom)-3:]
	}
	if matchesAnyLine(bottom, def.IdlePatterns) {
		return agent.StatusIdle
	}

	if matchesAny(lastNonEmpty, def.WorkingPatterns) {
		return agent.StatusWorking
	}
	if matchesAny(lastNonEmpty, def.WaitingPatterns) {
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
