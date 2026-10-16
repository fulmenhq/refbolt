package provider

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// HierarchicalFetcher implements the llmstxt-hierarchical strategy.
// It fetches a top-level llms.txt index, finds the per-service llms.txt URL
// that matches the provider's base_url prefix, fetches that service-level
// llms.txt, and splits it into individual pages.
type HierarchicalFetcher struct {
	cfg    ProviderConfig
	client *HTTPFetcher
}

// NewHierarchicalFetcher creates a fetcher for providers that use a
// hierarchical llms.txt index (e.g., AWS).
func NewHierarchicalFetcher(cfg ProviderConfig) (*HierarchicalFetcher, error) {
	if cfg.LLMSTxtURL == "" {
		return nil, fmt.Errorf("llms_txt_url is required for hierarchical strategy on provider %s", cfg.Slug)
	}
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("base_url is required for hierarchical strategy on provider %s", cfg.Slug)
	}
	base, err := url.Parse(cfg.BaseURL)
	if err != nil || base.Host == "" || (base.Scheme != "https" && base.Scheme != "http") {
		return nil, fmt.Errorf("base_url must be an HTTP documentation URL for provider %s", cfg.Slug)
	}
	httpFetcher, err := NewHTTPFetcher(cfg)
	if err != nil {
		return nil, err
	}
	return &HierarchicalFetcher{
		cfg:    cfg,
		client: httpFetcher,
	}, nil
}

func (f *HierarchicalFetcher) Name() string {
	return f.cfg.Name
}

// Fetch retrieves the top-level index, finds the matching service llms.txt,
// fetches it, and splits it into pages.
func (f *HierarchicalFetcher) Fetch(ctx context.Context) ([]Page, error) {
	// 1. Fetch the top-level index.
	indexPage, err := f.client.fetchURL(ctx, f.cfg.LLMSTxtURL, "index-llms.txt")
	if err != nil {
		return nil, fmt.Errorf("fetching hierarchical index %s: %w", f.cfg.LLMSTxtURL, err)
	}

	// 2. Parse index and find matching service llms.txt URL.
	prefix := servicePrefix(f.cfg.BaseURL)
	allURLs := parseIndexLLMSTxtURLs(indexPage.Content)
	matched := matchServiceURLs(allURLs, prefix)
	base, _ := url.Parse(f.cfg.BaseURL)
	var scoped []string
	seen := make(map[string]bool)
	for _, raw := range matched {
		u, err := url.Parse(raw)
		if err == nil && u.Host == base.Host && u.Scheme == base.Scheme && !seen[raw] {
			scoped = append(scoped, raw)
			seen[raw] = true
		}
	}
	matched = scoped

	if len(matched) == 0 {
		return nil, fmt.Errorf("no llms.txt entry found in index for service prefix %q (provider %s); check that base_url matches an indexed service", prefix, f.cfg.Slug)
	}

	if len(matched) != 1 {
		return nil, fmt.Errorf("ambiguous service prefix %q: matched %d guides; use a guide-specific base_url", prefix, len(matched))
	}
	serviceURL := matched[0]
	fmt.Printf("  → %s: matched index entry %s\n", f.cfg.Slug, serviceURL)

	// 3. Fetch the service-level llms.txt (or llms-full.txt).
	serviceFilename := llmsTxtFilename(serviceURL)
	servicePage, err := f.client.fetchURL(ctx, serviceURL, serviceFilename)
	if err != nil {
		return nil, fmt.Errorf("fetching service llms.txt %s: %w", serviceURL, err)
	}

	var pages []Page
	// Save the raw service llms.txt.
	pages = append(pages, *servicePage)

	// 4. Split into individual pages using existing splitters.
	split, err := SplitLLMSTxt(servicePage.Content, serviceURL)
	if err != nil {
		fmt.Printf("  ⚠ %s: split error: %v\n", f.cfg.Slug, err)
	}
	if len(split) == 0 {
		var fullErr error
		split, fullErr = SplitLLMSFullTxt(servicePage.Content, serviceURL)
		if fullErr != nil {
			fmt.Printf("  ⚠ %s: full-split error: %v\n", f.cfg.Slug, fullErr)
		}
	}
	pages = append(pages, split...)
	fmt.Printf("  ✓ %s: %d sections from service llms.txt\n", f.cfg.Slug, len(split))
	curated, err := f.fetchCuratedPages(ctx, servicePage.Content)
	if err != nil {
		return nil, err
	}
	pages = append(pages, curated...)

	return pages, nil
}

var guideMarkdownLink = regexp.MustCompile(`\[[^\]]*\]\((https?://[^\s)]+)\)`)

// Fetch only literal Markdown paths published by the selected guide's index.
// This does not crawl the guide or follow links recursively.
func (f *HierarchicalFetcher) fetchCuratedPages(ctx context.Context, index []byte) ([]Page, error) {
	allowed := make(map[string]bool)
	for _, m := range guideMarkdownLink.FindAllSubmatch(index, -1) {
		allowed[string(m[1])] = true
	}
	base, _ := url.Parse(f.cfg.BaseURL)
	var targets []string
	seen := make(map[string]bool)
	for _, p := range f.cfg.Paths {
		if strings.ContainsAny(p, "*?[#") {
			return nil, fmt.Errorf("hierarchical path %q must be a literal Markdown path", p)
		}
		target, err := pageURL(f.cfg.BaseURL, p)
		if err != nil {
			return nil, err
		}
		u, err := url.Parse(target)
		if err != nil || u.Host != base.Host || !strings.HasPrefix(u.Path, servicePrefix(f.cfg.BaseURL)) ||
			!strings.HasSuffix(u.Path, ".md") || !allowed[target] {
			return nil, fmt.Errorf("path %q is not a Markdown page published by guide %s", p, f.cfg.BaseURL)
		}
		if !seen[target] {
			targets = append(targets, target)
			seen[target] = true
		}
	}
	interval := time.Second
	if f.cfg.RateLimit != nil && f.cfg.RateLimit.RequestsPerSecond > 0 {
		interval = time.Duration(float64(time.Second) / f.cfg.RateLimit.RequestsPerSecond)
	}
	var pages []Page
	for i, target := range targets {
		if i > 0 {
			timer := time.NewTimer(interval)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			case <-timer.C:
			}
		}
		u, _ := url.Parse(target)
		pagePath := strings.TrimPrefix(u.Path, servicePrefix(f.cfg.BaseURL))
		page, err := f.client.fetchURL(ctx, target, pagePath)
		if err != nil {
			return nil, fmt.Errorf("fetching curated guide page: %w", err)
		}
		if looksLikeHTML(page.Content) || len(bytes.TrimSpace(page.Content)) == 0 {
			return nil, fmt.Errorf("guide page %s did not return nonempty Markdown", target)
		}
		pages = append(pages, *page)
	}
	return pages, nil
}

// llmsTxtLinkRe matches Markdown inline links to llms.txt files:
//
//	[llms.txt](https://docs.aws.amazon.com/glue/latest/dg/llms.txt)
var llmsTxtLinkRe = regexp.MustCompile(`\[llms(?:-full)?\.txt\]\((https?://[^\s)]+)\)`)

// parseIndexLLMSTxtURLs extracts all llms.txt URLs from a hierarchical index.
func parseIndexLLMSTxtURLs(content []byte) []string {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	var urls []string
	for scanner.Scan() {
		line := scanner.Text()
		matches := llmsTxtLinkRe.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			if len(m) >= 2 {
				urls = append(urls, m[1])
			}
		}
	}
	return urls
}

// servicePrefix derives the URL path prefix from a provider's base_url
// for use in matching against index entries.
//
// Examples:
//
//	"https://docs.aws.amazon.com/glue/latest"     → "/glue/latest/"
//	"https://docs.aws.amazon.com/bedrock/latest/userguide" → "/bedrock/latest/userguide/"
func servicePrefix(baseURL string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	p := strings.TrimRight(u.Path, "/")
	if p == "" {
		return "/"
	}
	return p + "/"
}

// matchServiceURLs returns index URLs whose path starts with the given prefix.
// This is an exact prefix match — "/bedrock/latest/userguide/" will NOT match
// "/bedrock-agentcore/latest/...".
func matchServiceURLs(indexURLs []string, prefix string) []string {
	var matched []string
	for _, rawURL := range indexURLs {
		u, err := url.Parse(rawURL)
		if err != nil {
			continue
		}
		// Normalize: ensure path has trailing slash for prefix comparison.
		path := u.Path
		if !strings.HasSuffix(path, "/") {
			// For llms.txt URLs like /glue/latest/dg/llms.txt, check the directory.
			dir := path[:strings.LastIndex(path, "/")+1]
			if strings.HasPrefix(dir, prefix) {
				matched = append(matched, rawURL)
			}
		} else if strings.HasPrefix(path, prefix) {
			matched = append(matched, rawURL)
		}
	}
	return matched
}
