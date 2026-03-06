package revoke

import (
	"context"
	"strings"

	"gdrive-audit/internal/logger"
	"gdrive-audit/pkg/config"
	"gdrive-audit/pkg/gdrive"
	"gdrive-audit/pkg/models"
)

// UserRevoker handles revocation of a specific user from all files
type UserRevoker struct {
	driveClient gdrive.DriveClient
	config       *config.Config
	logger       logger.Logger
	dryRun       bool
}

// NewUserRevoker creates a new user revoker
func NewUserRevoker(driveClient gdrive.DriveClient, cfg *config.Config, log logger.Logger, dryRun bool) *UserRevoker {
	return &UserRevoker{
		driveClient: driveClient,
		config:      cfg,
		logger:      log,
		dryRun:      dryRun,
	}
}

// RevokeUserFromFiles revokes a specific user's access from all files in the scan result.
// Returns the number of permissions revoked (or that would be revoked in dry-run) and the number of files affected.
func (ur *UserRevoker) RevokeUserFromFiles(ctx context.Context, email string, scanResult *models.ScanResult) (revokedCount int, filesAffected int, err error) {
	// Find all files shared with this email
	filesToRevoke := make([]models.FileIssue, 0)
	for _, issue := range scanResult.Issues {
		for _, perm := range issue.Permissions {
			if perm.Email == email || (perm.Domain != "" && strings.HasSuffix(email, "@"+perm.Domain)) {
				// Check if this file is already in our list
				found := false
				for _, existing := range filesToRevoke {
					if existing.FileID == issue.FileID {
						found = true
						break
					}
				}
				if !found {
					filesToRevoke = append(filesToRevoke, issue)
				}
			}
		}
	}

	if len(filesToRevoke) == 0 {
		ur.logger.Info("No files found shared with user: %s", email)
		return 0, 0, nil
	}

	ur.logger.Info("Found %d files to revoke access from: %s", len(filesToRevoke), email)

	// Revoke user from each file
	revoked := 0
	failedCount := 0

	for _, issue := range filesToRevoke {
		// Get all permissions for the file
		permissions, listErr := ur.driveClient.ListPermissions(ctx, issue.FileID)
		if listErr != nil {
			ur.logger.Error("Failed to list permissions for file: fileID=%s, error=%v", issue.FileID, listErr)
			failedCount++
			continue
		}

		// Find and revoke permissions for this user
		for _, perm := range permissions {
			if perm.EmailAddress == email || (perm.Domain != "" && strings.HasSuffix(email, "@"+perm.Domain)) {
				if ur.dryRun {
					ur.logger.Info("[DRY-RUN] Would revoke permission: fileID=%s, fileName=%s, permissionID=%s, email=%s, role=%s",
						issue.FileID, issue.FileName, perm.ID, email, perm.Role)
					revoked++
				} else {
					if delErr := ur.driveClient.DeletePermission(ctx, issue.FileID, perm.ID); delErr != nil {
						ur.logger.Error("Failed to revoke permission: fileID=%s, permissionID=%s, error=%v", issue.FileID, perm.ID, delErr)
						failedCount++
						continue
					}
					ur.logger.Info("Revoked permission: fileID=%s, fileName=%s, permissionID=%s, email=%s, role=%s",
						issue.FileID, issue.FileName, perm.ID, email, perm.Role)
					revoked++
				}
			}
		}
	}

	if ur.dryRun {
		ur.logger.Info("[DRY-RUN] Would revoke %d permissions from %d files: %s", revoked, len(filesToRevoke), email)
	} else {
		ur.logger.Info("Revoked %d permissions from %d files: %s", revoked, len(filesToRevoke), email)
		if failedCount > 0 {
			ur.logger.Warn("%d permissions failed to revoke", failedCount)
		}
	}

	return revoked, len(filesToRevoke), nil
}
