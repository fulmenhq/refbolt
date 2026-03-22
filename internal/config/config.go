package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fulmenhq/gofulmen/logging"
	"github.com/fulmenhq/gofulmen/schema"
	"github.com/fulmenhq/refbolt/internal/provider"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const (
	schemaID  = "providers/v0/providers"
	envPrefix = "REFBOLT"
)

var (
	cfg *viper.Viper
	log *logging.Logger
)

// Load initializes configuration from defaults, config file, and env vars.
// If a config file is found, it is validated against the providers schema.
func Load() error {
	var err error
	log, err = logging.NewCLI("refbolt")
	if err != nil {
		return fmt.Errorf("initializing logger: %w", err)
	}

	cfg = viper.New()

	// Defaults
	cfg.SetDefault("archive_root", "/data/archive")

	// Env prefix: REFBOLT_
	cfg.SetEnvPrefix(envPrefix)
	cfg.AutomaticEnv()
	cfg.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	// Config file (optional override)
	configPath := os.Getenv("REFBOLT_CONFIG")
	if configPath == "" {
		configPath = filepath.Join("configs", "providers.yaml")
	}
	cfg.SetConfigFile(configPath)
	cfg.SetConfigType("yaml")

	if err := cfg.ReadInConfig(); err != nil {
		// No config file is fine — use defaults + env vars.
		var notFound viper.ConfigFileNotFoundError
		var pathErr *os.PathError
		if !errors.As(err, &notFound) && !errors.As(err, &pathErr) {
			return fmt.Errorf("reading config: %w", err)
		}
		log.Debug("No config file found, using defaults + env vars")
		return nil
	}

	log.Info(fmt.Sprintf("Loaded config from %s", cfg.ConfigFileUsed()))

	// Validate against schema if schemas directory exists.
	if err := validateConfig(cfg.ConfigFileUsed()); err != nil {
		return fmt.Errorf("config validation: %w", err)
	}

	return nil
}

// validateConfig runs the loaded config file through the providers schema.
func validateConfig(configPath string) error {
	schemasDir := findSchemasDir()
	if schemasDir == "" {
		log.Debug("No schemas/ directory found, skipping validation")
		return nil
	}

	catalog := schema.NewCatalog(schemasDir)

	// Read the config file as YAML, then convert to JSON for schema validation.
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("reading config for validation: %w", err)
	}

	var data interface{}
	if err := yaml.Unmarshal(raw, &data); err != nil {
		return fmt.Errorf("parsing config YAML: %w", err)
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("converting config to JSON: %w", err)
	}

	diags, err := catalog.ValidateDataByID(schemaID, jsonBytes)
	if err != nil {
		// Schema not found in catalog is non-fatal — log and continue.
		log.Debug(fmt.Sprintf("Schema validation skipped: %v", err))
		return nil
	}

	for _, d := range diags {
		log.Warn(fmt.Sprintf("Config validation: %s: %s", d.Pointer, d.Message))
	}

	return nil
}

// findSchemasDir walks up from cwd looking for a schemas/ directory.
func findSchemasDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		candidate := filepath.Join(dir, "schemas")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
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
type Topic struct {
	Slug      string
	Providers []provider.ProviderConfig
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
			Slug: stringVal(tm, "slug"),
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
