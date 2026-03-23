package git

import (
	"fmt"
	"strings"
	"time"
)

// SyncResult captures what was synced for a single provider.
// Populated by the sync command and passed to BuildCommitMessage.
type SyncResult struct {
	TopicSlug    string
	ProviderSlug string
	FilesWritten int
}

// BuildCommitMessage constructs a structured commit message from sync results.
//
// Format:
//
//	refbolt sync: 2026-03-22
//
//	Providers updated:
//	- xai: 96 files (llm-api)
//	- anthropic: 488 files (llm-api)
//
//	Archive root: /data/archive
//
// Trailers (if any) are appended after a blank line.
func BuildCommitMessage(results []SyncResult, archiveRoot string, trailers []string) string {
	date := time.Now().Format("2006-01-02")

	var b strings.Builder
	fmt.Fprintf(&b, "refbolt sync: %s\n", date)

	if len(results) > 0 {
		b.WriteString("\nProviders updated:\n")
		for _, r := range results {
			word := "files"
			if r.FilesWritten == 1 {
				word = "file"
			}
			fmt.Fprintf(&b, "- %s: %d %s (%s)\n", r.ProviderSlug, r.FilesWritten, word, r.TopicSlug)
		}
	}

	fmt.Fprintf(&b, "\nArchive root: %s\n", archiveRoot)

	if len(trailers) > 0 {
		b.WriteString("\n")
		for _, t := range trailers {
			b.WriteString(t)
			b.WriteString("\n")
		}
	}

	return b.String()
}
