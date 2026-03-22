package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGitHubRawFetcherFetch(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-token")

	var repoRequests int
	var apiAuthHeader string
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/trinodb/trino" {
			repoRequests++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"default_branch":"main"}`))
			return
		}
		apiAuthHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tree":[{"path":"docs/src/main/sphinx/admin/web-interface.md","type":"blob"},{"path":"docs/src/main/sphinx/security/_index.md","type":"blob"},{"path":"docs/src/main/sphinx/security/rules.txt","type":"blob"},{"path":"README.md","type":"blob"}]}`))
	}))
	defer apiServer.Close()

	var rawAuthHeaders []string
	rawServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawAuthHeaders = append(rawAuthHeaders, r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/trinodb/trino/main/docs/src/main/sphinx/admin/web-interface.md":
			_, _ = w.Write([]byte("# Web Interface\n"))
		case "/trinodb/trino/main/docs/src/main/sphinx/security/_index.md":
			_, _ = w.Write([]byte("# Security\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer rawServer.Close()

	fetcher, err := NewGitHubRawFetcher(ProviderConfig{
		Slug:           "trino",
		Name:           "Trino",
		BaseURL:        "https://trino.io/docs/current",
		FetchStrategy:  StrategyGitHubRaw,
		GitHubRepo:     "trinodb/trino",
		GitHubDocsPath: "docs/src/main/sphinx/",
		Paths:          []string{"**/*.md"},
		RateLimit:      &RateLimitConfig{RequestsPerSecond: 1000},
	})
	if err != nil {
		t.Fatal(err)
	}
	fetcher.apiBaseURL = apiServer.URL
	fetcher.rawBaseURL = rawServer.URL

	pages, err := fetcher.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if apiAuthHeader != "Bearer test-token" {
		t.Fatalf("Authorization header = %q, want Bearer test-token", apiAuthHeader)
	}
	if repoRequests != 1 {
		t.Fatalf("repo metadata requests = %d, want 1", repoRequests)
	}
	if len(pages) != 2 {
		t.Fatalf("len(pages) = %d, want 2", len(pages))
	}
	if pages[0].Path != "admin/web-interface.md" {
		t.Fatalf("pages[0].Path = %q, want admin/web-interface.md", pages[0].Path)
	}
	if pages[1].Path != "security/index.md" {
		t.Fatalf("pages[1].Path = %q, want security/index.md", pages[1].Path)
	}
	if strings.TrimSpace(string(pages[0].Content)) != "# Web Interface" {
		t.Fatalf("pages[0].Content = %q", string(pages[0].Content))
	}
	for _, header := range rawAuthHeaders {
		if header != "" {
			t.Fatalf("raw request Authorization header = %q, want empty", header)
		}
	}
}

func TestGitHubRawFetcherRateLimitErrorIncludesTokenHint(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Resource", "core")
		w.Header().Set("X-RateLimit-Reset", "1777777777")
		w.WriteHeader(http.StatusForbidden)
	}))
	defer apiServer.Close()

	fetcher, err := NewGitHubRawFetcher(ProviderConfig{
		Slug:           "trino",
		Name:           "Trino",
		BaseURL:        "https://trino.io/docs/current",
		FetchStrategy:  StrategyGitHubRaw,
		GitHubRepo:     "trinodb/trino",
		GitHubDocsPath: "docs/src/main/sphinx/",
		Paths:          []string{"**/*.md"},
		AuthEnvVar:     "CUSTOM_GITHUB_TOKEN",
		RateLimit:      &RateLimitConfig{RequestsPerSecond: 1000},
	})
	if err != nil {
		t.Fatal(err)
	}
	fetcher.apiBaseURL = apiServer.URL

	_, err = fetcher.Fetch(context.Background())
	if err == nil {
		t.Fatal("Fetch() error = nil, want rate limit error")
	}
	if !strings.Contains(err.Error(), "CUSTOM_GITHUB_TOKEN") {
		t.Fatalf("error = %q, want CUSTOM_GITHUB_TOKEN hint", err)
	}
	if !strings.Contains(err.Error(), "authenticated access") {
		t.Fatalf("error = %q, want authenticated access hint", err)
	}
}

func TestGitHubRawFetcherTooManyRequestsIncludesTokenHint(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer apiServer.Close()

	fetcher, err := NewGitHubRawFetcher(ProviderConfig{
		Slug:           "trino",
		Name:           "Trino",
		BaseURL:        "https://trino.io/docs/current",
		FetchStrategy:  StrategyGitHubRaw,
		GitHubRepo:     "trinodb/trino",
		GitHubDocsPath: "docs/src/main/sphinx/",
		RateLimit:      &RateLimitConfig{RequestsPerSecond: 1000},
	})
	if err != nil {
		t.Fatal(err)
	}
	fetcher.apiBaseURL = apiServer.URL

	_, err = fetcher.Fetch(context.Background())
	if err == nil {
		t.Fatal("Fetch() error = nil, want rate limit error")
	}
	if !strings.Contains(err.Error(), "GITHUB_TOKEN") {
		t.Fatalf("error = %q, want GITHUB_TOKEN hint", err)
	}
	if !strings.Contains(err.Error(), "status=429") {
		t.Fatalf("error = %q, want status=429", err)
	}
}

func TestGitHubRawFetcherRaw403ReturnsGenericHTTPError(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/trinodb/trino" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"default_branch":"main"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tree":[{"path":"docs/src/main/sphinx/admin/web-interface.md","type":"blob"}]}`))
	}))
	defer apiServer.Close()

	rawServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer rawServer.Close()

	fetcher, err := NewGitHubRawFetcher(ProviderConfig{
		Slug:           "trino",
		Name:           "Trino",
		BaseURL:        "https://trino.io/docs/current",
		FetchStrategy:  StrategyGitHubRaw,
		GitHubRepo:     "trinodb/trino",
		GitHubDocsPath: "docs/src/main/sphinx/",
		AuthEnvVar:     "GITHUB_TOKEN",
		RateLimit:      &RateLimitConfig{RequestsPerSecond: 1000},
		Paths:          []string{"**/*.md"},
	})
	if err != nil {
		t.Fatal(err)
	}
	fetcher.apiBaseURL = apiServer.URL
	fetcher.rawBaseURL = rawServer.URL

	_, err = fetcher.Fetch(context.Background())
	if err == nil {
		t.Fatal("Fetch() error = nil, want HTTP 403 error")
	}
	if !strings.Contains(err.Error(), "HTTP 403") {
		t.Fatalf("error = %q, want HTTP 403", err)
	}
	if strings.Contains(err.Error(), "GITHUB_TOKEN") {
		t.Fatalf("error = %q, should not include token hint for raw fetch", err)
	}
}

func TestPatternMatcherMatches(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		target   string
		want     bool
	}{
		{name: "root markdown", patterns: []string{"**/*.md"}, target: "index.md", want: true},
		{name: "nested markdown", patterns: []string{"**/*.md"}, target: "security/index.md", want: true},
		{name: "exact subtree", patterns: []string{"docs/**/*.md"}, target: "docs/reference/index.md", want: true},
		{name: "reject other extension", patterns: []string{"**/*.md"}, target: "security/rules.txt", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher, err := newPatternMatcher("test-provider", tt.patterns)
			if err != nil {
				t.Fatalf("newPatternMatcher() error = %v", err)
			}
			got := matcher.matches(tt.target)
			if got != tt.want {
				t.Fatalf("matches(%v, %q) = %v, want %v", tt.patterns, tt.target, got, tt.want)
			}
		})
	}
}

func TestBuildGitHubRawURL(t *testing.T) {
	got := buildGitHubRawURL(
		"https://raw.githubusercontent.com",
		"trinodb/trino",
		"release/v1",
		"docs/src/main/sphinx/query language.md",
	)
	want := "https://raw.githubusercontent.com/trinodb/trino/release%2Fv1/docs/src/main/sphinx/query%20language.md"
	if got != want {
		t.Fatalf("buildGitHubRawURL() = %q, want %q", got, want)
	}
}

func TestGitHubRawFetcherErrorsWhenNoFilesMatch(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/trinodb/trino" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"default_branch":"main"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tree":[{"path":"docs/src/main/sphinx/security/rules.txt","type":"blob"}]}`))
	}))
	defer apiServer.Close()

	fetcher, err := NewGitHubRawFetcher(ProviderConfig{
		Slug:           "trino",
		Name:           "Trino",
		BaseURL:        "https://trino.io/docs/current",
		FetchStrategy:  StrategyGitHubRaw,
		GitHubRepo:     "trinodb/trino",
		GitHubDocsPath: "docs/src/main/sphinx/",
		Paths:          []string{"**/*.md"},
		RateLimit:      &RateLimitConfig{RequestsPerSecond: 1000},
	})
	if err != nil {
		t.Fatal(err)
	}
	fetcher.apiBaseURL = apiServer.URL

	_, err = fetcher.Fetch(context.Background())
	if err == nil {
		t.Fatal("Fetch() error = nil, want no-match error")
	}
	if !strings.Contains(err.Error(), "no GitHub files matched") {
		t.Fatalf("error = %q, want no-match error", err)
	}
}

func TestGitHubRawFetcherResolvesDefaultBranchWhenUnset(t *testing.T) {
	var treePath string
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/trinodb/trino":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"default_branch":"master"}`))
		case "/repos/trinodb/trino/git/trees/master":
			treePath = r.URL.Path + "?" + r.URL.RawQuery
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"tree":[{"path":"docs/src/main/sphinx/admin/web-interface.md","type":"blob"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer apiServer.Close()

	rawServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("# Web Interface\n"))
	}))
	defer rawServer.Close()

	fetcher, err := NewGitHubRawFetcher(ProviderConfig{
		Slug:           "trino",
		Name:           "Trino",
		BaseURL:        "https://trino.io/docs/current",
		FetchStrategy:  StrategyGitHubRaw,
		GitHubRepo:     "trinodb/trino",
		GitHubDocsPath: "docs/src/main/sphinx/",
		Paths:          []string{"**/*.md"},
		RateLimit:      &RateLimitConfig{RequestsPerSecond: 1000},
	})
	if err != nil {
		t.Fatal(err)
	}
	fetcher.apiBaseURL = apiServer.URL
	fetcher.rawBaseURL = rawServer.URL

	pages, err := fetcher.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if len(pages) != 1 {
		t.Fatalf("len(pages) = %d, want 1", len(pages))
	}
	if treePath != "/repos/trinodb/trino/git/trees/master?recursive=1" {
		t.Fatalf("tree request path = %q, want default branch master", treePath)
	}
}

func TestGitHubRawFetcherErrorsWhenDefaultBranchMissing(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/trinodb/trino":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"default_branch":""}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer apiServer.Close()

	fetcher, err := NewGitHubRawFetcher(ProviderConfig{
		Slug:           "trino",
		Name:           "Trino",
		BaseURL:        "https://trino.io/docs/current",
		FetchStrategy:  StrategyGitHubRaw,
		GitHubRepo:     "trinodb/trino",
		GitHubDocsPath: "docs/src/main/sphinx/",
		Paths:          []string{"**/*.md"},
		RateLimit:      &RateLimitConfig{RequestsPerSecond: 1000},
	})
	if err != nil {
		t.Fatal(err)
	}
	fetcher.apiBaseURL = apiServer.URL

	_, err = fetcher.Fetch(context.Background())
	if err == nil {
		t.Fatal("Fetch() error = nil, want missing default_branch error")
	}
	if !strings.Contains(err.Error(), "default_branch") {
		t.Fatalf("error = %q, want default_branch hint", err)
	}
}

func TestGitHubRawFetcherUsesConfiguredBranchWithoutRepoLookup(t *testing.T) {
	var repoRequests int
	var treePath string
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/trinodb/trino":
			repoRequests++
			http.Error(w, "unexpected repo metadata lookup", http.StatusInternalServerError)
		case "/repos/trinodb/trino/git/trees/release/1.0":
			treePath = r.RequestURI
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"tree":[{"path":"docs/src/main/sphinx/admin/web-interface.md","type":"blob"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer apiServer.Close()

	rawServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("# Web Interface\n"))
	}))
	defer rawServer.Close()

	fetcher, err := NewGitHubRawFetcher(ProviderConfig{
		Slug:           "trino",
		Name:           "Trino",
		BaseURL:        "https://trino.io/docs/current",
		FetchStrategy:  StrategyGitHubRaw,
		GitHubRepo:     "trinodb/trino",
		GitHubDocsPath: "docs/src/main/sphinx/",
		GitHubBranch:   "release/1.0",
		Paths:          []string{"**/*.md"},
		RateLimit:      &RateLimitConfig{RequestsPerSecond: 1000},
	})
	if err != nil {
		t.Fatal(err)
	}
	fetcher.apiBaseURL = apiServer.URL
	fetcher.rawBaseURL = rawServer.URL

	pages, err := fetcher.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if len(pages) != 1 {
		t.Fatalf("len(pages) = %d, want 1", len(pages))
	}
	if repoRequests != 0 {
		t.Fatalf("repo metadata requests = %d, want 0", repoRequests)
	}
	if treePath != "/repos/trinodb/trino/git/trees/release%2F1.0?recursive=1" {
		t.Fatalf("tree request path = %q, want configured branch release/1.0", treePath)
	}
}
