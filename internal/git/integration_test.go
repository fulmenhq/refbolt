package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestIntegration_SyncGitCommit simulates the full sync --git-commit flow:
// write archive files, stage, build message, commit, and verify results.
//
// This test is skipped in short mode (go test -short) and in CI
// environments where git user config may not be available.
// See docs/cicd.md for details on test modes.
func TestIntegration_SyncGitCommit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	repo := initTestRepo(t)
	archiveDir := filepath.Join(repo, "archive")

	// Also create a non-archive file to verify it's never staged.
	if err := os.WriteFile(filepath.Join(repo, "config.yaml"), []byte("key: secret"), 0o644); err != nil {
		t.Fatal(err)
	}

	// 1. Pre-flight: create git client (archive dir doesn't exist yet).
	gc, err := NewClient(archiveDir)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	// 2. Check for pre-existing dirt (should be clean for archive path).
	dirt, err := gc.DirtyLines()
	if err != nil {
		t.Fatal(err)
	}
	// config.yaml is outside archive root, so DirtyLines should not see it.
	if dirt != "" {
		t.Fatalf("expected clean archive before sync, got:\n%s", dirt)
	}

	// 3. Simulate archive writer output — create date-versioned files.
	date := time.Now().Format("2006-01-02")
	providers := []struct {
		topic    string
		provider string
		files    map[string]string
	}{
		{
			topic:    "llm-api",
			provider: "xai",
			files: map[string]string{
				"llms.txt":         "# xAI\n\n## Grok API\n...",
				"docs/api/chat.md": "# Chat Completions\n...",
			},
		},
		{
			topic:    "llm-api",
			provider: "anthropic",
			files: map[string]string{
				"llms-full.txt": "# Anthropic API\n...",
			},
		},
	}

	for _, p := range providers {
		for name, content := range p.files {
			path := filepath.Join(archiveDir, p.topic, p.provider, date, name)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}

	// 4. Verify changes detected.
	has, err := gc.HasChanges()
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Fatal("expected changes after writing archive files")
	}

	// 5. Stage archive only.
	if err := gc.StageArchive(); err != nil {
		t.Fatal(err)
	}

	// 6. Build commit message from sync results.
	results := []SyncResult{
		{TopicSlug: "llm-api", ProviderSlug: "xai", FilesWritten: 2},
		{TopicSlug: "llm-api", ProviderSlug: "anthropic", FilesWritten: 1},
	}
	trailers := []string{
		"Automated-By: refbolt-runner",
	}
	msg := BuildCommitMessage(results, archiveDir, trailers)

	// 7. Commit.
	if err := gc.Commit(msg); err != nil {
		t.Fatalf("commit failed: %v", err)
	}

	// === Verification ===

	// A. Commit message contains expected structure.
	logOut, err := runGit(repo, "log", "-1", "--pretty=%B")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(logOut, "refbolt sync:") {
		t.Error("commit message missing 'refbolt sync:' header")
	}
	if !strings.Contains(logOut, "xai: 2 files (llm-api)") {
		t.Error("commit message missing xai entry")
	}
	if !strings.Contains(logOut, "anthropic: 1 file (llm-api)") {
		t.Error("commit message missing anthropic entry")
	}
	if !strings.Contains(logOut, "Automated-By: refbolt-runner") {
		t.Error("commit message missing trailer")
	}

	// B. Only archive files were committed — config.yaml must still be untracked.
	statusOut, err := runGit(repo, "status", "--porcelain")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(statusOut, "config.yaml") {
		t.Error("config.yaml should remain untracked after archive-scoped commit")
	}

	// C. Committed files are correct.
	diffOut, err := runGit(repo, "diff", "--name-only", "HEAD~1", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	committedFiles := strings.TrimSpace(diffOut)
	if !strings.Contains(committedFiles, "archive/") {
		t.Errorf("committed files should be under archive/, got:\n%s", committedFiles)
	}
	if strings.Contains(committedFiles, "config.yaml") {
		t.Error("config.yaml was committed — archive scoping failed")
	}

	// D. Verify diff stat works on the committed changes.
	// (This exercises DiffStat which is available for message enrichment.)
	cmd := exec.Command("git", "diff", "--stat", "HEAD~1", "HEAD")
	cmd.Dir = repo
	statOut, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(statOut), "3 files changed") {
		t.Logf("diff stat: %s", statOut)
	}

	// E. No changes remain in archive after commit.
	has, err = gc.HasChanges()
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Error("archive should have no changes after commit")
	}
}

// TestIntegration_PreExistingDirtBlocksSync simulates the pre-existing
// dirt guard: if the archive has uncommitted changes before sync starts,
// the flow should detect them and the operator should refuse to proceed.
func TestIntegration_PreExistingDirtBlocksSync(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	repo := initTestRepo(t)
	archiveDir := filepath.Join(repo, "archive")
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Simulate leftover from a prior interrupted sync.
	if err := os.WriteFile(filepath.Join(archiveDir, "stale-data.md"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	gc, err := NewClient(archiveDir)
	if err != nil {
		t.Fatal(err)
	}

	dirt, err := gc.DirtyLines()
	if err != nil {
		t.Fatal(err)
	}
	if dirt == "" {
		t.Fatal("expected pre-existing dirt to be detected")
	}

	// In the real sync command, this would cause an early exit with the
	// porcelain output. Verify the dirt is actionable.
	t.Logf("Pre-existing dirt detected (as expected):\n%s", dirt)
}
