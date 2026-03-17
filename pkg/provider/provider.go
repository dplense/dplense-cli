package provider

import (
	"context"

	"github.com/dplense/dplense-cli/pkg/models"
)

// Name identifies a cloud storage provider.
type Name string

const (
	Google    Name = "google"
	Microsoft Name = "microsoft"
	Slack     Name = "slack"
)

// ScanOptions holds provider-agnostic scan parameters.
type ScanOptions struct {
	Scope            string
	FilterSharedWith string
	FilterPublicOnly bool
	FilterRiskLevels []string

	// Progress callbacks (optional)
	ProgressFunc       func(filesScanned, issuesFound int)
	TargetProgressFunc func(targetIdx, targetTotal int, targetName string)
	StatusFunc         func(msg string) // plain status message (no log prefix)
}

// Provider is the core abstraction for a cloud storage audit source.
// Each provider handles its own authentication, API calls, scoping,
// and data mapping. All providers produce models.ScanResult output.
type Provider interface {
	// Name returns the provider identifier.
	Name() Name

	// ValidateConfig checks that the provider's configuration is
	// present and valid. Called before Scan or Revoke.
	ValidateConfig() error

	// Scopes returns the list of valid scope strings for this provider.
	Scopes() []string

	// Scan performs a full security audit and returns issues.
	Scan(ctx context.Context, opts ScanOptions) (*models.ScanResult, error)

	// RevokePermission removes a specific permission from a resource.
	// dryRun=true means preview only. Returns a human-readable summary.
	RevokePermission(ctx context.Context, resourceID, permissionID string, dryRun bool) (string, error)

	// RevokeUserAccess removes all of a user's permissions from
	// resources found in a previous scan result.
	// Returns (permissionsRevoked, resourcesAffected, error).
	RevokeUserAccess(ctx context.Context, email string, scanResult *models.ScanResult, dryRun bool) (int, int, error)
}
