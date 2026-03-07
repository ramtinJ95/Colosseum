package tmux

import (
	"context"
	"fmt"
	"strings"
)

const defaultSessionPrefix = "colo-"
const dashboardReturnKey = "e"

type Client struct {
	Commander     Commander
	SessionPrefix string
}

func NewClient(cmdr Commander) *Client {
	return &Client{
		Commander:     cmdr,
		SessionPrefix: defaultSessionPrefix,
	}
}

func (c *Client) CreateSession(ctx context.Context, name string, startDir string) (string, error) {
	format := BuildFormat(FormatPaneID)
	output, err := c.Commander.Run(ctx, "new-session", "-d", "-s", c.fullName(name), "-c", startDir, "-P", "-F", format)
	if err != nil {
		return "", fmt.Errorf("create session %q: %w", name, err)
	}
	return strings.TrimSpace(output), nil
}

func (c *Client) KillSession(ctx context.Context, name string) error {
	_, err := c.Commander.Run(ctx, "kill-session", "-t", c.fullName(name))
	if err != nil {
		return fmt.Errorf("kill session %q: %w", name, err)
	}
	return nil
}

func (c *Client) SessionExists(ctx context.Context, name string) bool {
	_, err := c.Commander.Run(ctx, "has-session", "-t", c.fullName(name))
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
			sessions = append(sessions, strings.TrimPrefix(line, c.SessionPrefix))
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
		dashboardReturnKey,
		"switch-client",
		"-t", dashboardSession,
	); err != nil {
		return fmt.Errorf("bind dashboard return key: %w", err)
	}

	_, err = c.Commander.Run(ctx, "switch-client", "-t", c.fullName(name))
	if err != nil {
		return fmt.Errorf("switch client to %q: %w", name, err)
	}
	return nil
}

func (c *Client) currentSession(ctx context.Context) (string, error) {
	output, err := c.Commander.Run(ctx, "display-message", "-p", "#{session_name}")
	if err != nil {
		return "", fmt.Errorf("display current session: %w", err)
	}
	return strings.TrimSpace(output), nil
}

func (c *Client) fullName(name string) string {
	return c.SessionPrefix + name
}
