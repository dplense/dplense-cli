package google

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"golang.org/x/oauth2/google"
	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/drivelabels/v2"
	"google.golang.org/api/option"
)

// RequiredScopes contains all OAuth scopes required by the application
var RequiredScopes = []string{
	"https://www.googleapis.com/auth/drive.readonly",                // Read-only access to Drive
	"https://www.googleapis.com/auth/drive.metadata.readonly",       // Read-only access to Drive metadata
	"https://www.googleapis.com/auth/drive.labels.readonly",         // Read-only access to Drive labels
	"https://www.googleapis.com/auth/admin.directory.user.readonly", // Read-only access to users
}

// DriveScopes returns the scopes needed for Drive API operations.
// Note: drive.labels.readonly is NOT included here — it is only used by
// newLabelsService via LabelsScopes(). Including it here would break
// authentication if the scope is not authorized in domain-wide delegation.
func DriveScopes() []string {
	return []string{
		"https://www.googleapis.com/auth/drive.readonly",
		"https://www.googleapis.com/auth/drive.metadata.readonly",
	}
}

// LabelsScopes returns the scopes needed for Drive Labels API operations
func LabelsScopes() []string {
	return []string{
		"https://www.googleapis.com/auth/drive.labels.readonly",
	}
}

// DirectoryScopes returns the scopes needed for Admin Directory API operations
func DirectoryScopes() []string {
	return []string{
		"https://www.googleapis.com/auth/admin.directory.user.readonly",
	}
}

// GetScopes returns all required OAuth scopes
func GetScopes() []string {
	return RequiredScopes
}

// FormatScopesForDisplay formats scopes for display to the user
func FormatScopesForDisplay(scopes []string) string {
	var builder strings.Builder
	builder.WriteString("\nRequired OAuth Scopes for domain-wide delegation:\n")
	for _, scope := range scopes {
		builder.WriteString(fmt.Sprintf("  - %s\n", scope))
	}
	return builder.String()
}

// GetDomainWideDelegationInstructions returns instructions for setting up domain-wide delegation
func GetDomainWideDelegationInstructions(clientID string) string {
	scopesDisplay := FormatScopesForDisplay(RequiredScopes)
	return fmt.Sprintf(`%s
Make sure these scopes are authorized in Google Admin Console:
  1. Go to: https://admin.google.com
  2. Security > API Controls > Domain-wide Delegation
  3. Add/Edit your service account (Client ID: %s)
  4. Authorize the scopes listed above (comma-separated):
     %s
  5. Wait a few minutes for changes to propagate`, scopesDisplay, clientID, strings.Join(RequiredScopes, ", "))
}

// newDriveService creates a new Google Drive service using Service Account authentication
func newDriveService(ctx context.Context, credentialsPath, impersonateUser string) (*drive.Service, error) {
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

// newLabelsService creates a new Drive Labels API service using Service Account authentication
func newLabelsService(ctx context.Context, credentialsPath, impersonateUser string) (*drivelabels.Service, error) {
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

// newDirectoryService creates a new Google Admin Directory service client using a Service Account.
// It supports domain-wide delegation by impersonating a user.
func newDirectoryService(ctx context.Context, credentialsFile, impersonateUser string) (*admin.Service, error) {
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
