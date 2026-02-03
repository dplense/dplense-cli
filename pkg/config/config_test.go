package config

import (
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.DefaultScope != "active" {
		t.Errorf("Expected default scope 'active', got '%s'", config.DefaultScope)
	}
	if !config.DryRun {
		t.Error("Expected DryRun to be true by default")
	}
	if config.CredentialsPath == "" {
		t.Error("Expected CredentialsPath to be set")
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				DefaultScope:    "active",
				CredentialsPath: "/path/to/credentials.json",
			},
			wantErr: false,
		},
		{
			name: "empty scope",
			config: &Config{
				DefaultScope:    "",
				CredentialsPath: "/path/to/credentials.json",
			},
			wantErr: true,
		},
		{
			name: "invalid scope",
			config: &Config{
				DefaultScope:    "invalid",
				CredentialsPath: "/path/to/credentials.json",
			},
			wantErr: true,
		},
		{
			name: "valid user scope",
			config: &Config{
				DefaultScope:    "user:test@example.com",
				CredentialsPath: "/path/to/credentials.json",
			},
			wantErr: false,
		},
		{
			name: "empty credentials path",
			config: &Config{
				DefaultScope:    "active",
				CredentialsPath: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetConfigPath(t *testing.T) {
	path, err := GetConfigPath()
	if err != nil {
		t.Fatalf("GetConfigPath() error = %v", err)
	}
	if path == "" {
		t.Error("Expected non-empty config path")
	}
	if filepath.Ext(path) != ".yaml" {
		t.Errorf("Expected .yaml extension, got %s", filepath.Ext(path))
	}
}
