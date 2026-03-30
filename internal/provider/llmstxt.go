package provider

import (
	"bufio"
	"bytes"
	"fmt"
	"net/url"
	"strings"
)

// SplitLLMSTxt parses the xAI-style llms.txt format into individual pages.
// The format uses `===/<path>===` delimiters between sections.
// Each section becomes a Page with the path derived from the delimiter.
//
// Example input:
//
//	===/overview===
//	# Welcome
//	Some content here.
//
//	===/developers/tools/overview===
//	# Tools Overview
//	More content.
func SplitLLMSTxt(content []byte, sourceURL string) ([]Page, error) {
	var pages []Page
	var currentPath string
	var currentContent bytes.Buffer

	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "===/") && strings.HasSuffix(line, "===") {
			// Flush the previous section.
			if currentPath != "" {
				pages = append(pages, Page{
					SourceURL: sourceURL,
					Path:      pathToArchivePath(currentPath),
					Content:   copyBytes(bytes.TrimSpace(currentContent.Bytes())),
				})
			}

			// Start a new section.
			currentPath = strings.TrimPrefix(line, "===")
			currentPath = strings.TrimSuffix(currentPath, "===")
			currentContent.Reset()
			continue
		}

		if currentPath != "" {
			currentContent.WriteString(line)
			currentContent.WriteByte('\n')
		}
	}
	if err := scanner.Err(); err != nil {
		return pages, fmt.Errorf("scanning llms.txt: %w", err)
	}

	// Flush the last section.
	if currentPath != "" {
		pages = append(pages, Page{
			SourceURL: sourceURL,
			Path:      pathToArchivePath(currentPath),
			Content:   copyBytes(bytes.TrimSpace(currentContent.Bytes())),
		})
	}

	return pages, nil
}

// SplitLLMSFullTxt parses llms-full.txt files that use URL-based section delimiters.
// This format is used by Anthropic (platform.claude.com), DigitalOcean, and other
// providers that publish full-content dumps with section URL markers.
//
// Recognized URL line prefixes:
//   - "URL: <url>"    (Anthropic format)
//   - "Source: <url>"  (DigitalOcean format)
//
// The section boundary pattern is:
//
//	---
//
//	# Page Title
//
//	URL: https://platform.claude.com/docs/en/some/path
//
//	# Page Title  (duplicate — stripped from content)
//
//	<page content...>
//
// The URL line is the reliable split point. The preceding --- and # Title are
// inter-section preamble. The duplicate # Title after the URL is stripped.
func SplitLLMSFullTxt(content []byte, sourceURL string) ([]Page, error) {
	var pages []Page
	var currentURL string
	var currentContent bytes.Buffer
	var skipNextHeading bool

	scanner := bufio.NewScanner(bytes.NewReader(content))
	// Large llms-full.txt files (Anthropic ~24MB, DO ~40MB); increase scanner buffer.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if sectionURL, ok := parseSectionURL(line); ok {
			// Flush the previous section.
			if currentURL != "" {
				pages = append(pages, makeLLMSFullPage(currentURL, sourceURL, &currentContent))
			}

			currentURL = sectionURL
			currentContent.Reset()
			skipNextHeading = true
			continue
		}

		if currentURL == "" {
			// Before the first URL line — skip file preamble.
			continue
		}

		// Strip the duplicate # Title that immediately follows the URL line.
		if skipNextHeading {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				// Blank lines between URL and the heading — skip.
				continue
			}
			if strings.HasPrefix(trimmed, "# ") {
				skipNextHeading = false
				continue
			}
			// Non-blank, non-heading line — stop skipping, include it.
			skipNextHeading = false
		}

		// Trim trailing preamble: if we see a "---" line that precedes the
		// next section's "# Title" + "URL:", it belongs to the boundary,
		// not to this section's content. We handle this by trimming trailing
		// "---" and blank lines when flushing (in makeLLMSFullPage).
		currentContent.WriteString(line)
		currentContent.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		return pages, fmt.Errorf("scanning llms-full.txt: %w", err)
	}

	// Flush the last section.
	if currentURL != "" {
		pages = append(pages, makeLLMSFullPage(currentURL, sourceURL, &currentContent))
	}

	return pages, nil
}

// SplitFrontmatterFullTxt parses llms-full.txt files that use YAML frontmatter
// as section delimiters. This format is used by Cloudflare and potentially other
// providers that generate rendered Markdown dumps with frontmatter boundaries.
//
// The section boundary pattern is:
//
//	---
//	title: Page Title
//	description: ...
//	image: ...
//	---
//
//	[Skip to content]  (boilerplate — stripped)
//	Was this helpful?  (boilerplate — stripped)
//	[ Edit page ](https://github.com/...) (boilerplate — URL extracted for archive path)
//	Copy page          (boilerplate — stripped)
//
//	# Page Title       (content starts here)
//
// The frontmatter `---` block with `title:` is the section boundary.
// Archive paths are derived from the `[ Edit page ]` GitHub URL if present,
// or from the frontmatter title as a fallback.
func SplitFrontmatterFullTxt(content []byte, sourceURL string) ([]Page, error) {
	var pages []Page
	var sections []frontmatterSection

	scanner := bufio.NewScanner(bytes.NewReader(content))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var (
		inFrontmatter bool
		currentTitle  string // title of the section being collected
		pendingTitle  string // title being parsed in current frontmatter block
		rawContent    bytes.Buffer
		collecting    bool
		foundTitle    bool
	)

	flushSection := func() {
		if currentTitle != "" && rawContent.Len() > 0 {
			sections = append(sections, frontmatterSection{
				title:   currentTitle,
				content: copyBytes(rawContent.Bytes()),
			})
		}
		currentTitle = ""
		rawContent.Reset()
		collecting = false
	}

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Detect frontmatter start: "---" when not already in frontmatter.
		if trimmed == "---" && !inFrontmatter {
			inFrontmatter = true
			pendingTitle = ""
			foundTitle = false
			continue
		}

		if inFrontmatter {
			if trimmed == "---" {
				if foundTitle {
					// End of a real frontmatter block (had title:).
					// Flush the previous section, start new one.
					if collecting {
						flushSection()
					}
					currentTitle = pendingTitle
					inFrontmatter = false
					collecting = true
					continue
				}
				// False start: "---" followed by "---" without title:.
				// Stay scanning in case the next line starts frontmatter.
				continue
			}
			// Extract title from frontmatter.
			if strings.HasPrefix(trimmed, "title:") {
				pendingTitle = strings.TrimSpace(strings.TrimPrefix(trimmed, "title:"))
				foundTitle = true
				continue
			}
			// Blank lines in potential frontmatter — skip.
			if trimmed == "" {
				continue
			}
			// Other frontmatter line (description:, image:, etc.) — skip.
			if foundTitle {
				continue
			}
			// Non-blank, non-title, non---- line without title found.
			// This was a false start (the --- was just a HR).
			inFrontmatter = false
			// Fall through to content collection.
		}

		if !collecting {
			continue
		}

		// Strip Cloudflare boilerplate between frontmatter and content.
		if isBoilerplateLine(trimmed) {
			continue
		}

		rawContent.WriteString(line)
		rawContent.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		return pages, fmt.Errorf("scanning frontmatter llms-full.txt: %w", err)
	}

	// Flush last section.
	flushSection()

	// Convert sections to pages.
	for _, s := range sections {
		archivePath := titleToArchivePath(s.title)
		cleaned := trimTrailingBoundary(s.content)
		// Also strip any leading blank lines.
		cleaned = bytes.TrimLeft(cleaned, "\n\r\t ")

		pages = append(pages, Page{
			SourceURL: sourceURL,
			Path:      archivePath,
			Content:   cleaned,
		})
	}

	return pages, nil
}

// frontmatterSection holds a parsed section from a frontmatter-delimited file.
type frontmatterSection struct {
	title   string
	content []byte
}

// isBoilerplateLine returns true for Cloudflare-style boilerplate lines that
// appear between the YAML frontmatter and the actual page content.
func isBoilerplateLine(line string) bool {
	if line == "" {
		return false // blank lines are not boilerplate — they're formatting
	}
	boilerplate := []string{
		"[Skip to content]",
		"Was this helpful?",
		"YesNo",
		"Copy page",
	}
	for _, b := range boilerplate {
		if strings.Contains(line, b) {
			return true
		}
	}
	// "[ Edit page ]" and "[ Report issue ]" links.
	if strings.HasPrefix(line, "[ Edit page ]") || strings.HasPrefix(line, "[ Report issue ]") {
		return true
	}
	return false
}

// titleToArchivePath converts a page title to a filesystem path.
// "Getting started" → "getting-started.md"
// "Workers KV API" → "workers-kv-api.md"
func titleToArchivePath(title string) string {
	s := strings.ToLower(title)
	s = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		if r == ' ' || r == '_' || r == '/' {
			return '-'
		}
		return -1 // drop
	}, s)
	// Collapse multiple hyphens.
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	s = strings.Trim(s, "-")
	if s == "" {
		s = "index"
	}
	return s + ".md"
}

// makeLLMSFullPage creates a Page from a URL-delimited section.
// It derives the archive path from the page URL and trims trailing
// boundary markers (--- lines and blank lines) from the content.
func makeLLMSFullPage(pageURL, sourceURL string, buf *bytes.Buffer) Page {
	archivePath := llmsFullURLToPath(pageURL)
	// Trim trailing boundary: content may end with "\n---\n\n# Next Title\n"
	// but since we stop collecting at the URL line, we only have trailing
	// "---" and blank lines to clean up.
	raw := buf.Bytes()
	cleaned := trimTrailingBoundary(raw)

	return Page{
		SourceURL: pageURL,
		Path:      archivePath,
		Content:   copyBytes(cleaned),
	}
}

// llmsFullURLToPath extracts an archive path from a full page URL.
// It strips the scheme+host and leading /docs/ prefix, then converts
// via pathToArchivePath.
//
// Example:
//
//	"https://platform.claude.com/docs/en/build-with-claude/tool-use"
//	→ "en/build-with-claude/tool-use.md"
func llmsFullURLToPath(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return pathToArchivePath(rawURL)
	}
	p := u.Path
	// Strip common docs prefixes.
	p = strings.TrimPrefix(p, "/docs/")
	p = strings.TrimPrefix(p, "/docs")
	if p == "" || p == "/" {
		return "index.md"
	}
	return pathToArchivePath(p)
}

// parseSectionURL extracts the URL from a section delimiter line.
// Supports "URL: <url>" (Anthropic) and "Source: <url>" (DigitalOcean).
func parseSectionURL(line string) (string, bool) {
	for _, prefix := range []string{"URL: ", "Source: "} {
		if strings.HasPrefix(line, prefix) {
			u := strings.TrimSpace(strings.TrimPrefix(line, prefix))
			if u != "" {
				return u, true
			}
		}
	}
	return "", false
}

// FilterByBaseURL filters pages to only those whose SourceURL starts with
// the given baseURL prefix. When baseURL has no path beyond the domain root
// (e.g., "https://platform.claude.com" or "https://platform.claude.com/"),
// all pages pass through — this preserves backwards compatibility with
// providers like Anthropic that don't scope by URL prefix.
//
// This enables scoped provider entries where multiple providers share the
// same llms-full.txt but each archives only its URL prefix (e.g., DO API
// Reference at docs.digitalocean.com/reference/api).
func FilterByBaseURL(pages []Page, baseURL string) []Page {
	if baseURL == "" {
		return pages
	}

	// Parse the base URL to check if it has a meaningful path.
	u, err := url.Parse(baseURL)
	if err != nil {
		return pages
	}
	path := strings.TrimRight(u.Path, "/")
	if path == "" {
		// Domain-only base URL → no filtering (backwards-compat).
		return pages
	}

	// Normalize: ensure baseURL prefix ends without trailing slash for matching.
	prefix := strings.TrimRight(baseURL, "/")

	var filtered []Page
	for _, p := range pages {
		sourceNorm := strings.TrimRight(p.SourceURL, "/")
		if strings.HasPrefix(sourceNorm, prefix) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// copyBytes returns a copy of b that doesn't share the underlying array.
// This is necessary when the source is a bytes.Buffer that will be Reset.
func copyBytes(b []byte) []byte {
	if b == nil {
		return nil
	}
	return append([]byte(nil), b...)
}

// trimTrailingBoundary removes a trailing inter-section boundary from content.
// The boundary pattern is: ...\n---\n\n# Next Title\n — we only strip this
// when the heading is actually paired with a preceding --- separator, so a
// page that legitimately ends with a # heading is left intact.
func trimTrailingBoundary(content []byte) []byte {
	s := bytes.TrimRight(content, " \t\n")

	// Check for the combined pattern: "---" followed by "# Title" at the end.
	// Only strip the heading if it sits directly after a --- separator.
	if idx := bytes.LastIndex(s, []byte("\n")); idx >= 0 {
		lastLine := bytes.TrimSpace(s[idx+1:])
		before := bytes.TrimRight(s[:idx], " \t\n")
		if bytes.HasPrefix(lastLine, []byte("# ")) && bytes.HasSuffix(before, []byte("---")) {
			// Strip both the heading and the --- separator.
			s = bytes.TrimSuffix(before, []byte("---"))
			s = bytes.TrimRight(s, " \t\n")
			return s
		}
	}

	// No heading+--- pair, but still strip a bare trailing --- separator.
	if bytes.HasSuffix(s, []byte("---")) {
		candidate := bytes.TrimSuffix(s, []byte("---"))
		candidate = bytes.TrimRight(candidate, " \t\n")
		// Only strip if the --- is on its own line (not part of content like "foo---").
		if len(candidate) == 0 || candidate[len(candidate)-1] == '\n' {
			s = candidate
		}
	}

	return s
}
