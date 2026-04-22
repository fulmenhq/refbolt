package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fulmenhq/gofulmen/logging"
	"github.com/fulmenhq/refbolt/internal/provider"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const envPrefix = "REFBOLT"

var (
	cfg        *viper.Viper
	log        *logging.Logger
	configUsed string // resolved config source for reporting
)

// LoadOptions controls how configuration is loaded.
type LoadOptions struct {
	// ConfigPath is the resolved config file path. Empty means use embedded catalog.
	ConfigPath string
	// Strict enables strict validation (errors instead of warnings).
	// Used by `refbolt validate`.
	Strict bool
	// UseEmbedded forces use of the embedded catalog even if ConfigPath is set.
	// Applies archive_root override for local CLI use.
	UseEmbedded bool
	// Verbose toggles debug-level logging on the package-wide logger so
	// existing log.Debug(...) calls surface. Set from the `--verbose/-v`
	// persistent flag on the root command (FA-111 item #8).
	Verbose bool
}

// Load initializes configuration from the resolved config source and env vars.
// The caller (root command) is responsible for config path resolution.
func Load(opts LoadOptions) error {
	var err error
	log, err = logging.NewCLI("refbolt")
	if err != nil {
		return fmt.Errorf("initializing logger: %w", err)
	}
	if opts.Verbose {
		log.SetLevel(logging.DEBUG)
	}

	cfg = viper.New()

	// Defaults
	cfg.SetDefault("archive_root", "/data/archive")

	// Env prefix: REFBOLT_
	cfg.SetEnvPrefix(envPrefix)
	cfg.AutomaticEnv()
	cfg.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	if opts.UseEmbedded || opts.ConfigPath == "" {
		// Fall back to embedded catalog.
		if len(embeddedCatalog) == 0 {
			log.Debug("No embedded catalog available and no config file specified")
			return nil
		}
		cfg.SetConfigType("yaml")
		if err := cfg.ReadConfig(strings.NewReader(string(embeddedCatalog))); err != nil {
			return fmt.Errorf("reading embedded catalog: %w", err)
		}
		// Override archive_root for local CLI use — /data/archive is a container path.
		if os.Getenv("REFBOLT_ARCHIVE_ROOT") == "" {
			cfg.Set("archive_root", "./archive")
		}
		configUsed = "(embedded catalog)"
		log.Debug("Using embedded provider catalog")
	} else {
		cfg.SetConfigFile(opts.ConfigPath)
		cfg.SetConfigType("yaml")

		if err := cfg.ReadInConfig(); err != nil {
			// An explicit config path that doesn't exist is an error,
			// not a graceful fallback.
			return fmt.Errorf("reading config %s: %w", opts.ConfigPath, err)
		}
		configUsed = cfg.ConfigFileUsed()
		log.Info(fmt.Sprintf("Loaded config from %s", configUsed))
	}

	// Validate against embedded schema.
	if opts.Strict {
		return validateStrict()
	}
	validatePermissive()
	return nil
}

// ConfigUsed returns a description of the config source that was loaded.
func ConfigUsed() string {
	return configUsed
}

// validatePermissive logs schema diagnostics as warnings but does not fail.
// Used during normal startup (sync, version, etc.).
func validatePermissive() {
	diags := runSchemaValidation()
	for _, d := range diags {
		log.Warn(fmt.Sprintf("Config validation: %s", d))
	}
}

// validateStrict returns an error if any schema or catalog diagnostics are found.
// Used by `refbolt validate` to provide deterministic exit codes.
func validateStrict() error {
	diags := runSchemaValidation()

	// Cross-check provider slugs against embedded catalog.
	catalogDiags := checkCatalogSlugs()
	diags = append(diags, catalogDiags...)

	if len(diags) == 0 {
		return nil
	}
	var b strings.Builder
	for _, d := range diags {
		fmt.Fprintf(&b, "  %s\n", d)
	}
	return fmt.Errorf("config validation failed:\n%s", b.String())
}

// checkCatalogSlugs verifies that all provider slugs in the loaded config
// exist in the embedded catalog. Users cannot invent providers in v0.0.2.
func checkCatalogSlugs() []string {
	if len(embeddedCatalog) == 0 {
		return nil
	}

	catalogTopics, err := CatalogTopics()
	if err != nil {
		return nil
	}

	// Build catalog slug set.
	catalogSlugs := make(map[string]bool)
	for _, t := range catalogTopics {
		for _, p := range t.Providers {
			catalogSlugs[p.Slug] = true
		}
	}

	// Check loaded config slugs.
	var diags []string
	for _, t := range Topics() {
		for _, p := range t.Providers {
			if !catalogSlugs[p.Slug] {
				diags = append(diags, fmt.Sprintf("unknown provider slug %q (not in embedded catalog)", p.Slug))
			}
		}
	}
	return diags
}

// runSchemaValidation validates the current config against the embedded schema.
// Returns a list of human-readable diagnostic strings.
func runSchemaValidation() []string {
	if len(embeddedSchema) == 0 {
		return nil
	}

	// Get the raw config bytes for validation.
	var configBytes []byte
	if configUsed == "(embedded catalog)" {
		configBytes = embeddedCatalog
	} else if configUsed != "" {
		var err error
		configBytes, err = os.ReadFile(configUsed)
		if err != nil {
			return []string{fmt.Sprintf("cannot read config for validation: %v", err)}
		}
	} else {
		return nil
	}

	// Parse YAML to generic structure, then convert to JSON for schema validation.
	var data interface{}
	if err := yaml.Unmarshal(configBytes, &data); err != nil {
		return []string{fmt.Sprintf("invalid YAML: %v", err)}
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return []string{fmt.Sprintf("YAML→JSON conversion failed: %v", err)}
	}

	// Convert YAML schema to JSON for the JSON Schema compiler.
	var schemaData interface{}
	if err := yaml.Unmarshal(embeddedSchema, &schemaData); err != nil {
		return []string{fmt.Sprintf("schema parse failed: %v", err)}
	}
	schemaJSON, err := json.Marshal(schemaData)
	if err != nil {
		return []string{fmt.Sprintf("schema YAML→JSON conversion failed: %v", err)}
	}

	// Compile and validate against embedded schema.
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("schema.json", strings.NewReader(string(schemaJSON))); err != nil {
		return []string{fmt.Sprintf("schema compilation failed: %v", err)}
	}
	sch, err := compiler.Compile("schema.json")
	if err != nil {
		return []string{fmt.Sprintf("schema compilation failed: %v", err)}
	}

	var rawJSON interface{}
	if err := json.Unmarshal(jsonBytes, &rawJSON); err != nil {
		return []string{fmt.Sprintf("JSON parse failed: %v", err)}
	}

	if err := sch.Validate(rawJSON); err != nil {
		var validationErr *jsonschema.ValidationError
		if errors.As(err, &validationErr) {
			var diags []string
			for _, cause := range validationErr.Causes {
				diags = append(diags, fmt.Sprintf("%s: %s", cause.InstanceLocation, cause.Message))
			}
			if len(diags) == 0 {
				diags = append(diags, validationErr.Message)
			}
			return diags
		}
		return []string{err.Error()}
	}

	return nil
}

// ResolveConfigPath implements the config resolution chain:
//  1. explicit flag path (if non-empty)
//  2. REFBOLT_CONFIG env var
//  3. ./providers.yaml (CWD)
//  4. ~/.config/refbolt/providers.yaml (XDG)
//  5. "" (empty = use embedded catalog)
func ResolveConfigPath(flagPath string) string {
	// 1. Explicit flag.
	if flagPath != "" {
		return flagPath
	}

	// 2. Env var.
	if envPath := os.Getenv("REFBOLT_CONFIG"); envPath != "" {
		return envPath
	}

	// 3. CWD.
	if _, err := os.Stat("providers.yaml"); err == nil {
		return "providers.yaml"
	}

	// 4. XDG.
	home, err := os.UserHomeDir()
	if err == nil {
		xdg := filepath.Join(home, ".config", "refbolt", "providers.yaml")
		if _, err := os.Stat(xdg); err == nil {
			return xdg
		}
	}

	// 5. Embedded catalog fallback.
	return ""
}

// Logger returns the package-level logger for use by other packages.
func Logger() *logging.Logger {
	return log
}

// ArchiveRoot returns the base URI for the archive tree.
func ArchiveRoot() string {
	return cfg.GetString("archive_root")
}

// TopicSlugs returns the list of configured topic slugs.
func TopicSlugs() []string {
	topics := cfg.Get("topics")
	if topics == nil {
		return nil
	}
	slice, ok := topics.([]interface{})
	if !ok {
		return nil
	}
	var slugs []string
	for _, t := range slice {
		if m, ok := t.(map[string]interface{}); ok {
			if s, ok := m["slug"].(string); ok {
				slugs = append(slugs, s)
			}
		}
	}
	return slugs
}

// Topic holds a parsed topic from config with its providers.
// Name and Description are optional in the schema; when absent, callers
// that need a display label should fall back to Slug.
type Topic struct {
	Slug        string
	Name        string
	Description string
	Providers   []provider.ProviderConfig
}

// Topics returns all configured topics with fully typed provider configs.
func Topics() []Topic {
	raw := cfg.Get("topics")
	if raw == nil {
		return nil
	}
	slice, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	var topics []Topic
	for _, t := range slice {
		tm, ok := t.(map[string]interface{})
		if !ok {
			continue
		}
		topic := Topic{
			Slug:        stringVal(tm, "slug"),
			Name:        stringVal(tm, "name"),
			Description: stringVal(tm, "description"),
		}
		rawProviders, ok := tm["providers"].([]interface{})
		if !ok {
			continue
		}
		for _, p := range rawProviders {
			pm, ok := p.(map[string]interface{})
			if !ok {
				continue
			}
			pc := provider.ProviderConfig{
				Slug:           stringVal(pm, "slug"),
				Name:           stringVal(pm, "name"),
				BaseURL:        stringVal(pm, "base_url"),
				FetchStrategy:  provider.FetchStrategy(stringVal(pm, "fetch_strategy")),
				LLMSTxtURL:     stringVal(pm, "llms_txt_url"),
				OpenAPIURL:     stringVal(pm, "openapi_url"),
				GitHubRepo:     stringVal(pm, "github_repo"),
				GitHubDocsPath: stringVal(pm, "github_docs_path"),
				GitHubBranch:   stringVal(pm, "github_branch"),
				AuthEnvVar:     stringVal(pm, "auth_env_var"),
				FetchTimeout:   durationVal(pm, "fetch_timeout"),
				Enabled:        boolPtrVal(pm, "enabled"),
			}
			if rm, ok := pm["rate_limit"].(map[string]interface{}); ok {
				rl := provider.RateLimitConfig{
					RequestsPerSecond: floatVal(rm, "requests_per_second"),
					Burst:             intVal(rm, "burst"),
				}
				if rl.RequestsPerSecond > 0 || rl.Burst > 0 {
					pc.RateLimit = &rl
				}
			}
			if paths, ok := pm["paths"].([]interface{}); ok {
				for _, path := range paths {
					if s, ok := path.(string); ok {
						pc.Paths = append(pc.Paths, s)
					}
				}
			}
			topic.Providers = append(topic.Providers, pc)
		}
		topics = append(topics, topic)
	}
	return topics
}

// CatalogTopics returns topics from the embedded catalog (for init command
// and any other caller that needs a pure catalog view, independent of user
// config).
func CatalogTopics() ([]Topic, error) {
	if len(embeddedCatalog) == 0 {
		return nil, fmt.Errorf("no embedded catalog available")
	}
	return parseCatalogBytes(embeddedCatalog)
}

// parseCatalogBytes parses a catalog YAML blob into a []Topic without
// disturbing the package-level cfg. Kept private so every catalog-bytes
// consumer funnels through the same codepath — avoids the cfg-swap footgun
// creeping into multiple callers.
func parseCatalogBytes(data []byte) ([]Topic, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(strings.NewReader(string(data))); err != nil {
		return nil, fmt.Errorf("parsing catalog: %w", err)
	}

	// Temporarily swap cfg to parse topics, then restore.
	oldCfg := cfg
	cfg = v
	topics := Topics()
	cfg = oldCfg

	return topics, nil
}

func stringVal(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func boolPtrVal(m map[string]interface{}, key string) *bool {
	v, ok := m[key].(bool)
	if !ok {
		return nil
	}
	b := v
	return &b
}

func floatVal(m map[string]interface{}, key string) float64 {
	switch v := m[key].(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case uint:
		return float64(v)
	case uint64:
		return float64(v)
	default:
		return 0
	}
}

func durationVal(m map[string]interface{}, key string) time.Duration {
	s, ok := m[key].(string)
	if !ok || s == "" {
		return 0
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0
	}
	return d
}

func intVal(m map[string]interface{}, key string) int {
	switch v := m[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case uint:
		return int(v)
	case uint64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	default:
		return 0
	}
}

// Providers returns all provider slugs across all topics (flat list).
func Providers() []string {
	topics := cfg.Get("topics")
	if topics == nil {
		return nil
	}
	slice, ok := topics.([]interface{})
	if !ok {
		return nil
	}
	var slugs []string
	for _, t := range slice {
		tm, ok := t.(map[string]interface{})
		if !ok {
			continue
		}
		providers, ok := tm["providers"].([]interface{})
		if !ok {
			continue
		}
		for _, p := range providers {
			if pm, ok := p.(map[string]interface{}); ok {
				if s, ok := pm["slug"].(string); ok {
					slugs = append(slugs, s)
				}
			}
		}
	}
	return slugs
}
