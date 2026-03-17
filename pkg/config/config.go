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

// GoogleConfig holds Google Drive-specific configuration
type GoogleConfig struct {
	CredentialsPath string   `yaml:"credentials_path" mapstructure:"credentials_path"`
	ImpersonateUser string   `yaml:"impersonate_user" mapstructure:"impersonate_user"`
	IncludedDrives  []string `yaml:"included_drives" mapstructure:"included_drives"`
	ExcludedDrives  []string `yaml:"excluded_drives" mapstructure:"excluded_drives"`
}

// MicrosoftConfig holds Microsoft 365 (SharePoint/OneDrive) configuration
type MicrosoftConfig struct {
	TenantID     string `yaml:"tenant_id" mapstructure:"tenant_id"`
	ClientID     string `yaml:"client_id" mapstructure:"client_id"`
	ClientSecret string `yaml:"client_secret" mapstructure:"client_secret"`
}

// SlackConfig holds Slack configuration
type SlackConfig struct {
	BotToken  string `yaml:"bot_token" mapstructure:"bot_token"`
	UserToken string `yaml:"user_token" mapstructure:"user_token"`
}

// Config represents the application configuration
type Config struct {
	// Provider-agnostic settings
	InternalDomains []string      `yaml:"internal_domains" mapstructure:"internal_domains"`
	TrustedDomains  []string      `yaml:"trusted_domains" mapstructure:"trusted_domains"`
	DefaultScope    string        `yaml:"default_scope" mapstructure:"default_scope"`
	DryRun          bool          `yaml:"dry_run" mapstructure:"dry_run"`
	Logging         LoggingConfig `yaml:"logging" mapstructure:"logging"`
	DefaultProvider string        `yaml:"default_provider" mapstructure:"default_provider"`

	// Provider-specific configs
	Google    GoogleConfig    `yaml:"google" mapstructure:"google"`
	Microsoft MicrosoftConfig `yaml:"microsoft" mapstructure:"microsoft"`
	Slack     SlackConfig     `yaml:"slack" mapstructure:"slack"`

	// Deprecated: kept for backward compatibility. Migrated to Google sub-config.
	CredentialsPath string   `yaml:"credentials_path" mapstructure:"credentials_path"`
	ImpersonateUser string   `yaml:"impersonate_user" mapstructure:"impersonate_user"`
	IncludedDrives  []string `yaml:"included_drives" mapstructure:"included_drives"`
	ExcludedDrives  []string `yaml:"excluded_drives" mapstructure:"excluded_drives"`
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	defaultCredentialsPath := filepath.Join(homeDir, ".dplense", "credentials.json")
	defaultLogPath := filepath.Join(homeDir, ".dplense", "audit.log")

	return &Config{
		InternalDomains: []string{},
		TrustedDomains:  []string{},
		DefaultScope:    "shared-drives",
		DryRun:          true, // Safety default
		DefaultProvider: "google",
		CredentialsPath: defaultCredentialsPath,
		ImpersonateUser: "",
		IncludedDrives:  []string{},
		ExcludedDrives:  []string{},
		Logging: LoggingConfig{
			Enabled:  false,
			FilePath: defaultLogPath,
		},
	}
}

// MigrateGoogleConfig copies deprecated top-level Google fields into the
// nested Google config, if the nested fields are empty. This ensures
// backward compatibility with existing config files.
func (c *Config) MigrateGoogleConfig() {
	if c.Google.CredentialsPath == "" && c.CredentialsPath != "" {
		c.Google.CredentialsPath = c.CredentialsPath
	}
	if c.Google.ImpersonateUser == "" && c.ImpersonateUser != "" {
		c.Google.ImpersonateUser = c.ImpersonateUser
	}
	if len(c.Google.IncludedDrives) == 0 && len(c.IncludedDrives) > 0 {
		c.Google.IncludedDrives = c.IncludedDrives
	}
	if len(c.Google.ExcludedDrives) == 0 && len(c.ExcludedDrives) > 0 {
		c.Google.ExcludedDrives = c.ExcludedDrives
	}
}

// ResolveProvider returns the effective provider name.
func (c *Config) ResolveProvider(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if c.DefaultProvider != "" {
		return c.DefaultProvider
	}
	return "google"
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.DefaultScope == "" {
		return fmt.Errorf("default_scope cannot be empty")
	}

	// Scope validation is now provider-specific; only validate basic format here
	validScopes := map[string]bool{
		"active":        true,
		"suspended":     true,
		"shared-drives": true,
		"sharepoint":    true,
		"onedrive":      true,
		"all":           true,
		"files":         true,
		"channels":      true,
		"guests":        true,
	}

	if !validScopes[c.DefaultScope] && !isUserScope(c.DefaultScope) && !isSiteScope(c.DefaultScope) {
		return fmt.Errorf("invalid default_scope: %s", c.DefaultScope)
	}

	// Credentials are only required for google provider when it's the default
	effectiveProvider := c.DefaultProvider
	if effectiveProvider == "" {
		effectiveProvider = "google"
	}
	if effectiveProvider == "google" {
		creds := c.Google.CredentialsPath
		if creds == "" {
			creds = c.CredentialsPath
		}
		if creds == "" {
			return fmt.Errorf("credentials_path cannot be empty for google provider")
		}
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

// isSiteScope checks if the scope is a site scope (site:url)
func isSiteScope(scope string) bool {
	return len(scope) > 5 && scope[:5] == "site:"
}

// GetConfigDir returns the default configuration directory
func GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".dplense"), nil
}

// GetConfigPath returns the default configuration file path
func GetConfigPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "config.yaml"), nil
}
