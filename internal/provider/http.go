package provider

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

// HTTPFetcher is the default fetcher that retrieves pages via HTTP.
// It handles both direct Markdown URLs and HTML pages (via Jina Reader fallback).
type HTTPFetcher struct {
	cfg    ProviderConfig
	client *http.Client
}

func newHTTPClient() *http.Client {
	// Some doc sites return 404 when Go's default HTTP/2 ALPN is negotiated.
	// Using HTTP/1.1 only resolves this compatibility issue.
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			NextProtos: []string{"http/1.1"},
		},
	}

	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}
}

// NewHTTPFetcher creates a fetcher for any provider using HTTP.
func NewHTTPFetcher(cfg ProviderConfig) (*HTTPFetcher, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("base_url is required for provider %s", cfg.Slug)
	}
	return &HTTPFetcher{
		cfg:    cfg,
		client: newHTTPClient(),
	}, nil
}

func (f *HTTPFetcher) Name() string {
	return f.cfg.Name
}

// Fetch retrieves pages for all configured paths.
// If llms_txt_url is set, fetches and splits that first (most efficient).
// Then fetches any literal paths not covered by the llms.txt content.
func (f *HTTPFetcher) Fetch(ctx context.Context) ([]Page, error) {
	var pages []Page

	// Strategy 1: If llms_txt_url is configured, fetch and split it.
	// This is the most efficient path — one HTTP request gets all pages.
	if f.cfg.LLMSTxtURL != "" {
		// Derive the archive filename from the URL (e.g. "llms.txt", "llms-full.txt").
		llmsFilename := llmsTxtFilename(f.cfg.LLMSTxtURL)
		llmsPage, err := f.fetchURL(ctx, f.cfg.LLMSTxtURL, llmsFilename)
		if err != nil {
			fmt.Printf("  ⚠ %s: %v (falling back to individual pages)\n", llmsFilename, err)
		} else {
			// Save the raw file.
			pages = append(pages, *llmsPage)
			// Split into individual pages.
			// Try xAI-style delimiters first, then URL-based (Anthropic-style).
			split, err := SplitLLMSTxt(llmsPage.Content, f.cfg.LLMSTxtURL)
			if err != nil {
				fmt.Printf("  ⚠ %s: split error: %v\n", llmsFilename, err)
			}
			if len(split) == 0 {
				var fullErr error
				split, fullErr = SplitLLMSFullTxt(llmsPage.Content, f.cfg.LLMSTxtURL)
				if fullErr != nil {
					fmt.Printf("  ⚠ %s: full-split error: %v\n", llmsFilename, fullErr)
				}
			}
			pages = append(pages, split...)
			fmt.Printf("  ✓ %s: %d sections extracted\n", llmsFilename, len(split))
		}
	}

	// Strategy 2: Fetch individual paths (for pages not in llms.txt, or when no llms.txt).
	for _, p := range f.cfg.Paths {
		// Skip glob patterns for now — only fetch literal paths.
		if strings.Contains(p, "*") {
			continue
		}

		page, err := f.fetchPage(ctx, p)
		if err != nil {
			fmt.Printf("  ⚠ %s: %v\n", p, err)
			continue
		}
		pages = append(pages, *page)
	}

	// Fetch OpenAPI spec if configured.
	if f.cfg.OpenAPIURL != "" {
		page, err := f.fetchURL(ctx, f.cfg.OpenAPIURL, "openapi.yaml")
		if err != nil {
			fmt.Printf("  ⚠ openapi: %v\n", err)
		} else {
			pages = append(pages, *page)
		}
	}

	return pages, nil
}

// fetchPage fetches a single page by path from the provider's base URL.
func (f *HTTPFetcher) fetchPage(ctx context.Context, pagePath string) (*Page, error) {
	fullURL, err := url.JoinPath(f.cfg.BaseURL, pagePath)
	if err != nil {
		return nil, fmt.Errorf("joining URL: %w", err)
	}

	archivePath := pathToArchivePath(pagePath)

	strategy := f.cfg.FetchStrategy
	if strategy == "" {
		strategy = StrategyAuto
	}

	switch strategy {
	case StrategyNative:
		return f.fetchDirect(ctx, fullURL, archivePath)
	case StrategyJina:
		return f.fetchViaJina(ctx, fullURL, archivePath)
	case StrategyAuto:
		// Try direct first; if it fails or returns HTML, fall back to Jina.
		page, err := f.fetchDirect(ctx, fullURL, archivePath)
		if err == nil {
			return page, nil
		}
		return f.fetchViaJina(ctx, fullURL, archivePath)
	default:
		return nil, fmt.Errorf("unknown fetch strategy: %s", strategy)
	}
}

// fetchDirect fetches a URL and returns the content as-is.
func (f *HTTPFetcher) fetchDirect(ctx context.Context, rawURL, archivePath string) (*Page, error) {
	return f.fetchURL(ctx, rawURL, archivePath)
}

// fetchViaJina fetches a URL through Jina Reader for HTML-to-Markdown conversion.
func (f *HTTPFetcher) fetchViaJina(ctx context.Context, rawURL, archivePath string) (*Page, error) {
	jinaURL := "https://r.jina.ai/" + rawURL
	return f.fetchURL(ctx, jinaURL, archivePath)
}

// fetchURL does the actual HTTP GET and returns a Page.
func (f *HTTPFetcher) fetchURL(ctx context.Context, rawURL, archivePath string) (*Page, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", "fularchive/0.1 (+https://github.com/fulmenhq/fularchive)")
	// Do NOT use text/markdown in Accept — some CDNs (docs.x.ai) return 404 for it.
	req.Header.Set("Accept", "*/*")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", rawURL, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, rawURL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	return &Page{
		SourceURL: rawURL,
		Path:      archivePath,
		Content:   body,
	}, nil
}

// llmsTxtFilename extracts the filename from an llms_txt_url for use as the
// archive path. Falls back to "llms.txt" if the URL has no usable filename.
// Examples:
//
//	"https://docs.x.ai/llms.txt"                          → "llms.txt"
//	"https://docs.pydantic.dev/latest/llms-full.txt"      → "llms-full.txt"
func llmsTxtFilename(rawURL string) string {
	if u, err := url.Parse(rawURL); err == nil {
		if base := path.Base(u.Path); base != "" && base != "." && base != "/" {
			return base
		}
	}
	return "llms.txt"
}

// pathToArchivePath converts a URL path to a filesystem-safe archive path.
// Examples:
//
//	"/en/docs/build-with-claude/tool-use" → "en/docs/build-with-claude/tool-use.md"
//	"/llms-full.txt" → "llms-full.txt"
//	"/developers/tools/overview" → "developers/tools/overview.md"
func pathToArchivePath(urlPath string) string {
	// Strip leading slash.
	p := strings.TrimPrefix(urlPath, "/")
	if p == "" {
		p = "index.md"
	}

	// If it already has an extension, keep it.
	ext := path.Ext(p)
	if ext != "" {
		return p
	}

	// Otherwise, add .md
	return p + ".md"
}
