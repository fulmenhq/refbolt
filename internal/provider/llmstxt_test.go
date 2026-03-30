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

func TestSplitLLMSFullTxt_SourcePrefix(t *testing.T) {
	content := []byte(`# DigitalOcean Documentation - Complete

> Full text of all DigitalOcean documentation pages.

---

Source: https://docs.digitalocean.com/reference/api/create-token/

# How to Create a Personal Access Token

Create tokens from the API section of the control panel.

---

Source: https://docs.digitalocean.com/products/kubernetes/getting-started/

# Getting Started with Kubernetes

Deploy your first cluster.
`)

	pages, err := SplitLLMSFullTxt(content, "https://docs.digitalocean.com/llms-full.txt")
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(pages))
	}
	if pages[0].SourceURL != "https://docs.digitalocean.com/reference/api/create-token/" {
		t.Errorf("page[0] SourceURL = %q", pages[0].SourceURL)
	}
	if pages[1].SourceURL != "https://docs.digitalocean.com/products/kubernetes/getting-started/" {
		t.Errorf("page[1] SourceURL = %q", pages[1].SourceURL)
	}
	if !strings.Contains(string(pages[0].Content), "Create tokens") {
		t.Errorf("page[0] missing expected content")
	}
}

func TestFilterByBaseURL_Scoped(t *testing.T) {
	pages := []Page{
		{SourceURL: "https://docs.digitalocean.com/reference/api/create-token/", Path: "a.md"},
		{SourceURL: "https://docs.digitalocean.com/reference/api/list-droplets/", Path: "b.md"},
		{SourceURL: "https://docs.digitalocean.com/products/kubernetes/getting-started/", Path: "c.md"},
		{SourceURL: "https://docs.digitalocean.com/products/spaces/overview/", Path: "d.md"},
	}

	filtered := FilterByBaseURL(pages, "https://docs.digitalocean.com/reference/api")
	if len(filtered) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(filtered))
	}
	if filtered[0].Path != "a.md" || filtered[1].Path != "b.md" {
		t.Errorf("unexpected pages: %v", filtered)
	}
}

func TestFilterByBaseURL_DomainOnly_PassesAll(t *testing.T) {
	pages := []Page{
		{SourceURL: "https://platform.claude.com/docs/en/get-started", Path: "a.md"},
		{SourceURL: "https://platform.claude.com/docs/en/tool-use", Path: "b.md"},
	}

	// Domain-only base URL → no filtering (backwards-compat with Anthropic).
	filtered := FilterByBaseURL(pages, "https://platform.claude.com")
	if len(filtered) != 2 {
		t.Fatalf("expected all 2 pages to pass, got %d", len(filtered))
	}
}

func TestFilterByBaseURL_EmptyBaseURL(t *testing.T) {
	pages := []Page{
		{SourceURL: "https://example.com/a", Path: "a.md"},
	}
	filtered := FilterByBaseURL(pages, "")
	if len(filtered) != 1 {
		t.Fatalf("expected 1 page, got %d", len(filtered))
	}
}

func TestFilterByBaseURL_NoMatch(t *testing.T) {
	pages := []Page{
		{SourceURL: "https://docs.digitalocean.com/products/spaces/overview/", Path: "a.md"},
	}
	filtered := FilterByBaseURL(pages, "https://docs.digitalocean.com/reference/api")
	if len(filtered) != 0 {
		t.Fatalf("expected 0 pages, got %d", len(filtered))
	}
}

func TestParseSectionURL(t *testing.T) {
	tests := []struct {
		line string
		want string
		ok   bool
	}{
		{"URL: https://platform.claude.com/docs/en/get-started", "https://platform.claude.com/docs/en/get-started", true},
		{"Source: https://docs.digitalocean.com/reference/api/", "https://docs.digitalocean.com/reference/api/", true},
		{"# Some heading", "", false},
		{"---", "", false},
		{"URL: ", "", false},
		{"Source: ", "", false},
	}
	for _, tt := range tests {
		got, ok := parseSectionURL(tt.line)
		if ok != tt.ok || got != tt.want {
			t.Errorf("parseSectionURL(%q) = (%q, %v), want (%q, %v)", tt.line, got, ok, tt.want, tt.ok)
		}
	}
}

func TestSplitFrontmatterFullTxt(t *testing.T) {
	content := []byte(`---
title: Cloudflare Workers KV
description: Workers KV is edge storage.
image: https://developers.cloudflare.com/preview.png
---

[Skip to content](#_top)

Was this helpful?

YesNo

[ Edit page ](https://github.com/cloudflare/cloudflare-docs/edit/production/src/content/docs/kv/index.mdx) [ Report issue ](https://github.com/cloudflare/cloudflare-docs/issues/new/choose)

Copy page

# Cloudflare Workers KV

Create a global, low-latency, key-value data storage.

---

---
title: Getting started
description: Learn how to get started with KV.
image: https://developers.cloudflare.com/preview.png
---

[Skip to content](#_top)

Was this helpful?

YesNo

[ Edit page ](https://github.com/cloudflare/cloudflare-docs/edit/production/src/content/docs/kv/get-started.mdx) [ Report issue ](https://github.com/cloudflare/cloudflare-docs/issues/new/choose)

Copy page

# Getting started

Create a basic key-value store.

## Prerequisites

You need a Cloudflare account.
`)

	pages, err := SplitFrontmatterFullTxt(content, "https://developers.cloudflare.com/kv/llms-full.txt")
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(pages))
	}

	// First page: KV overview.
	if pages[0].Path != "cloudflare-workers-kv.md" {
		t.Errorf("page[0] Path = %q", pages[0].Path)
	}
	if !strings.Contains(string(pages[0].Content), "# Cloudflare Workers KV") {
		t.Error("page[0] missing heading")
	}
	// Boilerplate should be stripped.
	if strings.Contains(string(pages[0].Content), "Skip to content") {
		t.Error("page[0] boilerplate not stripped")
	}
	if strings.Contains(string(pages[0].Content), "Was this helpful") {
		t.Error("page[0] boilerplate not stripped")
	}
	if strings.Contains(string(pages[0].Content), "Edit page") {
		t.Error("page[0] boilerplate not stripped")
	}

	// Second page: Getting started.
	if pages[1].Path != "getting-started.md" {
		t.Errorf("page[1] Path = %q", pages[1].Path)
	}
	if !strings.Contains(string(pages[1].Content), "# Getting started") {
		t.Error("page[1] missing heading")
	}
	if !strings.Contains(string(pages[1].Content), "## Prerequisites") {
		t.Error("page[1] missing subheading")
	}
}

func TestSplitFrontmatterFullTxt_NoFrontmatter(t *testing.T) {
	content := []byte("# Just a heading\n\nSome content.\n")
	pages, err := SplitFrontmatterFullTxt(content, "https://example.com/llms-full.txt")
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 0 {
		t.Errorf("expected 0 pages for non-frontmatter content, got %d", len(pages))
	}
}

func TestTitleToArchivePath(t *testing.T) {
	tests := []struct {
		title string
		want  string
	}{
		{"Getting started", "getting-started.md"},
		{"Workers KV API", "workers-kv-api.md"},
		{"Cloudflare Workers KV", "cloudflare-workers-kv.md"},
		{"Create a namespace (API)", "create-a-namespace-api.md"},
		{"", "index.md"},
	}
	for _, tt := range tests {
		got := titleToArchivePath(tt.title)
		if got != tt.want {
			t.Errorf("titleToArchivePath(%q) = %q, want %q", tt.title, got, tt.want)
		}
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
