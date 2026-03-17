package google

import (
	"context"
	"fmt"
	"strings"

	"github.com/dplense/dplense-cli/internal/filter"
	"github.com/dplense/dplense-cli/internal/logger"
	"github.com/dplense/dplense-cli/pkg/config"
)

// fileRevoker handles revocation of external permissions from files
type fileRevoker struct {
	driveClient DriveClient
	config      *config.Config
	logger      logger.Logger
	dryRun      bool
}

// newFileRevoker creates a new file revoker
func newFileRevoker(driveClient DriveClient, cfg *config.Config, log logger.Logger, dryRun bool) *fileRevoker {
	return &fileRevoker{
		driveClient: driveClient,
		config:      cfg,
		logger:      log,
		dryRun:      dryRun,
	}
}

// revokeExternalPermissions revokes all external permissions from a file.
// Returns the number of permissions revoked (or that would be revoked in dry-run).
func (fr *fileRevoker) revokeExternalPermissions(ctx context.Context, fileID string) (int, error) {
	// Get all permissions for the file
	permissions, err := fr.driveClient.ListPermissions(ctx, fileID)
	if err != nil {
		return 0, fmt.Errorf("failed to list permissions for file %s: %w", fileID, err)
	}

	// Filter to external permissions only
	externalPerms := make([]Permission, 0)
	for _, perm := range permissions {
		// Skip owner permissions
		if perm.Role == "owner" {
			continue
		}

		// Check if permission is external
		isPublic := perm.Type == "anyone"
		isExternal := !filter.IsInternalDomain(perm.EmailAddress, fr.config.InternalDomains) &&
			!filter.IsInAllowlist(perm.EmailAddress, perm.Domain, fr.config.TrustedDomains)

		if isPublic || isExternal {
			externalPerms = append(externalPerms, perm)
		}
	}

	if len(externalPerms) == 0 {
		fr.logger.Info("No external permissions found on file: fileID=%s", fileID)
		return 0, nil
	}

	// Revoke each external permission
	revokedCount := 0
	for _, perm := range externalPerms {
		if fr.dryRun {
			fr.logger.Info("[DRY-RUN] Would revoke permission: fileID=%s, permissionID=%s, type=%s, email=%s, domain=%s, role=%s",
				fileID, perm.ID, perm.Type, perm.EmailAddress, perm.Domain, perm.Role)
			revokedCount++
		} else {
			if err := fr.driveClient.DeletePermission(ctx, fileID, perm.ID); err != nil {
				return revokedCount, fmt.Errorf("failed to revoke permission %s: %w", perm.ID, err)
			}
			fr.logger.Info("Revoked permission: fileID=%s, permissionID=%s, type=%s, email=%s, domain=%s, role=%s",
				fileID, perm.ID, perm.Type, perm.EmailAddress, perm.Domain, perm.Role)
			revokedCount++
		}
	}

	if fr.dryRun {
		fr.logger.Info("[DRY-RUN] Would revoke %d external permissions from file: %s", revokedCount, fileID)
	} else {
		fr.logger.Info("Revoked %d external permissions from file: %s", revokedCount, fileID)
	}

	return revokedCount, nil
}

// revokeUserFromFile revokes a specific user's access from a file
func (fr *fileRevoker) revokeUserFromFile(ctx context.Context, fileID string, userEmail string) error {
	// Get all permissions for the file
	permissions, err := fr.driveClient.ListPermissions(ctx, fileID)
	if err != nil {
		return fmt.Errorf("failed to list permissions for file %s: %w", fileID, err)
	}

	// Find the matching permission
	var userPerm *Permission
	if strings.EqualFold(userEmail, "anyone") {
		// Special case: revoke public "Anyone with link" permission
		for _, perm := range permissions {
			if perm.Type == "anyone" {
				userPerm = &perm
				break
			}
		}
	} else {
		for _, perm := range permissions {
			if strings.EqualFold(perm.EmailAddress, userEmail) {
				userPerm = &perm
				break
			}
		}
	}

	if userPerm == nil {
		fr.logger.Info("User not found in file permissions: fileID=%s, user=%s", fileID, userEmail)
		return fmt.Errorf("user %s not found in file permissions", userEmail)
	}

	// Don't revoke owner permissions
	if userPerm.Role == "owner" {
		fr.logger.Warn("Cannot revoke owner permission: fileID=%s, user=%s", fileID, userEmail)
		return fmt.Errorf("cannot revoke owner permission for %s", userEmail)
	}

	// Revoke the permission
	if fr.dryRun {
		fr.logger.Info("[DRY-RUN] Would revoke permission: fileID=%s, permissionID=%s, user=%s, role=%s",
			fileID, userPerm.ID, userEmail, userPerm.Role)
	} else {
		if err := fr.driveClient.DeletePermission(ctx, fileID, userPerm.ID); err != nil {
			return fmt.Errorf("failed to revoke permission for %s: %w", userEmail, err)
		}
		fr.logger.Info("Revoked permission: fileID=%s, permissionID=%s, user=%s, role=%s",
			fileID, userPerm.ID, userEmail, userPerm.Role)
	}

	return nil
}
