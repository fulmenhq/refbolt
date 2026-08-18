package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// IncrementalFetcher configures per-path conditional GET for native multi-path
// providers. Call before Fetch when prior hints and a latest archive directory
// are available from a previous sync.
type IncrementalFetcher interface {
	SetIncrementalContext(archiveLatestDir string, prior FetchHint)
}

// SetIncrementalContext enables conditional GET for literal native paths.
func (f *HTTPFetcher) SetIncrementalContext(archiveLatestDir string, prior FetchHint) {
	f.archiveLatestDir = archiveLatestDir
	f.priorHint = prior
}

// LastFetchHint returns hints collected during the most recent Fetch.
func (f *HTTPFetcher) LastFetchHint() FetchHint {
	return f.lastFetchHint
}

func (f *HTTPFetcher) usesPerPathHints() bool {
	if f.cfg.FetchStrategy != StrategyNative {
		return false
	}
	// Providers with a bulk llms_txt_url use single-source HEAD on that URL for
	// provider-level skip (Anthropic, xAI, Pydantic). Per-path HEAD skip applies
	// only to path-only native providers such as starlink-api-v2-status.
	if f.cfg.LLMSTxtURL != "" {
		return false
	}
	return len(literalPaths(f.cfg.Paths)) > 1 ||
		(len(literalPaths(f.cfg.Paths)) > 0 && f.cfg.OpenAPIURL != "")
}

func literalPaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if p == "" || strings.Contains(p, "*") {
			continue
		}
		out = append(out, p)
	}
	return out
}

func (f *HTTPFetcher) checkPathHints(ctx context.Context) (FetchHint, error) {
	paths := literalPaths(f.cfg.Paths)
	if len(paths) == 0 {
		return FetchHint{}, fmt.Errorf("no literal paths for per-path hint check")
	}

	hint := FetchHint{PathHints: make(map[string]FetchHint, len(paths))}
	for _, pagePath := range paths {
		fullURL, err := pageURL(f.cfg.BaseURL, pagePath)
		if err != nil {
			return FetchHint{}, err
		}
		archivePath := pathToArchivePath(pagePath)
		ph, err := f.headURL(ctx, fullURL)
		if err != nil {
			return FetchHint{}, fmt.Errorf("HEAD %s: %w", fullURL, err)
		}
		hint.PathHints[archivePath] = ph
	}
	return hint, nil
}

func (f *HTTPFetcher) headURL(ctx context.Context, rawURL string) (FetchHint, error) {
	var hint FetchHint

	timeout := f.cfg.EffectiveFetchTimeout()
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodHead, rawURL, nil)
	if err != nil {
		return hint, fmt.Errorf("creating HEAD request: %w", err)
	}
	req.Header.Set("User-Agent", "refbolt/0.1 (+https://github.com/fulmenhq/refbolt)")

	resp, err := f.client.Do(req)
	if err != nil {
		return hint, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode >= 400 {
		return hint, fmt.Errorf("HEAD returned %d", resp.StatusCode)
	}

	hint.ETag = resp.Header.Get("ETag")
	hint.LastModified = resp.Header.Get("Last-Modified")
	hint.ContentLength = resp.ContentLength
	return hint, nil
}

func fetchHintEqual(a, b FetchHint) bool {
	if a.ETag != "" && b.ETag != "" {
		return a.ETag == b.ETag
	}
	if a.LastModified != "" && b.LastModified != "" &&
		a.ContentLength > 0 && b.ContentLength > 0 {
		return a.LastModified == b.LastModified && a.ContentLength == b.ContentLength
	}
	return false
}

// PathHintsMatch reports whether all current per-path hints match stored values.
func PathHintsMatch(stored, current map[string]FetchHint) bool {
	if len(stored) == 0 || len(current) == 0 {
		return false
	}
	if len(stored) != len(current) {
		return false
	}
	for k, cur := range current {
		prev, ok := stored[k]
		if !ok || !fetchHintEqual(prev, cur) {
			return false
		}
	}
	return true
}

func pageURL(baseURL, pagePath string) (string, error) {
	return url.JoinPath(baseURL, strings.TrimPrefix(pagePath, "/"))
}

func (f *HTTPFetcher) fetchNativeConditional(ctx context.Context, fullURL, archivePath string, prior FetchHint) (*Page, FetchHint, error) {
	reqCtx, cancel := context.WithTimeout(ctx, f.cfg.EffectiveFetchTimeout())
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, FetchHint{}, err
	}
	req.Header.Set("User-Agent", "refbolt/0.1 (+https://github.com/fulmenhq/refbolt)")
	req.Header.Set("Accept", "*/*")
	if prior.ETag != "" {
		req.Header.Set("If-None-Match", prior.ETag)
	} else if prior.LastModified != "" {
		req.Header.Set("If-Modified-Since", prior.LastModified)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, FetchHint{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotModified {
		body, readErr := readArchivedPage(f.archiveLatestDir, archivePath)
		if readErr != nil {
			page, fetchErr := f.fetchURL(ctx, fullURL, archivePath)
			if fetchErr != nil {
				return nil, FetchHint{}, fetchErr
			}
			return page, prior, nil
		}
		return &Page{
			SourceURL: fullURL,
			Path:      archivePath,
			Content:   body,
		}, prior, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, FetchHint{}, fmt.Errorf("HTTP %d for %s", resp.StatusCode, fullURL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, FetchHint{}, err
	}

	return &Page{
		SourceURL: fullURL,
		Path:      archivePath,
		Content:   body,
	}, responseHint(resp), nil
}

func responseHint(resp *http.Response) FetchHint {
	return FetchHint{
		ETag:          resp.Header.Get("ETag"),
		LastModified:  resp.Header.Get("Last-Modified"),
		ContentLength: resp.ContentLength,
	}
}

func readArchivedPage(latestDir, archivePath string) ([]byte, error) {
	if latestDir == "" {
		return nil, fmt.Errorf("no archive directory configured")
	}
	return os.ReadFile(filepath.Join(latestDir, archivePath))
}
