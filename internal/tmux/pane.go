package tmux

import (
	"context"
	"fmt"
	"strings"
)

type PaneInfo struct {
	ID     string
	Width  int
	Height int
}

func (c *Client) SplitWindow(ctx context.Context, session string, horizontal bool, startDir string) (string, error) {
	orientation := "-v"
	if horizontal {
		orientation = "-h"
	}

	format := BuildFormat(FormatPaneID)
	output, err := c.Commander.Run(ctx,
		"split-window", orientation,
		"-t", c.fullName(session),
		"-c", startDir,
		"-P", "-F", format,
	)
	if err != nil {
		return "", fmt.Errorf("split window in %q: %w", session, err)
	}
	return strings.TrimSpace(output), nil
}

func (c *Client) CapturePane(ctx context.Context, target string, lines int) (string, error) {
	output, err := c.Commander.Run(ctx,
		"capture-pane", "-p",
		"-t", target,
		"-S", fmt.Sprintf("-%d", lines),
	)
	if err != nil {
		return "", fmt.Errorf("capture pane %q: %w", target, err)
	}
	return output, nil
}

func (c *Client) SendKeys(ctx context.Context, target string, keys string) error {
	if _, err := c.Commander.Run(ctx, "send-keys", "-t", target, "-l", keys); err != nil {
		return fmt.Errorf("send keys to %q: %w", target, err)
	}
	if _, err := c.Commander.Run(ctx, "send-keys", "-t", target, "Enter"); err != nil {
		return fmt.Errorf("send enter to %q: %w", target, err)
	}
	return nil
}

func (c *Client) SendLiteralKeys(ctx context.Context, target string, text string) error {
	_, err := c.Commander.Run(ctx, "send-keys", "-t", target, "-l", text)
	if err != nil {
		return fmt.Errorf("send literal keys to %q: %w", target, err)
	}
	return nil
}

func (c *Client) ResizePane(ctx context.Context, target string, width, height int) error {
	_, err := c.Commander.Run(ctx,
		"resize-pane", "-t", target,
		"-x", fmt.Sprintf("%d", width),
		"-y", fmt.Sprintf("%d", height),
	)
	if err != nil {
		return fmt.Errorf("resize pane %q: %w", target, err)
	}
	return nil
}

func (c *Client) ListPanes(ctx context.Context, session string) ([]PaneInfo, error) {
	format := BuildFormat(FormatPaneID, FormatPaneWidth, FormatPaneHeight)
	output, err := c.Commander.Run(ctx, "list-panes", "-t", c.fullName(session), "-F", format)
	if err != nil {
		return nil, fmt.Errorf("list panes in %q: %w", session, err)
	}

	if output == "" {
		return nil, nil
	}

	var panes []PaneInfo
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := ParseFields(line)
		if len(fields) < 3 {
			continue
		}
		panes = append(panes, PaneInfo{
			ID:     fields[0],
			Width:  ParseInt(fields[1]),
			Height: ParseInt(fields[2]),
		})
	}
	return panes, nil
}
