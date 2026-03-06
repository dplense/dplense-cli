package auth

import (
	"fmt"
	"strings"
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
// NewLabelsService via LabelsScopes(). Including it here would break
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
