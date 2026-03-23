// Package git provides thin wrappers around os/exec git commands for
// archive-scoped commit and push operations. It shells out to git
// directly — no go-git dependency.
package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Client wraps git operations scoped to an archive root inside a git worktree.
type Client struct {
	// archiveRoot is the absolute, canonicalized path to the archive directory.
	archiveRoot string
	// repoRoot is the absolute path to the git worktree root.
	repoRoot string
	// relArchive is the repo-root-relative path used for git add.
	relArchive string
}

// NewClient creates a git Client after pre-flight validation:
//  1. git must be on PATH
//  2. archiveRoot must be inside a git worktree
//
// Returns an error if either check fails.
func NewClient(archiveRoot string) (*Client, error) {
	// Pre-flight: git on PATH.
	if _, err := exec.LookPath("git"); err != nil {
		return nil, fmt.Errorf("git not found on PATH: %w", err)
	}

	// Canonicalize archive root (resolve symlinks).
	absArchive, err := filepath.EvalSymlinks(archiveRoot)
	if err != nil {
		// If the directory doesn't exist yet, use Abs instead.
		absArchive, err = filepath.Abs(archiveRoot)
		if err != nil {
			return nil, fmt.Errorf("cannot resolve archive root %q: %w", archiveRoot, err)
		}
	}

	// Pre-flight: archive root inside a git worktree.
	// The archive directory may not exist yet (first sync), so walk up
	// to the nearest existing ancestor to run rev-parse.
	gitDir := absArchive
	for {
		if info, statErr := os.Stat(gitDir); statErr == nil && info.IsDir() {
			break
		}
		parent := filepath.Dir(gitDir)
		if parent == gitDir {
			return nil, fmt.Errorf("no existing ancestor directory found for archive root %q", absArchive)
		}
		gitDir = parent
	}
	out, err := runGit(gitDir, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, fmt.Errorf("archive root %q is not inside a git repository: %w", absArchive, err)
	}
	repoRoot := strings.TrimSpace(out)

	// When the archive dir doesn't exist yet, absArchive may not be
	// fully resolved (e.g., /var/... vs /private/var/... on macOS).
	// Re-derive the canonical archive path from the resolved ancestor.
	if gitDir != absArchive {
		// gitDir is resolved (exists), so use it as the canonical base.
		resolvedGitDir, _ := filepath.EvalSymlinks(gitDir)
		if resolvedGitDir != "" {
			// Compute the suffix that was walked up, reattach to resolved base.
			suffix, _ := filepath.Rel(gitDir, absArchive)
			absArchive = filepath.Join(resolvedGitDir, suffix)
		}
	}

	// Compute repo-root-relative path for safe staging.
	relArchive, err := filepath.Rel(repoRoot, absArchive)
	if err != nil {
		return nil, fmt.Errorf("cannot compute relative path from %q to %q: %w", repoRoot, absArchive, err)
	}

	// Verify the relative path doesn't escape the repo (e.g., "../something").
	if strings.HasPrefix(relArchive, "..") {
		return nil, fmt.Errorf("archive root %q is outside git repository %q", absArchive, repoRoot)
	}

	// Reject archive_root == repoRoot: relArchive would be "." and
	// git add would stage the entire worktree (code, config, credentials).
	if relArchive == "." {
		return nil, fmt.Errorf("archive root %q is the git repository root; must be a dedicated subdirectory to prevent staging non-archive files", absArchive)
	}

	return &Client{
		archiveRoot: absArchive,
		repoRoot:    repoRoot,
		relArchive:  relArchive,
	}, nil
}

// HasChanges checks if there are any changes (staged or unstaged, tracked or
// untracked) under the archive root. Returns true if there are changes to commit.
func (c *Client) HasChanges() (bool, error) {
	out, err := runGit(c.repoRoot, "status", "--porcelain", c.relArchive)
	if err != nil {
		return false, fmt.Errorf("git status failed: %w", err)
	}
	return strings.TrimSpace(out) != "", nil
}

// DirtyLines returns git status --porcelain output for the archive root.
// Non-empty result means there are pre-existing changes that the current
// sync did not produce.
func (c *Client) DirtyLines() (string, error) {
	out, err := runGit(c.repoRoot, "status", "--porcelain", c.relArchive)
	if err != nil {
		return "", fmt.Errorf("git status failed: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// StageArchive stages all changes under the archive root.
// Uses the repo-root-relative path to prevent symlink escapes.
func (c *Client) StageArchive() error {
	_, err := runGit(c.repoRoot, "add", c.relArchive)
	if err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}
	return nil
}

// Commit creates a commit with the given message.
// Returns an error if there are no staged changes.
func (c *Client) Commit(message string) error {
	_, err := runGit(c.repoRoot, "commit", "-m", message)
	if err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}
	return nil
}

// Push pushes to the remote. If branch is non-empty, pushes the current HEAD
// to that remote branch (git push origin HEAD:<branch>). Otherwise pushes
// the current branch to its tracking remote.
func (c *Client) Push(branch string) error {
	var args []string
	if branch != "" {
		args = []string{"push", "origin", "HEAD:" + branch}
	} else {
		args = []string{"push"}
	}
	_, err := runGit(c.repoRoot, args...)
	if err != nil {
		return fmt.Errorf("git push failed: %w", err)
	}
	return nil
}

// DiffStat returns the --stat output of staged changes for use in
// commit message construction.
func (c *Client) DiffStat() (string, error) {
	out, err := runGit(c.repoRoot, "diff", "--cached", "--stat")
	if err != nil {
		return "", fmt.Errorf("git diff --stat failed: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// runGit executes a git command in the given directory and returns stdout.
func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %s", err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}
