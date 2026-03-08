package worktrunk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const defaultTimeout = 15 * time.Second

type Commander interface {
	Run(ctx context.Context, args ...string) (string, error)
}

type CommandError struct {
	Binary string
	Args   []string
	Stderr string
}

func (e *CommandError) Error() string {
	return fmt.Sprintf("%s %s: %s", e.Binary, strings.Join(e.Args, " "), e.Stderr)
}

type execCommander struct {
	binary  string
	timeout time.Duration
}

func (c *execCommander) Run(ctx context.Context, args ...string) (string, error) {
	timeout := c.timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr != "" {
			return "", &CommandError{Binary: c.binary, Args: args, Stderr: stderrStr}
		}
		return "", fmt.Errorf("%s %s: %w", c.binary, strings.Join(args, " "), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

type Client struct {
	wt  Commander
	git Commander
}

type Info struct {
	Branch         string `json:"branch"`
	Path           string `json:"path"`
	Kind           string `json:"kind"`
	MainState      string `json:"main_state"`
	OperationState string `json:"operation_state"`
	IsMain         bool   `json:"is_main"`
	IsCurrent      bool   `json:"is_current"`
	Worktree       struct {
		State    string `json:"state"`
		Reason   string `json:"reason"`
		Detached bool   `json:"detached"`
	} `json:"worktree"`
}

func (i Info) BranchName() string {
	return strings.TrimSpace(i.Branch)
}

func (i Info) PathValue() string {
	return filepath.Clean(strings.TrimSpace(i.Path))
}

type Snapshot struct {
	RepoRoot      string
	CheckoutPath  string
	Branch        string
	BaseBranch    string
	DefaultBranch string
	MergeBaseSHA  string
}

func NewClient() *Client {
	return &Client{
		wt:  &execCommander{binary: "wt", timeout: defaultTimeout},
		git: &execCommander{binary: "git", timeout: defaultTimeout},
	}
}

func NewClientWithCommanders(wt Commander, git Commander) *Client {
	return &Client{wt: wt, git: git}
}

func (c *Client) IsAvailable() bool {
	_, err := exec.LookPath("wt")
	return err == nil
}

func (c *Client) Create(ctx context.Context, repoPath, branch, base string) (Snapshot, error) {
	args := []string{"-C", repoPath, "switch", "--create", "--no-cd"}
	if strings.TrimSpace(base) != "" {
		args = append(args, "--base", base)
	}
	args = append(args, branch)
	if _, err := c.wt.Run(ctx, args...); err != nil {
		return Snapshot{}, err
	}
	return c.ResolveBranch(ctx, repoPath, branch, base)
}

func (c *Client) Remove(ctx context.Context, repoPath string, branches ...string) error {
	args := []string{"-C", repoPath, "remove", "--foreground", "--yes", "--no-delete-branch"}
	args = append(args, branches...)
	_, err := c.wt.Run(ctx, args...)
	return err
}

func (c *Client) Merge(ctx context.Context, checkoutPath, target string) error {
	args := []string{"-C", checkoutPath, "merge", "--yes"}
	if strings.TrimSpace(target) != "" {
		args = append(args, target)
	}
	_, err := c.wt.Run(ctx, args...)
	return err
}

func (c *Client) CopyIgnored(ctx context.Context, checkoutPath string) error {
	_, err := c.wt.Run(ctx, "-C", checkoutPath, "step", "copy-ignored")
	return err
}

func (c *Client) List(ctx context.Context, repoPath string) ([]Info, error) {
	output, err := c.wt.Run(ctx, "-C", repoPath, "list", "--format=json")
	if err != nil {
		return nil, err
	}
	var infos []Info
	if err := json.Unmarshal([]byte(output), &infos); err != nil {
		return nil, fmt.Errorf("parsing wt list output: %w", err)
	}
	return infos, nil
}

func (c *Client) ResolvePath(ctx context.Context, checkoutPath string) (Snapshot, error) {
	repoRoot, err := c.git.Run(ctx, "-C", checkoutPath, "rev-parse", "--show-toplevel")
	if err != nil {
		return Snapshot{}, fmt.Errorf("resolving repo root for %s: %w", checkoutPath, err)
	}
	infos, err := c.List(ctx, repoRoot)
	if err != nil {
		return Snapshot{}, fmt.Errorf("listing worktrees: %w", err)
	}

	normalized := NormalizePath(checkoutPath)
	var info Info
	found := false
	for _, candidate := range infos {
		if candidate.Kind != "worktree" {
			continue
		}
		candidatePath := candidate.PathValue()
		if !pathContains(normalized, candidatePath) {
			continue
		}
		if !found || len(candidatePath) > len(info.PathValue()) {
			info = candidate
			found = true
		}
	}
	if !found {
		return Snapshot{}, fmt.Errorf("worktree %q not found in wt list output", checkoutPath)
	}

	defaultBranch := defaultBranchFromInfos(infos)
	if defaultBranch == "" {
		defaultBranch, err = c.defaultBranch(ctx, repoRoot)
		if err != nil {
			return Snapshot{}, fmt.Errorf("default branch not found in wt list output: %w", err)
		}
	}
	mergeBase, err := c.mergeBase(ctx, repoRoot, info.Branch, defaultBranch)
	if err != nil {
		return Snapshot{}, err
	}
	return Snapshot{
		RepoRoot:      repoRoot,
		CheckoutPath:  info.PathValue(),
		Branch:        info.Branch,
		BaseBranch:    defaultBranch,
		DefaultBranch: defaultBranch,
		MergeBaseSHA:  mergeBase,
	}, nil
}

func (c *Client) ResolveBranch(ctx context.Context, repoPath, branch, base string) (Snapshot, error) {
	infos, err := c.List(ctx, repoPath)
	if err != nil {
		return Snapshot{}, fmt.Errorf("listing worktrees: %w", err)
	}
	for _, info := range infos {
		if info.Kind != "worktree" || info.Branch != branch || strings.TrimSpace(info.Path) == "" {
			continue
		}
		defaultBranch := defaultBranchFromInfos(infos)
		if defaultBranch == "" {
			defaultBranch, err = c.defaultBranch(ctx, repoPath)
			if err != nil {
				return Snapshot{}, fmt.Errorf("default branch not found in wt list output: %w", err)
			}
		}
		baseBranch := strings.TrimSpace(base)
		if baseBranch == "" {
			baseBranch = defaultBranch
		}
		mergeBase, err := c.mergeBase(ctx, repoPath, branch, baseBranch)
		if err != nil {
			return Snapshot{}, err
		}
		return Snapshot{
			RepoRoot:      repoPath,
			CheckoutPath:  info.PathValue(),
			Branch:        branch,
			BaseBranch:    baseBranch,
			DefaultBranch: defaultBranch,
			MergeBaseSHA:  mergeBase,
		}, nil
	}
	return Snapshot{}, fmt.Errorf("worktree for branch %q not found", branch)
}

func (c *Client) defaultBranch(ctx context.Context, repoPath string) (string, error) {
	remoteHEAD, err := c.git.Run(ctx, "-C", repoPath, "symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD")
	if err == nil && strings.TrimSpace(remoteHEAD) != "" {
		return strings.TrimPrefix(strings.TrimSpace(remoteHEAD), "origin/"), nil
	}
	branch, err := c.git.Run(ctx, "-C", repoPath, "branch", "--show-current")
	if err != nil {
		return "", fmt.Errorf("resolving default branch for %s: %w", repoPath, err)
	}
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return "", fmt.Errorf("resolving default branch for %s: empty branch", repoPath)
	}
	return branch, nil
}

func defaultBranchFromInfos(infos []Info) string {
	for _, info := range infos {
		if info.IsMain && strings.TrimSpace(info.Branch) != "" {
			return strings.TrimSpace(info.Branch)
		}
	}
	return ""
}

func (c *Client) mergeBase(ctx context.Context, repoPath, branch, base string) (string, error) {
	if strings.TrimSpace(branch) == "" || strings.TrimSpace(base) == "" {
		return "", nil
	}
	sha, err := c.git.Run(ctx, "-C", repoPath, "merge-base", branch, base)
	if err != nil {
		var cmdErr *CommandError
		if errors.As(err, &cmdErr) && strings.Contains(cmdErr.Stderr, "no merge base") {
			return "", nil
		}
		return "", fmt.Errorf("resolving merge base for %s and %s: %w", branch, base, err)
	}
	return strings.TrimSpace(sha), nil
}

func NormalizePath(path string) string {
	return filepath.Clean(path)
}

func pathContains(path string, root string) bool {
	if NormalizePath(path) == NormalizePath(root) {
		return true
	}
	rel, err := filepath.Rel(NormalizePath(root), NormalizePath(path))
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
