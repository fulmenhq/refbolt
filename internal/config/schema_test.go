package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/fulmenhq/gofulmen/schema"
	"gopkg.in/yaml.v3"
)

func TestProvidersSchema_MetaValidation(t *testing.T) {
	root := findProjectRoot(t)
	schemaPath := filepath.Join(root, "schemas", "providers", "v0", "providers.schema.yaml")

	raw, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("Failed to read schema file: %v", err)
	}

	// Convert YAML schema to JSON for meta-validation.
	var schemaData interface{}
	if err := yaml.Unmarshal(raw, &schemaData); err != nil {
		t.Fatalf("Failed to parse schema YAML: %v", err)
	}

	jsonBytes, err := json.Marshal(schemaData)
	if err != nil {
		t.Fatalf("Failed to convert schema to JSON: %v", err)
	}

	diags, err := schema.ValidateSchemaBytes(jsonBytes)
	if err != nil {
		t.Fatalf("Meta-validation error: %v", err)
	}

	for _, d := range diags {
		t.Errorf("Schema diagnostic: %s: %s", d.Pointer, d.Message)
	}
}

func TestProvidersSchema_ValidatesConfigFile(t *testing.T) {
	root := findProjectRoot(t)
	schemaPath := filepath.Join(root, "schemas", "providers", "v0", "providers.schema.yaml")
	configPath := filepath.Join(root, "configs", "providers.yaml")

	// Load and compile the schema.
	schemaRaw, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("Failed to read schema: %v", err)
	}

	var schemaData interface{}
	if err := yaml.Unmarshal(schemaRaw, &schemaData); err != nil {
		t.Fatalf("Failed to parse schema YAML: %v", err)
	}

	schemaJSON, err := json.Marshal(schemaData)
	if err != nil {
		t.Fatalf("Failed to convert schema to JSON: %v", err)
	}

	validator, err := schema.NewValidator(schemaJSON)
	if err != nil {
		t.Fatalf("Failed to compile schema: %v", err)
	}

	// Load the config file.
	configRaw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	var configData interface{}
	if err := yaml.Unmarshal(configRaw, &configData); err != nil {
		t.Fatalf("Failed to parse config YAML: %v", err)
	}

	// Validate config against schema.
	diags, err := validator.ValidateData(configData)
	if err != nil {
		t.Fatalf("Validation error: %v", err)
	}

	for _, d := range diags {
		t.Errorf("Config validation: %s: %s", d.Pointer, d.Message)
	}
}
