package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initTestRepo creates a temporary git repository with an initial commit.
// Returns the repo root path.
func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@example.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "initial"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s failed: %s: %s", strings.Join(args, " "), err, out)
		}
	}
	return dir
}

func TestNewClient_GitNotOnPath(t *testing.T) {
	// Temporarily clear PATH to simulate git not found.
	origPath := os.Getenv("PATH")
	t.Setenv("PATH", "")
	defer os.Setenv("PATH", origPath)

	_, err := NewClient("/tmp/fake-archive")
	if err == nil {
		t.Fatal("expected error when git not on PATH")
	}
	if !strings.Contains(err.Error(), "git not found") {
		t.Errorf("expected 'git not found' error, got: %v", err)
	}
}

func TestNewClient_ArchiveOutsideRepo(t *testing.T) {
	outsideDir := t.TempDir()
	_, err := NewClient(outsideDir)
	if err == nil {
		t.Fatal("expected error for archive outside git repo")
	}
	if !strings.Contains(err.Error(), "not inside a git repository") {
		t.Errorf("expected 'not inside a git repository' error, got: %v", err)
	}
}

func TestNewClient_ArchiveInsideRepo(t *testing.T) {
	repo := initTestRepo(t)
	archiveDir := filepath.Join(repo, "archive")
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		t.Fatal(err)
	}

	c, err := NewClient(archiveDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.relArchive != "archive" {
		t.Errorf("relArchive = %q, want %q", c.relArchive, "archive")
	}
}

func TestHasChanges_NoChanges(t *testing.T) {
	repo := initTestRepo(t)
	archiveDir := filepath.Join(repo, "archive")
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		t.Fatal(err)
	}

	c, err := NewClient(archiveDir)
	if err != nil {
		t.Fatal(err)
	}

	has, err := c.HasChanges()
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Error("expected no changes in empty archive dir")
	}
}

func TestHasChanges_WithNewFile(t *testing.T) {
	repo := initTestRepo(t)
	archiveDir := filepath.Join(repo, "archive")
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a file to the archive dir.
	if err := os.WriteFile(filepath.Join(archiveDir, "test.md"), []byte("# Test"), 0o644); err != nil {
		t.Fatal(err)
	}

	c, err := NewClient(archiveDir)
	if err != nil {
		t.Fatal(err)
	}

	has, err := c.HasChanges()
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Error("expected changes after adding file")
	}
}

func TestStageAndCommit(t *testing.T) {
	repo := initTestRepo(t)
	archiveDir := filepath.Join(repo, "archive", "llm-api", "xai")
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write test files.
	if err := os.WriteFile(filepath.Join(archiveDir, "llms.txt"), []byte("# xAI docs"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Also write a non-archive file that should NOT be staged.
	if err := os.WriteFile(filepath.Join(repo, "config.yaml"), []byte("key: val"), 0o644); err != nil {
		t.Fatal(err)
	}

	c, err := NewClient(filepath.Join(repo, "archive"))
	if err != nil {
		t.Fatal(err)
	}

	// Stage archive only.
	if err := c.StageArchive(); err != nil {
		t.Fatal(err)
	}

	// Commit.
	if err := c.Commit("test commit"); err != nil {
		t.Fatal(err)
	}

	// Verify: config.yaml should still be untracked.
	out, err := runGit(repo, "status", "--porcelain")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "config.yaml") {
		t.Error("expected config.yaml to remain untracked")
	}
	if strings.Contains(out, "llms.txt") {
		t.Error("expected llms.txt to be committed (no longer in status)")
	}
}

func TestCommit_NothingStaged(t *testing.T) {
	repo := initTestRepo(t)
	archiveDir := filepath.Join(repo, "archive")
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		t.Fatal(err)
	}

	c, err := NewClient(archiveDir)
	if err != nil {
		t.Fatal(err)
	}

	// Commit with nothing staged should fail.
	err = c.Commit("empty commit")
	if err == nil {
		t.Fatal("expected error when nothing to commit")
	}
}

func TestPush_NoRemote(t *testing.T) {
	repo := initTestRepo(t)
	archiveDir := filepath.Join(repo, "archive")
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		t.Fatal(err)
	}

	c, err := NewClient(archiveDir)
	if err != nil {
		t.Fatal(err)
	}

	// Push without remote should fail with clear error.
	err = c.Push("")
	if err == nil {
		t.Fatal("expected error when pushing without remote")
	}
	if !strings.Contains(err.Error(), "git push failed") {
		t.Errorf("expected 'git push failed' error, got: %v", err)
	}
}

func TestPush_WithBranch(t *testing.T) {
	repo := initTestRepo(t)
	archiveDir := filepath.Join(repo, "archive")
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		t.Fatal(err)
	}

	c, err := NewClient(archiveDir)
	if err != nil {
		t.Fatal(err)
	}

	// Push to named branch without remote should fail, but with the right refspec.
	err = c.Push("archive/daily")
	if err == nil {
		t.Fatal("expected error when pushing without remote")
	}
	if !strings.Contains(err.Error(), "git push failed") {
		t.Errorf("expected 'git push failed' error, got: %v", err)
	}
}
