package audit

import (
	"context"
	"fmt"
	"strings"

	"gdrive-audit/internal/logger"
	"gdrive-audit/pkg/gdrive"
)

// Scope represents the scan scope type
type Scope string

const (
	// ScopeActive scans active users
	ScopeActive Scope = "active"
	// ScopeSuspended scans suspended users
	ScopeSuspended Scope = "suspended"
	// ScopeSharedDrives scans shared drives
	ScopeSharedDrives Scope = "shared-drives"
	// ScopeUserPrefix is the prefix for user-specific scopes
	ScopeUserPrefix Scope = "user:"
)

// Target represents a scan target (user or drive)
type Target struct {
	Type  string // "user" or "drive"
	ID    string
	Email string
	Name  string
	Owner string // Owner/creator of the drive or user
}

// ShouldIncludeDrive checks if a drive target is in the inclusion list.
// Returns true if includedDrives is empty (all included by default),
// target is not a drive type, or target matches by ID or Name.
func ShouldIncludeDrive(target Target, includedDrives []string) bool {
	if len(includedDrives) == 0 {
		return true
	}
	if target.Type != "drive" {
		return true
	}
	for _, included := range includedDrives {
		if target.ID == included || target.Name == included {
			return true
		}
	}
	return false
}

// ShouldExcludeDrive checks if a drive target is in the exclusion list.
// Returns false if excludedDrives is empty or target is not a drive type.
func ShouldExcludeDrive(target Target, excludedDrives []string) bool {
	if len(excludedDrives) == 0 {
		return false
	}
	if target.Type != "drive" {
		return false
	}
	for _, excluded := range excludedDrives {
		if target.ID == excluded || target.Name == excluded {
			return true
		}
	}
	return false
}

// ResolveScope resolves a scope string to a list of targets
func ResolveScope(scope string, dirClient gdrive.DirectoryClient, driveClient gdrive.DriveClient, log logger.Logger) ([]Target, error) {
	scopeLower := strings.ToLower(strings.TrimSpace(scope))

	switch {
	case scopeLower == string(ScopeActive):
		return resolveActiveUsers(context.Background(), dirClient, log)
	case scopeLower == string(ScopeSuspended):
		return resolveSuspendedUsers(context.Background(), dirClient, log)
	case scopeLower == string(ScopeSharedDrives):
		return resolveSharedDrives(context.Background(), driveClient, log)
	case strings.HasPrefix(scopeLower, string(ScopeUserPrefix)):
		email := strings.TrimPrefix(scopeLower, string(ScopeUserPrefix))
		return resolveUser(context.Background(), email, dirClient, log)
	default:
		return nil, fmt.Errorf("invalid scope: %s (must be 'active', 'suspended', 'shared-drives', or 'user:<email>')", scope)
	}
}

// resolveActiveUsers resolves active users
func resolveActiveUsers(ctx context.Context, dirClient gdrive.DirectoryClient, log logger.Logger) ([]Target, error) {
	if dirClient == nil {
		return nil, fmt.Errorf("directory client is required for 'active' scope but is not available.\n\n" +
			"To use 'active' scope:\n" +
			"1. Use --impersonate flag with a user email\n" +
			"2. Authorize Admin Directory API scope in Google Admin Console:\n" +
			"   - https://www.googleapis.com/auth/admin.directory.user.readonly\n\n" +
			"Alternatively, use a scope that doesn't require Directory API:\n" +
			"  - shared-drives: ./gdaudit scan --scope shared-drives")
	}
	log.Info("Resolving active users from directory...")
	users, err := dirClient.ListUsers(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("failed to list active users: %w", err)
	}

	targets := make([]Target, 0, len(users))
	for _, user := range users {
		targets = append(targets, Target{
			Type:  "user",
			ID:    user.ID,
			Email: user.Email,
			Name:  user.DisplayName,
			Owner: user.Email, // For users, the owner is themselves
		})
	}

	log.Info("Resolved %d active users", len(targets))
	return targets, nil
}

// resolveSuspendedUsers resolves suspended users
func resolveSuspendedUsers(ctx context.Context, dirClient gdrive.DirectoryClient, log logger.Logger) ([]Target, error) {
	if dirClient == nil {
		return nil, fmt.Errorf("directory client is required for 'suspended' scope but is not available. Please configure domain-wide delegation with --impersonate flag or use a different scope like 'shared-drives'")
	}
	log.Info("Resolving suspended users from directory...")
	users, err := dirClient.ListUsers(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("failed to list suspended users: %w", err)
	}

	targets := make([]Target, 0, len(users))
	for _, user := range users {
		targets = append(targets, Target{
			Type:  "user",
			ID:    user.ID,
			Email: user.Email,
			Name:  user.DisplayName,
			Owner: user.Email, // For users, the owner is themselves
		})
	}

	log.Info("Resolved %d suspended users", len(targets))
	return targets, nil
}

// resolveSharedDrives resolves shared drives
func resolveSharedDrives(ctx context.Context, driveClient gdrive.DriveClient, log logger.Logger) ([]Target, error) {
	if driveClient == nil {
		return nil, fmt.Errorf("drive client is required for 'shared-drives' scope but is not available")
	}
	log.Info("Identifying all shared drives...")

	// Try with admin access first, fall back to regular access if that fails
	drives, err := driveClient.ListDrives(ctx, true)
	if err != nil {
		// If admin access fails, try without admin access (may work if user has access)
		log.Warn("Failed to list shared drives with admin access, trying without: %v", err)
		drives, err = driveClient.ListDrives(ctx, false)
		if err != nil {
			return nil, fmt.Errorf("failed to list shared drives: %w", err)
		}
	}

	log.Info("Found %d shared drives (before filtering)", len(drives))

	targets := make([]Target, 0, len(drives))
	for _, drive := range drives {
		// For shared drives, use creator email if available, otherwise fall back to creation date
		owner := "Unknown"
		if drive.CreatedBy != nil && drive.CreatedBy.EmailAddress != "" {
			owner = drive.CreatedBy.EmailAddress
		} else if drive.CreatedTime != "" {
			// Fallback to creation date if creator not available
			if len(drive.CreatedTime) >= 10 {
				owner = drive.CreatedTime[:10]
			}
		}

		targets = append(targets, Target{
			Type:  "drive",
			ID:    drive.ID,
			Name:  drive.Name,
			Owner: owner,
		})
	}

	return targets, nil
}

// resolveUser resolves a single user by email.
// Tries Directory API first for richer metadata; falls back to email-only target
// when Directory API is unavailable or the user cannot be found (e.g. external users).
func resolveUser(ctx context.Context, email string, dirClient gdrive.DirectoryClient, log logger.Logger) ([]Target, error) {
	// Try Directory API for richer metadata (display name, ID)
	if dirClient != nil {
		log.Info("Resolving user via Directory API: %s", email)
		user, err := dirClient.GetUser(ctx, email)
		if err != nil {
			log.Info("Directory lookup not available for %s, using email directly", email)
		} else {
			log.Info("Resolved user: %s (%s)", user.Email, user.DisplayName)
			return []Target{
				{
					Type:  "user",
					ID:    user.ID,
					Email: user.Email,
					Name:  user.DisplayName,
					Owner: user.Email,
				},
			}, nil
		}
	}

	// Fallback: create target directly from the email
	log.Info("Using email directly: %s", email)
	return []Target{
		{
			Type:  "user",
			Email: email,
			Name:  email,
			Owner: email,
		},
	}, nil
}
