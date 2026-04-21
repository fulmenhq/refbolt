package config

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
)

// RegistryEntry holds the subset of `registry/providers.jsonl` fields that
// FA-101 surfaces through `refbolt catalog` output. Extra fields present in
// the JSONL (tags, notes, last_verified, tos_reviewed, capabilities, etc.)
// are intentionally ignored here — they are deferred to FA-108.
type RegistryEntry struct {
	Slug           string `json:"slug"`
	Description    string `json:"description"`
	EstimatedPages int    `json:"estimated_pages"`
}

// ParseRegistry decodes a JSONL byte stream into a slug-indexed map.
// Blank lines and leading/trailing whitespace are tolerated. A malformed
// line returns an error that identifies the 1-based line number so the
// caller can pinpoint bad data during development.
func ParseRegistry(data []byte) (map[string]RegistryEntry, error) {
	out := make(map[string]RegistryEntry)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	// Registry lines can be long (notes, URLs, etc.). Default scanner buffer
	// is 64KB which is plenty, but be explicit to avoid future surprises.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var entry RegistryEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			return nil, fmt.Errorf("registry: line %d: %w", lineNum, err)
		}
		if entry.Slug == "" {
			return nil, fmt.Errorf("registry: line %d: missing slug", lineNum)
		}
		out[entry.Slug] = entry
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("registry: scanning: %w", err)
	}
	return out, nil
}

// LoadRegistry parses the embedded registry bytes. Returns an empty map (not
// an error) when the registry has not been injected — catalog commands are
// expected to degrade gracefully without registry enrichment, so callers
// can treat "empty registry" and "embedded registry wasn't set" identically.
func LoadRegistry() (map[string]RegistryEntry, error) {
	if len(embeddedRegistry) == 0 {
		return map[string]RegistryEntry{}, nil
	}
	return ParseRegistry(embeddedRegistry)
}
