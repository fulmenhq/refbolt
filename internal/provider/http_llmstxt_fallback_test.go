package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const xaiIndexBody = `# SpaceXAI API Documentation

> Official docs at https://docs.x.ai

- [Get Started](https://docs.x.ai/developers/quickstart.md)
- [llms-full.txt](https://PLACEHOLDER/llms-full.txt): every documentation page concatenated into one markdown file (large)
`

const xaiDumpBody = `===/overview===
# Get started

Build with Grok.

===/developers/models===
# Models

Model documentation.
`

const sourceDelimitedDump = `# AGENT
Source: https://docs.x.com/AGENT

# X Developer Platform

Entire documentation set.

---
Source: https://docs.x.com/overview

# Overview

More pages.
`

func TestLooksLikeLLMSIndex(t *testing.T) {
	if !looksLikeLLMSIndex([]byte(xaiIndexBody)) {
		t.Fatal("xAI-style markdown index should be detected")
	}
	if looksLikeLLMSIndex([]byte(xaiDumpBody)) {
		t.Fatal("=== delimited dump should not look like an index")
	}
	if looksLikeLLMSIndex([]byte(sourceDelimitedDump)) {
		t.Fatal("Source: delimited dump should not look like an index")
	}
	if looksLikeLLMSIndex([]byte("plain text with no links")) {
		t.Fatal("plain text should not look like an index")
	}
}

func TestSiblingLLMSFullTxtURL(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"https://docs.x.ai/llms.txt", "https://docs.x.ai/llms-full.txt"},
		{"https://docs.x.com/x-api/llms.txt", "https://docs.x.com/x-api/llms-full.txt"},
		{"https://docs.x.ai/llms-full.txt", ""},
		{"https://platform.claude.com/llms-full.txt", ""},
		{"https://example.com/docs/", ""},
	}
	for _, tt := range tests {
		got := siblingLLMSFullTxtURL(tt.in)
		if got != tt.want {
			t.Errorf("siblingLLMSFullTxtURL(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestLLMSFullURLFromIndex(t *testing.T) {
	got := llmsFullURLFromIndex([]byte(strings.ReplaceAll(xaiIndexBody, "https://PLACEHOLDER/llms-full.txt", "https://docs.x.ai/llms-full.txt")))
	if got != "https://docs.x.ai/llms-full.txt" {
		t.Errorf("llmsFullURLFromIndex = %q", got)
	}
	if got := llmsFullURLFromIndex([]byte("# No links\n")); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestHTTPFetcher_IndexFallsBackToExplicitFullDump(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		switch r.URL.Path {
		case "/llms.txt":
			_, _ = w.Write([]byte(strings.ReplaceAll(xaiIndexBody, "https://PLACEHOLDER/llms-full.txt", "https://example.invalid/llms-full.txt")))
		case "/llms-full.txt":
			_, _ = w.Write([]byte(xaiDumpBody))
		case "/developers/models.md":
			_, _ = w.Write([]byte("# Supplemental models\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	f, err := NewHTTPFetcher(ProviderConfig{
		Slug:           "xai",
		Name:           "xAI",
		BaseURL:        srv.URL,
		FetchStrategy:  StrategyNative,
		LLMSTxtURL:     srv.URL + "/llms.txt",
		LLMSFullTxtURL: srv.URL + "/llms-full.txt",
		Paths:          []string{"/developers/models.md"},
	})
	if err != nil {
		t.Fatal(err)
	}

	pages, err := f.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	found := map[string]bool{}
	for _, p := range pages {
		found[p.Path] = true
	}
	for _, want := range []string{"llms.txt", "llms-full.txt", "overview.md", "developers/models.md"} {
		if !found[want] {
			t.Errorf("missing page %q in %v", want, found)
		}
	}

	for _, p := range pages {
		if p.Path == "overview.md" && !strings.Contains(string(p.Content), "Build with Grok") {
			t.Errorf("overview.md missing dump content: %s", p.Content)
		}
	}

	gotFull := false
	for _, p := range paths {
		if p == "GET /llms-full.txt" {
			gotFull = true
		}
	}
	if !gotFull {
		t.Errorf("expected GET /llms-full.txt, got %v", paths)
	}
}

func TestHTTPFetcher_DumpDoesNotFetchFull(t *testing.T) {
	var fullHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/llms-full.txt" {
			fullHits++
			_, _ = w.Write([]byte(xaiDumpBody))
			return
		}
		if r.URL.Path == "/llms.txt" {
			_, _ = w.Write([]byte(xaiDumpBody))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	f, err := NewHTTPFetcher(ProviderConfig{
		Slug:           "xai",
		BaseURL:        srv.URL,
		LLMSTxtURL:     srv.URL + "/llms.txt",
		LLMSFullTxtURL: srv.URL + "/llms-full.txt",
	})
	if err != nil {
		t.Fatal(err)
	}

	pages, err := f.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if fullHits != 0 {
		t.Errorf("llms-full.txt fetched %d times; dump already had sections", fullHits)
	}
	var gotOverview bool
	for _, p := range pages {
		if p.Path == "overview.md" {
			gotOverview = true
		}
	}
	if !gotOverview {
		t.Fatal("expected overview.md from primary dump")
	}
}

func TestHTTPFetcher_AutoDetectSiblingXAIDump(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/llms.txt":
			_, _ = w.Write([]byte("# Index\n\n- [Overview](/overview.md)\n"))
		case "/llms-full.txt":
			_, _ = w.Write([]byte(xaiDumpBody))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	f, err := NewHTTPFetcher(ProviderConfig{
		Slug:       "xai",
		BaseURL:    srv.URL,
		LLMSTxtURL: srv.URL + "/llms.txt",
		// No explicit llms_full_txt_url — sibling auto-detect.
	})
	if err != nil {
		t.Fatal(err)
	}

	pages, err := f.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, p := range pages {
		found[p.Path] = true
	}
	if !found["llms-full.txt"] || !found["overview.md"] {
		t.Errorf("auto-detect sibling should split llms-full.txt, got %v", found)
	}
}

func TestHTTPFetcher_AutoDetectIgnoresSourceDelimitedFirehose(t *testing.T) {
	var fullHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/llms.txt":
			_, _ = w.Write([]byte("# Index\n\n- [llms-full.txt](" + "http://" + r.Host + "/llms-full.txt)\n"))
		case "/llms-full.txt":
			fullHits++
			_, _ = w.Write([]byte(sourceDelimitedDump))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	f, err := NewHTTPFetcher(ProviderConfig{
		Slug:       "xdev",
		BaseURL:    srv.URL,
		LLMSTxtURL: srv.URL + "/llms.txt",
	})
	if err != nil {
		t.Fatal(err)
	}

	pages, err := f.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if fullHits == 0 {
		t.Fatal("expected a peek/GET of sibling llms-full.txt")
	}
	for _, p := range pages {
		if p.Path != "llms.txt" {
			t.Errorf("auto-detect should not ingest Source: firehose page %q", p.Path)
		}
	}
}

func TestHTTPFetcher_AnthropicStyleStillSplitsWithoutFallback(t *testing.T) {
	body := []byte(`---

# Get Started

URL: https://platform.claude.com/docs/en/get-started

# Get Started

Make your first API call.

---

# Tool Use

URL: https://platform.claude.com/docs/en/agents-and-tools/tool-use/overview

# Tool Use

Claude can use tools.
`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/llms-full.txt" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	f, err := NewHTTPFetcher(ProviderConfig{
		Slug:       "anthropic",
		BaseURL:    "https://platform.claude.com/docs",
		LLMSTxtURL: srv.URL + "/llms-full.txt",
	})
	if err != nil {
		t.Fatal(err)
	}
	pages, err := f.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) < 3 {
		t.Fatalf("expected raw dump + 2 sections, got %d", len(pages))
	}
	found := map[string]bool{}
	for _, p := range pages {
		found[p.Path] = true
	}
	if !found["llms-full.txt"] || !found["en/get-started.md"] {
		t.Errorf("anthropic-style split missing pages: %v", found)
	}
}

func TestHTTPFetcher_FullDump404KeepsIndex(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/llms.txt" {
			_, _ = w.Write([]byte(xaiIndexBody))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	f, err := NewHTTPFetcher(ProviderConfig{
		Slug:           "xai",
		BaseURL:        srv.URL,
		LLMSTxtURL:     srv.URL + "/llms.txt",
		LLMSFullTxtURL: srv.URL + "/llms-full.txt",
	})
	if err != nil {
		t.Fatal(err)
	}
	pages, err := f.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 1 || pages[0].Path != "llms.txt" {
		t.Errorf("expected only the index after full dump 404, got %+v", pages)
	}
}

func TestCheckHints_PrefersLLMSFullTxtURL(t *testing.T) {
	var headPaths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			headPaths = append(headPaths, r.URL.Path)
			w.Header().Set("ETag", `"full"`)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	f, err := NewHTTPFetcher(ProviderConfig{
		Slug:           "xai",
		BaseURL:        srv.URL,
		LLMSTxtURL:     srv.URL + "/llms.txt",
		LLMSFullTxtURL: srv.URL + "/llms-full.txt",
	})
	if err != nil {
		t.Fatal(err)
	}
	hint, err := f.CheckHints(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if hint.ETag != `"full"` {
		t.Errorf("ETag = %q", hint.ETag)
	}
	if len(headPaths) != 1 || headPaths[0] != "/llms-full.txt" {
		t.Errorf("HEAD paths = %v, want /llms-full.txt", headPaths)
	}
}

func TestUsesPerPathHints_LLMSFullTxtURL(t *testing.T) {
	f, err := NewHTTPFetcher(ProviderConfig{
		Slug:           "xai",
		BaseURL:        "https://docs.x.ai",
		FetchStrategy:  StrategyNative,
		LLMSFullTxtURL: "https://docs.x.ai/llms-full.txt",
		Paths:          []string{"a.md", "b.md"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.usesPerPathHints() {
		t.Fatal("native provider with llms_full_txt_url must not use per-path skip")
	}
}
