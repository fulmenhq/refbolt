package config

import (
	"strings"
	"testing"
)

func TestParseRegistry_GoldenPath(t *testing.T) {
	data := strings.Join([]string{
		`{"slug":"xai","description":"xAI Grok API","estimated_pages":96}`,
		`{"slug":"anthropic","description":"Claude API","estimated_pages":488}`,
		``,
		`{"slug":"openai","description":"OpenAI API","estimated_pages":150}`,
	}, "\n")

	got, err := ParseRegistry([]byte(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d entries, want 3", len(got))
	}
	if got["anthropic"].EstimatedPages != 488 {
		t.Errorf("anthropic.EstimatedPages = %d, want 488", got["anthropic"].EstimatedPages)
	}
	if got["xai"].Description != "xAI Grok API" {
		t.Errorf("xai.Description = %q, want %q", got["xai"].Description, "xAI Grok API")
	}
}

func TestParseRegistry_IgnoresExtraFields(t *testing.T) {
	// Registry entries carry many fields we don't surface in v0.0.4;
	// json.Unmarshal should silently ignore them.
	line := `{"slug":"trino","description":"Trino docs","estimated_pages":641,"tags":["data"],"tos_reviewed":true,"notes":"deep info","capabilities":{"llms_txt":false}}`
	got, err := ParseRegistry([]byte(line))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	e := got["trino"]
	if e.Description != "Trino docs" || e.EstimatedPages != 641 {
		t.Errorf("unexpected entry: %+v", e)
	}
}

func TestParseRegistry_MalformedLineReportsLineNumber(t *testing.T) {
	data := strings.Join([]string{
		`{"slug":"xai","description":"ok","estimated_pages":96}`,
		`not json`,
	}, "\n")
	_, err := ParseRegistry([]byte(data))
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "line 2") {
		t.Errorf("error should reference line 2, got: %v", err)
	}
}

func TestParseRegistry_MissingSlugErrors(t *testing.T) {
	_, err := ParseRegistry([]byte(`{"description":"no slug here","estimated_pages":10}`))
	if err == nil {
		t.Fatal("expected missing-slug error")
	}
	if !strings.Contains(err.Error(), "missing slug") {
		t.Errorf("want 'missing slug' in error, got: %v", err)
	}
}

func TestLoadRegistry_EmptyWhenUnset(t *testing.T) {
	// Ensure no leakage from other tests.
	t.Cleanup(func() { SetEmbeddedRegistry(nil) })
	SetEmbeddedRegistry(nil)

	got, err := LoadRegistry()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d entries, want 0", len(got))
	}
}
