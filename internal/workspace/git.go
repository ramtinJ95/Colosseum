package workspace

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type GitInspector interface {
	RepoRoot(ctx context.Context, path string) (string, error)
	CurrentBranch(ctx context.Context, path string) (string, error)
	DefaultBranch(ctx context.Context, path string) (string, error)
	MergeBase(ctx context.Context, path string, left string, right string) (string, error)
}

type gitInspector struct{}

func NewGitInspector() GitInspector {
	return gitInspector{}
}

func (gitInspector) RepoRoot(ctx context.Context, path string) (string, error) {
	output, err := runGit(ctx, path, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("discovering repo root for %q: %w", path, err)
	}
	return strings.TrimSpace(output), nil
}

func (gitInspector) CurrentBranch(ctx context.Context, path string) (string, error) {
	output, err := runGit(ctx, path, "branch", "--show-current")
	if err != nil {
		return "", fmt.Errorf("discovering current branch for %q: %w", path, err)
	}
	return strings.TrimSpace(output), nil
}

func (g gitInspector) DefaultBranch(ctx context.Context, path string) (string, error) {
	output, err := runGit(ctx, path, "symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD")
	if err == nil {
		branch := strings.TrimPrefix(strings.TrimSpace(output), "origin/")
		if branch != "" {
			return branch, nil
		}
	}
	return g.CurrentBranch(ctx, path)
}

func (gitInspector) MergeBase(ctx context.Context, path string, left string, right string) (string, error) {
	if strings.TrimSpace(left) == "" || strings.TrimSpace(right) == "" {
		return "", nil
	}

	output, err := runGit(ctx, path, "merge-base", left, right)
	if err != nil {
		return "", fmt.Errorf("resolving merge base for %q and %q in %q: %w", left, right, path, err)
	}
	return strings.TrimSpace(output), nil
}

func runGit(ctx context.Context, path string, args ...string) (string, error) {
	cmdArgs := []string{"-C", path}
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.CommandContext(ctx, "git", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			return "", fmt.Errorf("git %s: %w", strings.Join(cmdArgs, " "), err)
		}
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(cmdArgs, " "), err, trimmed)
	}
	return string(output), nil
}
