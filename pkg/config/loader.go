package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// LoadConfig loads configuration from the specified path or default location
func LoadConfig(path string) (*Config, error) {
	v := viper.New()

	// If path is provided, use it; otherwise use default
	if path == "" {
		defaultPath, err := GetConfigPath()
		if err != nil {
			return nil, fmt.Errorf("failed to get default config path: %w", err)
		}
		path = defaultPath
	}

	// Set config file path
	configDir := filepath.Dir(path)
	configFile := filepath.Base(path)
	v.SetConfigName(configFile[:len(configFile)-len(filepath.Ext(configFile))]) // Remove extension
	v.SetConfigType("yaml")
	v.AddConfigPath(configDir)

	// Set defaults
	v.SetDefault("internal_domains", []string{})
	v.SetDefault("trusted_domains", []string{})
	v.SetDefault("default_scope", "active")
	v.SetDefault("dry_run", true)
	v.SetDefault("credentials_path", "")
	v.SetDefault("impersonate_user", "")
	v.SetDefault("logging.enabled", false)
	v.SetDefault("logging.file_path", "")

	// Try to read config file (it's okay if it doesn't exist)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Config file exists but couldn't be read
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file doesn't exist, use defaults
	}

	config := &Config{}
	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Migrate deprecated top-level Google fields to nested config
	config.MigrateGoogleConfig()

	// Apply defaults if not set
	if config.CredentialsPath == "" {
		homeDir, _ := os.UserHomeDir()
		config.CredentialsPath = filepath.Join(homeDir, ".dplense", "credentials.json")
	}
	if config.Logging.Enabled && config.Logging.FilePath == "" {
		homeDir, _ := os.UserHomeDir()
		config.Logging.FilePath = filepath.Join(homeDir, ".dplense", "audit.log")
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// SaveConfig saves the configuration to the specified path.
// The file is created with 0600 permissions to protect secrets.
func SaveConfig(config *Config, path string) error {
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Ensure config directory exists
	configDir := filepath.Dir(path)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigFile(path)

	// Provider-agnostic
	v.Set("default_provider", config.DefaultProvider)
	v.Set("internal_domains", config.InternalDomains)
	v.Set("trusted_domains", config.TrustedDomains)
	v.Set("default_scope", config.DefaultScope)
	v.Set("dry_run", config.DryRun)
	v.Set("logging", config.Logging)

	// Deprecated top-level Google fields (backward compat)
	v.Set("credentials_path", config.CredentialsPath)
	v.Set("impersonate_user", config.ImpersonateUser)

	// Provider-specific configs
	v.Set("google", config.Google)
	v.Set("microsoft", config.Microsoft)
	v.Set("slack", config.Slack)

	if err := v.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Restrict file permissions — config may contain secrets
	if err := os.Chmod(path, 0600); err != nil {
		return fmt.Errorf("failed to set config file permissions: %w", err)
	}

	return nil
}
