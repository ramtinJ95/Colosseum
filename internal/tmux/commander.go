package tmux

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

const DefaultTimeout = 5 * time.Second

type Commander interface {
	Run(ctx context.Context, args ...string) (string, error)
	RunInteractive(ctx context.Context, args ...string) error
}

type TmuxError struct {
	Args   []string
	Stderr string
}

func (e *TmuxError) Error() string {
	return fmt.Sprintf("tmux %s: %s", strings.Join(e.Args, " "), e.Stderr)
}

type ExecCommander struct {
	TmuxPath string
	Timeout  time.Duration
}

func NewExecCommander() *ExecCommander {
	return &ExecCommander{
		TmuxPath: "tmux",
		Timeout:  DefaultTimeout,
	}
}

func (c *ExecCommander) Run(ctx context.Context, args ...string) (string, error) {
	timeout := c.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.TmuxPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr != "" {
			return "", &TmuxError{Args: args, Stderr: stderrStr}
		}
		return "", fmt.Errorf("tmux %s: %w", strings.Join(args, " "), err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

func (c *ExecCommander) RunInteractive(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, c.TmuxPath, args...)
	var stderr bytes.Buffer
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr != "" {
			return &TmuxError{Args: args, Stderr: stderrStr}
		}
		return fmt.Errorf("tmux %s: %w", strings.Join(args, " "), err)
	}

	return nil
}

func IsSessionNotFound(err error) bool {
	var tmuxErr *TmuxError
	if errors.As(err, &tmuxErr) {
		return strings.Contains(tmuxErr.Stderr, "session not found")
	}
	return false
}

func IsPaneNotFound(err error) bool {
	var tmuxErr *TmuxError
	if errors.As(err, &tmuxErr) {
		return strings.Contains(tmuxErr.Stderr, "can't find pane")
	}
	return false
}
