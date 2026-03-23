package git

import (
	"strings"
	"testing"
	"time"
)

func TestBuildCommitMessage_Basic(t *testing.T) {
	results := []SyncResult{
		{TopicSlug: "llm-api", ProviderSlug: "xai", FilesWritten: 96},
		{TopicSlug: "llm-api", ProviderSlug: "anthropic", FilesWritten: 488},
		{TopicSlug: "cloud-infra", ProviderSlug: "aws-glue-dg", FilesWritten: 1},
	}
	msg := BuildCommitMessage(results, "/data/archive", nil)

	today := time.Now().Format("2006-01-02")
	if !strings.Contains(msg, "refbolt sync: "+today) {
		t.Errorf("missing date header, got:\n%s", msg)
	}
	if !strings.Contains(msg, "- xai: 96 files (llm-api)") {
		t.Errorf("missing xai entry, got:\n%s", msg)
	}
	if !strings.Contains(msg, "- anthropic: 488 files (llm-api)") {
		t.Errorf("missing anthropic entry, got:\n%s", msg)
	}
	if !strings.Contains(msg, "- aws-glue-dg: 1 file (cloud-infra)") {
		t.Errorf("missing aws-glue-dg entry (should use singular 'file'), got:\n%s", msg)
	}
	if !strings.Contains(msg, "Archive root: /data/archive") {
		t.Errorf("missing archive root, got:\n%s", msg)
	}
}

func TestBuildCommitMessage_WithTrailers(t *testing.T) {
	results := []SyncResult{
		{TopicSlug: "llm-api", ProviderSlug: "xai", FilesWritten: 10},
	}
	trailers := []string{
		"Co-Authored-By: Claude Opus 4.6 <noreply@fulmenhq.dev>",
		"Signed-off-by: Bot <bot@example.com>",
	}
	msg := BuildCommitMessage(results, "/data/archive", trailers)

	if !strings.Contains(msg, "Co-Authored-By: Claude Opus 4.6 <noreply@fulmenhq.dev>") {
		t.Errorf("missing Co-Authored-By trailer, got:\n%s", msg)
	}
	if !strings.Contains(msg, "Signed-off-by: Bot <bot@example.com>") {
		t.Errorf("missing Signed-off-by trailer, got:\n%s", msg)
	}
}

func TestBuildCommitMessage_NoResults(t *testing.T) {
	msg := BuildCommitMessage(nil, "/data/archive", nil)

	if !strings.Contains(msg, "refbolt sync:") {
		t.Errorf("missing header, got:\n%s", msg)
	}
	// Should not contain "Providers updated:" section.
	if strings.Contains(msg, "Providers updated:") {
		t.Errorf("should not have providers section with no results, got:\n%s", msg)
	}
	if !strings.Contains(msg, "Archive root: /data/archive") {
		t.Errorf("missing archive root, got:\n%s", msg)
	}
}

func TestBuildCommitMessage_SingularFile(t *testing.T) {
	results := []SyncResult{
		{TopicSlug: "data-platform", ProviderSlug: "trino", FilesWritten: 1},
	}
	msg := BuildCommitMessage(results, "/archive", nil)

	if !strings.Contains(msg, "1 file") {
		t.Errorf("expected singular 'file', got:\n%s", msg)
	}
	if strings.Contains(msg, "1 files") {
		t.Errorf("should not have plural 'files' for count 1, got:\n%s", msg)
	}
}
