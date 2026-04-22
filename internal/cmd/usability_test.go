package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/fulmenhq/refbolt/internal/config"
)

// captureStdStreams runs fn with os.Stdout and os.Stderr redirected to
// pipes, returning the captured output. Existing init/validate commands
// print via fmt.Printf / fmt.Fprintln(os.Stderr, ...) — not through
// cmd.OutOrStdout() — so rootCmd.SetOut doesn't capture them. Use this
// for tests that need to assert on those streams.
func captureStdStreams(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()
	origOut, origErr := os.Stdout, os.Stderr
	outR, outW, _ := os.Pipe()
	errR, errW, _ := os.Pipe()
	os.Stdout, os.Stderr = outW, errW
	done := make(chan struct{})
	var outBuf, errBuf bytes.Buffer
	go func() {
		_, _ = io.Copy(&outBuf, outR)
		close(done)
	}()
	errDone := make(chan struct{})
	go func() {
		_, _ = io.Copy(&errBuf, errR)
		close(errDone)
	}()

	fn()

	_ = outW.Close()
	_ = errW.Close()
	<-done
	<-errDone
	os.Stdout, os.Stderr = origOut, origErr
	return outBuf.String(), errBuf.String()
}

// FA-111 regression tests. One file, all the first-run usability
// behaviors in one place so later reviewers can see the intent at a
// glance and easily spot drift if one of these silently regresses.

// TestUsability_NoSelectorSyncErrorIncludesHintBlock locks in item #4.
// The message must be multi-line with concrete next-step commands so
// a brand-new user isn't stuck after typing `refbolt sync`.
func TestUsability_NoSelectorSyncErrorIncludesHintBlock(t *testing.T) {
	setupCatalogFixture(t)
	clearCatalogFlags(t)
	// Prior tests (e.g. TestInitCmd_RealCatalog_RoundTripsValid) leave
	// configFlag pointing at a cleaned-up tempdir path, which would cause
	// sync to fail with a file-open error before reaching the no-selector
	// check we're testing. Reset explicitly.
	configFlag = ""
	t.Cleanup(func() { configFlag = "" })
	// Also reset sync-specific flag globals that prior sync tests may set.
	syncAll = false
	syncForce = false
	providerSlugs = nil
	topicSlugs = nil
	excludeProviders = nil
	t.Cleanup(func() {
		syncAll = false
		syncForce = false
		providerSlugs = nil
		topicSlugs = nil
		excludeProviders = nil
	})

	_, _, err := runCatalog(t, "sync")
	if err == nil {
		t.Fatal("expected error for sync with no selector")
	}
	msg := err.Error()
	// Spot-check: must include the three sync flags and the catalog list hint.
	for _, frag := range []string{
		"--all",
		"--topic",
		"--provider",
		"refbolt catalog list",
	} {
		if !strings.Contains(msg, frag) {
			t.Errorf("sync no-selector error missing %q, got: %q", frag, msg)
		}
	}
}

// TestUsability_InitSeedFlowTipOnStdoutMode locks in item #6. When
// `refbolt init --all` runs without --output, stderr should include
// the one-line tip pointing at --output.
func TestUsability_InitSeedFlowTipOnStdoutMode(t *testing.T) {
	setupCatalogFixture(t)

	_, stderr := captureStdStreams(t, func() {
		initAll = true
		initOutput = ""
		t.Cleanup(func() {
			initAll = false
			initOutput = ""
			rootCmd.SetArgs(nil)
		})
		rootCmd.SetArgs([]string{"init", "--all"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("init --all stdout mode: %v", err)
		}
	})
	if !strings.Contains(stderr, "--output providers.yaml") {
		t.Errorf("stdout-mode init should emit the seed-flow tip on stderr, got: %q", stderr)
	}
}

// TestUsability_InitSeedFlowTipSuppressedWithOutput — opposite side of
// item #6: `refbolt init --all --output <file>` must NOT emit the tip,
// since the user already knows the pattern.
func TestUsability_InitSeedFlowTipSuppressedWithOutput(t *testing.T) {
	setupCatalogFixture(t)

	outPath := t.TempDir() + "/providers.yaml"
	_, stderr := captureStdStreams(t, func() {
		initAll = true
		initOutput = outPath
		initForce = false
		t.Cleanup(func() {
			initAll = false
			initOutput = ""
			initForce = false
			rootCmd.SetArgs(nil)
		})
		rootCmd.SetArgs([]string{"init", "--all", "--output", outPath})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("init --all --output: %v", err)
		}
	})
	if strings.Contains(stderr, "rerun with --output") {
		t.Errorf("--output mode should not emit seed-flow tip, got: %q", stderr)
	}
}

// TestUsability_ValidateCustomizationTipOnEmbeddedCatalog covers item #11.
// When validate falls back to the embedded catalog, stderr should carry
// the `refbolt init --all --output providers.yaml` tip.
func TestUsability_ValidateCustomizationTipOnEmbeddedCatalog(t *testing.T) {
	setupCatalogFixture(t)

	stdout, stderr := captureStdStreams(t, func() {
		configFlag = ""
		t.Cleanup(func() {
			configFlag = ""
			rootCmd.SetArgs(nil)
		})
		rootCmd.SetArgs([]string{"validate"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("validate against embedded catalog: %v", err)
		}
	})
	// stdout reports the embedded catalog source (existing behavior,
	// not changed by FA-111 — kept as a sanity check).
	if !strings.Contains(stdout, "(embedded catalog)") {
		t.Errorf("validate stdout should mention (embedded catalog), got: %q", stdout)
	}
	// stderr customization tip (FA-111 addition).
	if !strings.Contains(stderr, "refbolt init --all --output providers.yaml") {
		t.Errorf("validate on embedded catalog should emit customization tip, got stderr: %q", stderr)
	}
}

// TestUsability_CatalogShowIncludesCredentialURL covers item #3 on the
// catalog surface. `catalog show openai` should include the Jina URL
// because openai uses the jina strategy.
func TestUsability_CatalogShowIncludesCredentialURL(t *testing.T) {
	setupCatalogFixture(t)
	clearCatalogFlags(t)

	stdout, _, err := runCatalog(t, "catalog", "show", "openai")
	if err != nil {
		t.Fatalf("catalog show openai: %v", err)
	}
	if !strings.Contains(stdout, "jina.ai/reader") {
		t.Errorf("catalog show openai should include jina.ai/reader URL, got: %q", stdout)
	}
}

// TestUsability_CatalogShowOmitsURLForProvidersWithNoCreds covers the
// graceful-omit path. `catalog show anthropic` uses native strategy,
// no credentials required — must not include any get-a-key URL.
func TestUsability_CatalogShowOmitsURLForProvidersWithNoCreds(t *testing.T) {
	setupCatalogFixture(t)
	clearCatalogFlags(t)

	stdout, _, err := runCatalog(t, "catalog", "show", "anthropic")
	if err != nil {
		t.Fatalf("catalog show anthropic: %v", err)
	}
	// Expect "Credentials:     none required"
	if !strings.Contains(stdout, "none required") {
		t.Errorf("catalog show anthropic should say 'none required', got: %q", stdout)
	}
	// Must NOT contain URL substrings reserved for credentialed providers.
	for _, forbidden := range []string{"jina.ai/reader", "github.com/settings"} {
		if strings.Contains(stdout, forbidden) {
			t.Errorf("catalog show anthropic should not include %q, got: %q", forbidden, stdout)
		}
	}
}

// TestUsability_CredentialURLHelper is a small unit test for the
// `config.CredentialURL` helper — ensures known env vars map to URLs
// and unknown env vars return empty string (so callers can skip the
// "Get a key" suffix cleanly).
func TestUsability_CredentialURLHelper(t *testing.T) {
	cases := []struct {
		envVar string
		want   string
	}{
		{"JINA_API_KEY", "https://jina.ai/reader"},
		{"GITHUB_TOKEN", "https://github.com/settings/tokens"},
		{"UNKNOWN_CRED", ""},
		{"", ""},
	}
	for _, tc := range cases {
		got := config.CredentialURL(tc.envVar)
		if got != tc.want {
			t.Errorf("CredentialURL(%q) = %q, want %q", tc.envVar, got, tc.want)
		}
	}
}

// TestUsability_VersionFlag covers item #7 — `refbolt --version` must
// print the same style as `refbolt version`.
//
// Note: cobra only wires `--version` when `rootCmd.Version` is non-empty.
// At runtime that's set via main.SetVersionInfo; tests need to call it
// explicitly to exercise the flag.
func TestUsability_VersionFlag(t *testing.T) {
	setupCatalogFixture(t)
	SetVersionInfo("0.0.4-test", "testcommit", "2026-04-22")
	t.Cleanup(func() {
		SetVersionInfo("dev", "unknown", "unknown")
		rootCmd.Version = ""
		rootCmd.SetVersionTemplate("")
	})

	var outBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetArgs([]string{"--version"})
	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		rootCmd.SetOut(nil)
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("--version: %v", err)
	}
	out := outBuf.String()
	if !strings.Contains(out, "refbolt 0.0.4-test") {
		t.Errorf("--version output should contain 'refbolt 0.0.4-test', got: %q", out)
	}
	if !strings.Contains(out, "commit:") || !strings.Contains(out, "built:") {
		t.Errorf("--version output should match version subcommand template (commit: / built:), got: %q", out)
	}
}

// TestUsability_PluralizeHelper covers the shared pluralize helper in
// plural.go. Small, fast, and the helper is called from three surfaces
// so a silent regression would ripple widely.
func TestUsability_PluralizeHelper(t *testing.T) {
	cases := []struct {
		n        int
		singular string
		plural   string
		want     string
	}{
		{1, "provider", "providers", "provider"},
		{0, "provider", "providers", "providers"},
		{2, "provider", "providers", "providers"},
		{1, "topic", "topics", "topic"},
		{7, "topic", "topics", "topics"},
	}
	for _, tc := range cases {
		got := pluralize(tc.n, tc.singular, tc.plural)
		if got != tc.want {
			t.Errorf("pluralize(%d, %q, %q) = %q, want %q", tc.n, tc.singular, tc.plural, got, tc.want)
		}
	}
}
