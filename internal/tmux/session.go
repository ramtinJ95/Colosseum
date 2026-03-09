package tmux

import (
	"context"
	"fmt"
	"strings"
)

const defaultSessionPrefix = "colo-"
const defaultReturnKey = "e"

type Client struct {
	Commander     Commander
	SessionPrefix string
	ReturnKey     string
}

func NewClient(cmdr Commander) *Client {
	return &Client{
		Commander:     cmdr,
		SessionPrefix: defaultSessionPrefix,
		ReturnKey:     defaultReturnKey,
	}
}

func (c *Client) CreateSession(ctx context.Context, name string, startDir string) (string, error) {
	format := BuildFormat(FormatPaneID)
	output, err := c.Commander.Run(ctx, "new-session", "-d", "-s", name, "-c", startDir, "-P", "-F", format)
	if err != nil {
		return "", fmt.Errorf("create session %q: %w", name, err)
	}
	return strings.TrimSpace(output), nil
}

func (c *Client) CreateDetachedSessionWithCommand(ctx context.Context, name string, startDir string, command []string) error {
	args := []string{"new-session", "-d", "-s", name, "-c", startDir}
	args = append(args, command...)
	if _, err := c.Commander.Run(ctx, args...); err != nil {
		return fmt.Errorf("create detached session %q: %w", name, err)
	}
	return nil
}

func (c *Client) KillSession(ctx context.Context, name string) error {
	_, err := c.Commander.Run(ctx, "kill-session", "-t", name)
	if err != nil {
		return fmt.Errorf("kill session %q: %w", name, err)
	}
	return nil
}

func (c *Client) SessionExists(ctx context.Context, name string) bool {
	_, err := c.Commander.Run(ctx, "has-session", "-t", name)
	return err == nil
}

func (c *Client) ListSessions(ctx context.Context) ([]string, error) {
	output, err := c.Commander.Run(ctx, "list-sessions", "-F", FormatSessionName)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	if output == "" {
		return nil, nil
	}

	var sessions []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, c.SessionPrefix) {
			sessions = append(sessions, line)
		}
	}
	return sessions, nil
}

func (c *Client) SwitchClient(ctx context.Context, name string) error {
	dashboardSession, err := c.currentSession(ctx)
	if err != nil {
		return fmt.Errorf("detect current session: %w", err)
	}

	if _, err := c.Commander.Run(
		ctx,
		"bind-key",
		"-N", "Colosseum dashboard",
		"-T", "prefix",
		c.ReturnKey,
		"switch-client",
		"-t", dashboardSession,
	); err != nil {
		return fmt.Errorf("bind dashboard return key: %w", err)
	}

	return c.SwitchSession(ctx, name)
}

func (c *Client) SwitchSession(ctx context.Context, name string) error {
	_, err := c.Commander.Run(ctx, "switch-client", "-t", name)
	if err != nil {
		return fmt.Errorf("switch client to %q: %w", name, err)
	}
	return nil
}

func (c *Client) AttachSession(ctx context.Context, name string) error {
	_, err := c.Commander.Run(ctx, "attach-session", "-t", name)
	if err != nil {
		return fmt.Errorf("attach session %q: %w", name, err)
	}
	return nil
}

func (c *Client) CurrentSession(ctx context.Context) (string, error) {
	return c.currentSession(ctx)
}

func (c *Client) currentSession(ctx context.Context) (string, error) {
	output, err := c.Commander.Run(ctx, "display-message", "-p", "#{session_name}")
	if err != nil {
		return "", fmt.Errorf("display current session: %w", err)
	}
	return strings.TrimSpace(output), nil
}
