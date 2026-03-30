package provider

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
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

// newHTTPClient creates an HTTP client without a client-level timeout.
// Timeouts are enforced per-request via context deadlines so that Jina
// retries can extend the deadline without being capped by the client.
func newHTTPClient() *http.Client {
	// Some doc sites return 404 when Go's default HTTP/2 ALPN is negotiated.
	// Using HTTP/1.1 only resolves this compatibility issue.
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			NextProtos: []string{"http/1.1"},
		},
	}

	return &http.Client{
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

// CheckHints issues a HEAD request to the provider's primary URL
// to get ETag/Last-Modified/Content-Length without downloading content.
// Used by incremental sync to skip unchanged providers.
//
// Only safe for single-source providers (one llms_txt_url or one path).
// Multi-path providers (e.g., OpenAI with 3 paths + openapi_url) return
// an error — a HEAD on one path cannot represent the entire provider.
func (f *HTTPFetcher) CheckHints(ctx context.Context) (FetchHint, error) {
	var hint FetchHint

	// Multi-source providers: refuse to offer hints.
	// A HEAD on one URL cannot represent all upstream sources.
	hasMultipleSources := len(f.cfg.Paths) > 1 || (len(f.cfg.Paths) > 0 && f.cfg.OpenAPIURL != "")
	if hasMultipleSources {
		return hint, fmt.Errorf("multi-source provider %s: HEAD check cannot represent all upstream URLs", f.cfg.Slug)
	}

	// Determine which single URL to HEAD.
	targetURL := f.cfg.LLMSTxtURL
	if targetURL == "" && len(f.cfg.Paths) > 0 {
		targetURL = f.cfg.BaseURL + f.cfg.Paths[0]
	}
	if targetURL == "" {
		return hint, fmt.Errorf("no URL available for HEAD check")
	}

	timeout := f.cfg.EffectiveFetchTimeout()
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodHead, targetURL, nil)
	if err != nil {
		return hint, fmt.Errorf("creating HEAD request: %w", err)
	}
	req.Header.Set("User-Agent", "refbolt/0.1 (+https://github.com/fulmenhq/refbolt)")

	resp, err := f.client.Do(req)
	if err != nil {
		return hint, fmt.Errorf("HEAD %s: %w", targetURL, err)
	}
	defer resp.Body.Close()

	// HEAD not supported or error — caller falls back to full fetch.
	if resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode >= 400 {
		return hint, fmt.Errorf("HEAD %s returned %d", targetURL, resp.StatusCode)
	}

	hint.ETag = resp.Header.Get("ETag")
	hint.LastModified = resp.Header.Get("Last-Modified")
	hint.ContentLength = resp.ContentLength

	return hint, nil
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
			// Split into individual pages.
			// Try xAI-style delimiters first, then URL-based (Anthropic/DO-style).
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
			// Try frontmatter-based splitting (Cloudflare-style).
			if len(split) == 0 {
				var fmErr error
				split, fmErr = SplitFrontmatterFullTxt(llmsPage.Content, f.cfg.LLMSTxtURL)
				if fmErr != nil {
					fmt.Printf("  ⚠ %s: frontmatter-split error: %v\n", llmsFilename, fmErr)
				}
			}

			// Apply base_url prefix filter for scoped providers (e.g., DO).
			// If filtering reduces the set, this is a scoped provider — skip
			// the raw bulk file to avoid archiving 40MB per scoped entry.
			// If filtering passes everything through, archive the raw file
			// (backwards-compat with Anthropic, Pydantic, xAI).
			preFilterCount := len(split)
			split = FilterByBaseURL(split, f.cfg.BaseURL)
			scoped := len(split) < preFilterCount

			if scoped && len(split) == 0 {
				return nil, fmt.Errorf("no pages matched base_url scope %q in %s", f.cfg.BaseURL, f.cfg.LLMSTxtURL)
			}

			if !scoped {
				pages = append(pages, *llmsPage)
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
		if err == nil && !looksLikeHTML(page.Content) {
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
// Jina Reader converts HTML pages to clean Markdown by prepending https://r.jina.ai/
// to the target URL. Supports optional API key auth for higher rate limits.
//
// On timeout-shaped errors, retries once with 2x the provider timeout.
// Non-timeout errors (4xx, 5xx, connection refused, etc.) are not retried.
func (f *HTTPFetcher) fetchViaJina(ctx context.Context, rawURL, archivePath string) (*Page, error) {
	jinaURL := "https://r.jina.ai/" + rawURL
	return f.doJinaFetchWithRetry(ctx, jinaURL, rawURL, archivePath)
}

// doJinaFetchWithRetry wraps doJinaFetch with one timeout retry at 2x timeout.
// sourceURL is the original provider URL (for Page.SourceURL and user-facing errors).
// fetchURL is the actual Jina proxy URL used for the HTTP request.
func (f *HTTPFetcher) doJinaFetchWithRetry(ctx context.Context, fetchURL, sourceURL, archivePath string) (*Page, error) {
	timeout := f.cfg.EffectiveFetchTimeout()

	page, err := f.doJinaFetch(ctx, fetchURL, sourceURL, archivePath, timeout)
	if err != nil && isTimeoutError(err) {
		retryTimeout := timeout * 2
		fmt.Printf("    Jina timeout for %s (%s), retrying with %s...\n", sourceURL, timeout, retryTimeout)
		page, err = f.doJinaFetch(ctx, fetchURL, sourceURL, archivePath, retryTimeout)
		if err != nil {
			return nil, fmt.Errorf("fetching %s via Jina failed after retry (timeout: %s → %s): %w\n  Hint: increase fetch_timeout for this provider in providers.yaml", sourceURL, timeout, retryTimeout, err)
		}
	}
	return page, err
}

// doJinaFetch performs a single Jina Reader fetch with the given timeout.
// fetchURL is the Jina proxy URL; sourceURL is preserved in Page.SourceURL.
func (f *HTTPFetcher) doJinaFetch(ctx context.Context, fetchURL, sourceURL, archivePath string, timeout time.Duration) (*Page, error) {
	// Use a per-request timeout via context so retries can extend the deadline.
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, fetchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating Jina request: %w", err)
	}
	req.Header.Set("User-Agent", "refbolt/0.1 (+https://github.com/fulmenhq/refbolt)")
	req.Header.Set("Accept", "text/markdown")

	// Jina auth: uses only JINA_API_KEY — never the provider's auth_env_var,
	// which belongs to the provider and must not be sent to third parties.
	if apiKey := jinaAPIKey(f.cfg); apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %s via Jina: %w", sourceURL, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if err := checkJinaResponse(resp, sourceURL); err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading Jina response: %w", err)
	}

	// Strip Jina's metadata header (Title/URL Source/Markdown Content lines)
	// to get clean Markdown content.
	content := stripJinaHeader(body)

	return &Page{
		SourceURL: sourceURL,
		Path:      archivePath,
		Content:   content,
	}, nil
}

// isTimeoutError returns true if the error is a timeout-shaped failure
// (context deadline exceeded, net timeout, i/o timeout).
func isTimeoutError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	return false
}

// fetchURL does the actual HTTP GET and returns a Page.
func (f *HTTPFetcher) fetchURL(ctx context.Context, rawURL, archivePath string) (*Page, error) {
	reqCtx, cancel := context.WithTimeout(ctx, f.cfg.EffectiveFetchTimeout())
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", "refbolt/0.1 (+https://github.com/fulmenhq/refbolt)")
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

// jinaAPIKey returns the Jina Reader API key from JINA_API_KEY.
// This intentionally does NOT use the provider's auth_env_var — that credential
// belongs to the provider (e.g. OPENAI_API_KEY) and must never be sent to a
// third-party service like Jina Reader.
func jinaAPIKey(_ ProviderConfig) string {
	return strings.TrimSpace(os.Getenv("JINA_API_KEY"))
}

// checkJinaResponse inspects an HTTP response from Jina Reader and returns
// a descriptive error for non-200 status codes.
func checkJinaResponse(resp *http.Response, originalURL string) error {
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		retryAfter := resp.Header.Get("Retry-After")
		hint := ""
		if retryAfter != "" {
			hint = fmt.Sprintf("; retry after %ss", retryAfter)
		}
		return fmt.Errorf("jina reader rate limit for %s (HTTP 429%s); set JINA_API_KEY for higher limits", originalURL, hint)
	case http.StatusPaymentRequired:
		return fmt.Errorf("jina reader requires payment for %s (HTTP 402); check JINA_API_KEY quota", originalURL)
	case http.StatusUnprocessableEntity:
		return fmt.Errorf("jina reader could not process %s (HTTP 422); page may be too complex or blocked", originalURL)
	default:
		return fmt.Errorf("jina reader HTTP %d for %s", resp.StatusCode, originalURL)
	}
}

// stripJinaHeader removes the metadata header that Jina Reader prepends to
// its Markdown output. The header format is:
//
//	Title: <title>
//	URL Source: <url>
//	Published Time: <time>  (optional)
//	Markdown Content:
//	<actual content>
//
// If no header is detected, the content is returned unchanged.
func stripJinaHeader(body []byte) []byte {
	marker := []byte("\nMarkdown Content:\n")
	idx := bytes.Index(body, marker)
	if idx < 0 {
		return body
	}
	return body[idx+len(marker):]
}

// looksLikeHTML checks if content appears to be HTML rather than Markdown.
// Used by the auto strategy to decide whether to fall back to Jina.
func looksLikeHTML(content []byte) bool {
	// Check the first 1KB for HTML indicators.
	sample := content
	if len(sample) > 1024 {
		sample = sample[:1024]
	}
	lower := bytes.ToLower(sample)
	return bytes.Contains(lower, []byte("<!doctype html")) ||
		bytes.Contains(lower, []byte("<html"))
}
