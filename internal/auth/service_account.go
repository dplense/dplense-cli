package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/drivelabels/v2"
	"google.golang.org/api/option"
)

// NewDriveService creates a new Google Drive service using Service Account authentication
func NewDriveService(ctx context.Context, credentialsPath, impersonateUser string) (*drive.Service, error) {
	b, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read credentials file %s: %w", credentialsPath, err)
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

	scopes := DriveScopes()

	if impersonateUser != "" {

		config, err := google.JWTConfigFromJSON(b, scopes...)
		if err != nil {
			return nil, fmt.Errorf("unable to parse service account key for impersonation: %w", err)
		}
		config.Subject = impersonateUser

		service, err := drive.NewService(ctx, option.WithTokenSource(config.TokenSource(ctx)))
		if err != nil {
			return nil, fmt.Errorf("failed to create Drive service: %w", err)
		}
		return service, nil
	}

	return drive.NewService(ctx, option.WithCredentialsJSON(b), option.WithScopes(scopes...))
}

// NewLabelsService creates a new Drive Labels API service using Service Account authentication
func NewLabelsService(ctx context.Context, credentialsPath, impersonateUser string) (*drivelabels.Service, error) {
	b, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read credentials file %s: %w", credentialsPath, err)
	}

	scopes := LabelsScopes()

	if impersonateUser != "" {
		config, err := google.JWTConfigFromJSON(b, scopes...)
		if err != nil {
			return nil, fmt.Errorf("unable to parse service account key for labels: %w", err)
		}
		config.Subject = impersonateUser

		service, err := drivelabels.NewService(ctx, option.WithTokenSource(config.TokenSource(ctx)))
		if err != nil {
			return nil, fmt.Errorf("failed to create Labels service: %w", err)
		}
		return service, nil
	}

	return drivelabels.NewService(ctx, option.WithCredentialsJSON(b), option.WithScopes(scopes...))
}
