package provider

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
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

	if len(matched) == 0 {
		return nil, fmt.Errorf("no llms.txt entry found in index for service prefix %q (provider %s); check that base_url matches an indexed service", prefix, f.cfg.Slug)
	}

	// One provider = one guide family. If multiple URLs match the prefix,
	// pick the one whose path is the closest match (longest common prefix).
	serviceURL := selectBestMatch(matched, prefix)
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

// selectBestMatch picks the index URL with the longest matching path prefix,
// ensuring one provider entry maps to exactly one guide family.
func selectBestMatch(urls []string, prefix string) string {
	best := urls[0]
	bestLen := 0
	for _, rawURL := range urls {
		u, err := url.Parse(rawURL)
		if err != nil {
			continue
		}
		// Count how many path segments match.
		common := commonPrefixLen(u.Path, prefix)
		if common > bestLen {
			bestLen = common
			best = rawURL
		}
	}
	return best
}

func commonPrefixLen(a, b string) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}
