package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// LoggingConfig holds file logging settings
type LoggingConfig struct {
	Enabled  bool   `yaml:"enabled" mapstructure:"enabled"`
	FilePath string `yaml:"file_path" mapstructure:"file_path"`
}

// Config represents the application configuration
type Config struct {
	InternalDomains []string      `yaml:"internal_domains" mapstructure:"internal_domains"`
	TrustedDomains  []string      `yaml:"trusted_domains" mapstructure:"trusted_domains"`
	IncludedDrives  []string      `yaml:"included_drives" mapstructure:"included_drives"` // Whitelist: only scan these drives
	ExcludedDrives  []string      `yaml:"excluded_drives" mapstructure:"excluded_drives"` // Blacklist: skip these drives
	DefaultScope    string        `yaml:"default_scope" mapstructure:"default_scope"`
	DryRun          bool          `yaml:"dry_run" mapstructure:"dry_run"`
	CredentialsPath string        `yaml:"credentials_path" mapstructure:"credentials_path"`
	ImpersonateUser string        `yaml:"impersonate_user" mapstructure:"impersonate_user"`
	Logging         LoggingConfig `yaml:"logging" mapstructure:"logging"`
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	defaultCredentialsPath := filepath.Join(homeDir, ".gdaudit", "credentials.json")
	defaultLogPath := filepath.Join(homeDir, ".gdaudit", "audit.log")

	return &Config{
		InternalDomains: []string{},
		TrustedDomains:  []string{},
		IncludedDrives:  []string{}, // Empty = all drives included
		ExcludedDrives:  []string{},
		DefaultScope:    "shared-drives",
		DryRun:          true, // Safety default
		CredentialsPath: defaultCredentialsPath,
		ImpersonateUser: "",
		Logging: LoggingConfig{
			Enabled:  false,
			FilePath: defaultLogPath,
		},
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.DefaultScope == "" {
		return fmt.Errorf("default_scope cannot be empty")
	}

	validScopes := map[string]bool{
		"active":        true,
		"suspended":     true,
		"shared-drives": true,
	}

	if !validScopes[c.DefaultScope] && !isUserScope(c.DefaultScope) {
		return fmt.Errorf("invalid default_scope: %s (must be 'active', 'suspended', 'shared-drives', or 'user:<email>')", c.DefaultScope)
	}

	if c.CredentialsPath == "" {
		return fmt.Errorf("credentials_path cannot be empty")
	}

	if c.Logging.Enabled && c.Logging.FilePath == "" {
		return fmt.Errorf("logging.file_path cannot be empty when logging.enabled is true")
	}

	return nil
}

// isUserScope checks if the scope is a user scope (user:email@domain.com)
func isUserScope(scope string) bool {
	return len(scope) > 5 && scope[:5] == "user:"
}

// GetConfigDir returns the default configuration directory
func GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".gdaudit"), nil
}

// GetConfigPath returns the default configuration file path
func GetConfigPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "config.yaml"), nil
}
