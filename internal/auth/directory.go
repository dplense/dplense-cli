package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/oauth2/google"
	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/option"
)

// NewDirectoryService creates a new Google Admin Directory service client using a Service Account.
// It supports domain-wide delegation by impersonating a user.
func NewDirectoryService(ctx context.Context, credentialsFile, impersonateUser string) (*admin.Service, error) {
	if impersonateUser == "" {
		return nil, fmt.Errorf("impersonate user is required for Admin Directory API access")
	}

	b, err := os.ReadFile(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read credentials file %s: %w", credentialsFile, err)
	}

	// Verify it is a Service Account
	var credsMap map[string]interface{}
	if err := json.Unmarshal(b, &credsMap); err != nil {
		return nil, fmt.Errorf("failed to parse credentials file: %w", err)
	}

	typeVal, ok := credsMap["type"].(string)
	if !ok || typeVal != "service_account" {
		return nil, fmt.Errorf("invalid credentials file: expected 'service_account', got '%s'. This tool only supports Service Account authentication", typeVal)
	}

	scopes := DirectoryScopes()
	config, err := google.JWTConfigFromJSON(b, scopes...)
	if err != nil {
		return nil, fmt.Errorf("unable to parse service account key for impersonation: %w", err)
	}
	config.Subject = impersonateUser

	service, err := admin.NewService(ctx, option.WithTokenSource(config.TokenSource(ctx)))
	if err != nil {
		return nil, fmt.Errorf("failed to create Directory service: %w", err)
	}

	return service, nil
}
