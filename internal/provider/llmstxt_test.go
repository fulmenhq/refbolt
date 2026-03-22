package provider

import (
	"strings"
	"testing"
)

func TestSplitLLMSTxt_XAIFormat(t *testing.T) {
	content := []byte(`Some preamble text that should be ignored.

===/overview===
# Welcome
Some overview content here.

===/developers/tools/overview===
# Tools Overview
More content about tools.

===/developers/models===
# Models
Model documentation.
`)

	pages, err := SplitLLMSTxt(content, "https://docs.x.ai/llms.txt")
	if err != nil {
		t.Fatal(err)
	}

	if len(pages) != 3 {
		t.Fatalf("Expected 3 pages, got %d", len(pages))
	}

	tests := []struct {
		idx  int
		path string
		snip string
	}{
		{0, "overview.md", "# Welcome"},
		{1, "developers/tools/overview.md", "# Tools Overview"},
		{2, "developers/models.md", "# Models"},
	}
	for _, tt := range tests {
		if pages[tt.idx].Path != tt.path {
			t.Errorf("page[%d].Path = %q, want %q", tt.idx, pages[tt.idx].Path, tt.path)
		}
		if got := string(pages[tt.idx].Content); !strings.Contains(got, tt.snip) {
			t.Errorf("page[%d].Content missing %q", tt.idx, tt.snip)
		}
	}
}

func TestSplitLLMSTxt_NoDelimiters(t *testing.T) {
	content := []byte("# Just a plain Markdown file\n\nNo delimiters here.\n")
	pages, err := SplitLLMSTxt(content, "https://example.com/llms.txt")
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 0 {
		t.Errorf("Expected 0 pages for content without delimiters, got %d", len(pages))
	}
}

func TestSplitLLMSFullTxt_AnthropicFormat(t *testing.T) {
	content := []byte(`# Anthropic Developer Documentation - Full Content

Some preamble about the documentation.

---

# English Documentation - Full Content

## Developer Guide

---

# Get Started

URL: https://platform.claude.com/docs/en/get-started

# Get Started

Make your first API call to Claude.

## Prerequisites

- An API key

---

# Tool Use

URL: https://platform.claude.com/docs/en/agents-and-tools/tool-use/overview

# Tool Use

Claude can use tools to interact with external systems.

### Example

` + "```python\n" + `response = client.messages.create(tools=[...])
` + "```\n" + `
---

# Prompt Caching

URL: https://platform.claude.com/docs/en/build-with-claude/prompt-caching

# Prompt Caching

Cache frequently used prompts for faster responses.
`)

	pages, err := SplitLLMSFullTxt(content, "https://platform.claude.com/llms-full.txt")
	if err != nil {
		t.Fatal(err)
	}

	if len(pages) != 3 {
		t.Fatalf("Expected 3 pages, got %d", len(pages))
	}

	tests := []struct {
		idx      int
		path     string
		wantSnip string
		noSnip   string
	}{
		{0, "en/get-started.md", "Make your first API call", "# Get Started"},
		{1, "en/agents-and-tools/tool-use/overview.md", "Claude can use tools", "# Tool Use"},
		{2, "en/build-with-claude/prompt-caching.md", "Cache frequently used prompts", "# Prompt Caching"},
	}
	for _, tt := range tests {
		if pages[tt.idx].Path != tt.path {
			t.Errorf("page[%d].Path = %q, want %q", tt.idx, pages[tt.idx].Path, tt.path)
		}
		got := string(pages[tt.idx].Content)
		if !strings.Contains(got, tt.wantSnip) {
			t.Errorf("page[%d].Content missing %q", tt.idx, tt.wantSnip)
		}
		// Verify the duplicate heading was stripped.
		if strings.Contains(got[:min(len(got), 50)], tt.noSnip) {
			t.Errorf("page[%d].Content starts with duplicate heading %q — should be stripped", tt.idx, tt.noSnip)
		}
	}

	// Verify content doesn't include trailing --- boundary.
	for i, p := range pages {
		got := string(p.Content)
		if len(got) > 3 && got[len(got)-3:] == "---" {
			t.Errorf("page[%d].Content ends with trailing ---", i)
		}
	}
}

func TestSplitLLMSFullTxt_NoURLLines(t *testing.T) {
	content := []byte("# Just a plain Markdown file\n\nNo URL lines here.\n")
	pages, err := SplitLLMSFullTxt(content, "https://example.com/llms-full.txt")
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 0 {
		t.Errorf("Expected 0 pages for content without URL lines, got %d", len(pages))
	}
}

func TestSplitLLMSFullTxt_PreservesCodeBlocks(t *testing.T) {
	content := []byte(`---

# API Reference

URL: https://platform.claude.com/docs/en/api-reference/messages

# API Reference

Create a message:

` + "```json\n" + `{
  "model": "claude-sonnet-4-5",
  "messages": [{"role": "user", "content": "Hello"}]
}
` + "```\n")

	pages, err := SplitLLMSFullTxt(content, "https://platform.claude.com/llms-full.txt")
	if err != nil {
		t.Fatal(err)
	}

	if len(pages) != 1 {
		t.Fatalf("Expected 1 page, got %d", len(pages))
	}
	got := string(pages[0].Content)
	if !strings.Contains(got, `"claude-sonnet-4-5"`) {
		t.Error("Code block content not preserved")
	}
}

func TestSplitLLMSFullTxt_PreservesTrailingHeading(t *testing.T) {
	// A page that legitimately ends with a # heading should NOT have it stripped.
	content := []byte(`---

# My Page

URL: https://example.com/docs/en/my-page

# My Page

Some content.

# Appendix
`)

	pages, err := SplitLLMSFullTxt(content, "https://example.com/llms-full.txt")
	if err != nil {
		t.Fatal(err)
	}

	if len(pages) != 1 {
		t.Fatalf("Expected 1 page, got %d", len(pages))
	}
	got := string(pages[0].Content)
	if !strings.Contains(got, "# Appendix") {
		t.Errorf("Trailing heading '# Appendix' was incorrectly stripped from content: %q", got)
	}
}

func TestLLMSFullURLToPath(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://platform.claude.com/docs/en/get-started", "en/get-started.md"},
		{"https://platform.claude.com/docs/en/build-with-claude/tool-use", "en/build-with-claude/tool-use.md"},
		{"https://platform.claude.com/docs/en/api-reference/messages.md", "en/api-reference/messages.md"},
		{"https://example.com/docs/page", "page.md"},
	}
	for _, tt := range tests {
		got := llmsFullURLToPath(tt.url)
		if got != tt.want {
			t.Errorf("llmsFullURLToPath(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}
