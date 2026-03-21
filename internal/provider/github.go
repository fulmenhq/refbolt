package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	defaultGitHubAPIBaseURL         = "https://api.github.com"
	defaultGitHubRawBaseURL         = "https://raw.githubusercontent.com"
	defaultGitHubAuthedRPS          = 5.0
	defaultGitHubUnauthenticatedRPS = 2.0
	gitHubAPIVersion                = "2022-11-28"
)

// GitHubRawFetcher discovers Markdown files through the GitHub tree API and
// then fetches their raw content from raw.githubusercontent.com.
type GitHubRawFetcher struct {
	cfg                ProviderConfig
	client             *http.Client
	apiBaseURL         string
	rawBaseURL         string
	branch             string
	authToken          string
	minRequestInterval time.Duration
	lastRequestAt      time.Time
}

type gitHubTreeResponse struct {
	Truncated bool             `json:"truncated"`
	Tree      []gitHubTreeNode `json:"tree"`
}

type gitHubTreeNode struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

type gitHubFile struct {
	RepoPath    string
	ArchivePath string
}

// NewGitHubRawFetcher creates a fetcher for providers whose Markdown source is
// stored in a public GitHub repository.
func NewGitHubRawFetcher(cfg ProviderConfig) (*GitHubRawFetcher, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("base_url is required for provider %s", cfg.Slug)
	}
	if cfg.GitHubRepo == "" {
		return nil, fmt.Errorf("github_repo is required for provider %s", cfg.Slug)
	}
	if strings.TrimSpace(cfg.GitHubDocsPath) == "" {
		return nil, fmt.Errorf("github_docs_path is required for provider %s", cfg.Slug)
	}

	authToken := githubAuthToken(cfg)
	branch := strings.TrimSpace(cfg.GitHubBranch)
	if branch == "" {
		branch = "main"
	}

	return &GitHubRawFetcher{
		cfg:                cfg,
		client:             newHTTPClient(),
		apiBaseURL:         defaultGitHubAPIBaseURL,
		rawBaseURL:         defaultGitHubRawBaseURL,
		branch:             branch,
		authToken:          authToken,
		minRequestInterval: githubMinRequestInterval(cfg.RateLimit, authToken != ""),
	}, nil
}

func (f *GitHubRawFetcher) Name() string {
	return f.cfg.Name
}

// Fetch retrieves all matching Markdown files from the configured GitHub repo.
func (f *GitHubRawFetcher) Fetch(ctx context.Context) ([]Page, error) {
	files, err := f.discoverFiles(ctx)
	if err != nil {
		return nil, err
	}

	pages := make([]Page, 0, len(files))
	for _, file := range files {
		page, err := f.fetchFile(ctx, file)
		if err != nil {
			return nil, err
		}
		pages = append(pages, *page)
	}

	return pages, nil
}

func (f *GitHubRawFetcher) discoverFiles(ctx context.Context) ([]gitHubFile, error) {
	treeURL := fmt.Sprintf("%s/repos/%s/git/trees/%s?recursive=1", strings.TrimRight(f.apiBaseURL, "/"), f.cfg.GitHubRepo, url.PathEscape(f.branch))
	resp, err := f.doRequest(ctx, treeURL, map[string]string{
		"Accept":               "application/vnd.github+json",
		"X-GitHub-Api-Version": gitHubAPIVersion,
	}, true)
	if err != nil {
		return nil, fmt.Errorf("discovering GitHub tree for %s: %w", f.cfg.Slug, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var payload gitHubTreeResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decoding GitHub tree response: %w", err)
	}
	if payload.Truncated {
		return nil, fmt.Errorf("GitHub tree response was truncated for %s; narrow github_docs_path or use a non-recursive strategy", f.cfg.GitHubRepo)
	}

	files := make([]gitHubFile, 0, len(payload.Tree))
	for _, node := range payload.Tree {
		if node.Type != "blob" {
			continue
		}

		relPath, ok := githubRelativePath(node.Path, f.cfg.GitHubDocsPath)
		if !ok {
			continue
		}
		if !matchesAnyPattern(relPath, f.cfg.Paths) {
			continue
		}

		files = append(files, gitHubFile{
			RepoPath:    node.Path,
			ArchivePath: githubPathToArchivePath(relPath),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].ArchivePath < files[j].ArchivePath
	})
	if len(files) == 0 {
		return nil, fmt.Errorf("no GitHub files matched github_docs_path=%q and paths=%v", f.cfg.GitHubDocsPath, f.cfg.Paths)
	}

	return files, nil
}

func (f *GitHubRawFetcher) fetchFile(ctx context.Context, file gitHubFile) (*Page, error) {
	rawURL := buildGitHubRawURL(f.rawBaseURL, f.cfg.GitHubRepo, f.branch, file.RepoPath)
	resp, err := f.doRequest(ctx, rawURL, nil, false)
	if err != nil {
		return nil, fmt.Errorf("fetching GitHub raw content for %s: %w", file.RepoPath, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading GitHub raw content for %s: %w", file.RepoPath, err)
	}

	return &Page{
		SourceURL: rawURL,
		Path:      file.ArchivePath,
		Content:   body,
	}, nil
}

func (f *GitHubRawFetcher) doRequest(ctx context.Context, rawURL string, headers map[string]string, allowAuth bool) (*http.Response, error) {
	if err := f.waitForTurn(ctx); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", "fularchive/0.1 (+https://github.com/fulmenhq/fularchive)")
	req.Header.Set("Accept", "*/*")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	if allowAuth && f.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+f.authToken)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", rawURL, err)
	}

	if allowAuth && (resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests) {
		err = githubRateLimitError(resp, rawURL, f.cfg.AuthEnvVar)
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		defer func() {
			_ = resp.Body.Close()
		}()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		if len(body) == 0 {
			return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, rawURL)
		}
		return nil, fmt.Errorf("HTTP %d for %s: %s", resp.StatusCode, rawURL, strings.TrimSpace(string(body)))
	}

	return resp, nil
}

func (f *GitHubRawFetcher) waitForTurn(ctx context.Context) error {
	if f.minRequestInterval <= 0 {
		return nil
	}
	if f.lastRequestAt.IsZero() {
		f.lastRequestAt = time.Now()
		return nil
	}

	wait := time.Until(f.lastRequestAt.Add(f.minRequestInterval))
	if wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}

	f.lastRequestAt = time.Now()
	return nil
}

func githubAuthToken(cfg ProviderConfig) string {
	envVar := strings.TrimSpace(cfg.AuthEnvVar)
	if envVar != "" {
		return strings.TrimSpace(os.Getenv(envVar))
	}
	return strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
}

func githubMinRequestInterval(rateLimit *RateLimitConfig, authenticated bool) time.Duration {
	if rateLimit != nil && rateLimit.RequestsPerSecond > 0 {
		return time.Duration(float64(time.Second) / rateLimit.RequestsPerSecond)
	}
	if authenticated {
		return time.Duration(float64(time.Second) / defaultGitHubAuthedRPS)
	}
	return time.Duration(float64(time.Second) / defaultGitHubUnauthenticatedRPS)
}

func githubRateLimitError(resp *http.Response, rawURL, authEnvVar string) error {
	remaining := resp.Header.Get("X-RateLimit-Remaining")
	resource := resp.Header.Get("X-RateLimit-Resource")
	resetHint := ""
	if reset := resp.Header.Get("X-RateLimit-Reset"); reset != "" {
		if unixSeconds, err := strconv.ParseInt(reset, 10, 64); err == nil {
			resetHint = fmt.Sprintf("; resets at %s", time.Unix(unixSeconds, 0).UTC().Format(time.RFC3339))
		}
	}
	if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
		resetHint += fmt.Sprintf("; retry-after=%ss", retryAfter)
	}

	tokenName := strings.TrimSpace(authEnvVar)
	if tokenName == "" {
		tokenName = "GITHUB_TOKEN"
	}

	if remaining == "0" || resource != "" || resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("GitHub API rate limit hit for %s (status=%d, resource=%s, remaining=%s%s); set %s for authenticated access or lower the provider rate_limit", rawURL, resp.StatusCode, resource, remaining, resetHint, tokenName)
	}

	return fmt.Errorf("GitHub API denied %s with HTTP %d%s; set %s for authenticated access if this repo exceeds anonymous limits", rawURL, resp.StatusCode, resetHint, tokenName)
}

func githubRelativePath(repoPath, docsPath string) (string, bool) {
	repoPath = path.Clean(strings.TrimPrefix(repoPath, "/"))
	docsPath = path.Clean(strings.Trim(strings.TrimSpace(docsPath), "/"))
	if docsPath == "." {
		docsPath = ""
	}
	if docsPath == "" {
		return repoPath, true
	}
	if repoPath == docsPath {
		return "", false
	}
	if !strings.HasPrefix(repoPath+"/", docsPath+"/") {
		return "", false
	}
	return strings.TrimPrefix(repoPath, docsPath+"/"), true
}

func githubPathToArchivePath(relPath string) string {
	relPath = strings.TrimPrefix(relPath, "/")
	if relPath == "" {
		return "index.md"
	}
	if path.Base(relPath) == "_index.md" {
		dir := path.Dir(relPath)
		if dir == "." {
			return "index.md"
		}
		return path.Join(dir, "index.md")
	}
	return relPath
}

func buildGitHubRawURL(baseURL, repo, branch, repoPath string) string {
	parts := []string{strings.TrimRight(baseURL, "/")}
	for _, segment := range strings.Split(strings.Trim(repo, "/"), "/") {
		if segment == "" {
			continue
		}
		parts = append(parts, url.PathEscape(segment))
	}
	parts = append(parts, url.PathEscape(branch))
	for _, segment := range strings.Split(strings.Trim(repoPath, "/"), "/") {
		if segment == "" {
			continue
		}
		parts = append(parts, url.PathEscape(segment))
	}
	return strings.Join(parts, "/")
}

func matchesAnyPattern(target string, patterns []string) bool {
	if len(patterns) == 0 {
		return true
	}
	for _, pattern := range patterns {
		matched, err := doublestarMatch(pattern, target)
		if err == nil && matched {
			return true
		}
	}
	return false
}

func doublestarMatch(pattern, target string) (bool, error) {
	pattern = strings.Trim(strings.TrimSpace(pattern), "/")
	target = strings.Trim(strings.TrimSpace(target), "/")
	if pattern == "" {
		return false, nil
	}

	var builder strings.Builder
	builder.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				if i+2 < len(pattern) && pattern[i+2] == '/' {
					builder.WriteString("(?:.*/)?")
					i += 2
				} else {
					builder.WriteString(".*")
					i++
				}
			} else {
				builder.WriteString("[^/]*")
			}
		case '?':
			builder.WriteString("[^/]")
		case '.', '(', ')', '+', '|', '^', '$', '{', '}', '[', ']', '\\':
			builder.WriteByte('\\')
			builder.WriteByte(pattern[i])
		default:
			builder.WriteByte(pattern[i])
		}
	}
	builder.WriteString("$")

	re, err := regexp.Compile(builder.String())
	if err != nil {
		return false, err
	}
	return re.MatchString(target), nil
}
