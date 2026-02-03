package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_DefaultPath(t *testing.T) {
	// Create a temporary config directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create a test config file
	testConfig := `internal_domains:
  - example.com
trusted_domains:
  - partner.com
default_scope: active
dry_run: true
credentials_path: /path/to/credentials.json
`

	if err := os.WriteFile(configPath, []byte(testConfig), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if len(config.InternalDomains) != 1 || config.InternalDomains[0] != "example.com" {
		t.Errorf("Expected internal_domains ['example.com'], got %v", config.InternalDomains)
	}
	if config.DefaultScope != "active" {
		t.Errorf("Expected default_scope 'active', got '%s'", config.DefaultScope)
	}
}

func TestLoadConfig_NonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nonexistent.yaml")

	// Should not error, should use defaults
	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() should not error on non-existent file: %v", err)
	}

	if config == nil {
		t.Fatal("Expected config to be returned even for non-existent file")
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	config := &Config{
		InternalDomains: []string{"example.com"},
		TrustedDomains:  []string{"partner.com"},
		DefaultScope:    "active",
		DryRun:          true,
		CredentialsPath: "/path/to/credentials.json",
	}

	if err := SaveConfig(config, configPath); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load it back and verify
	loadedConfig, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if len(loadedConfig.InternalDomains) != 1 || loadedConfig.InternalDomains[0] != "example.com" {
		t.Errorf("Expected internal_domains ['example.com'], got %v", loadedConfig.InternalDomains)
	}
}
