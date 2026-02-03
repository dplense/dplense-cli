package revoke

import (
	"context"
	"fmt"

	"gdrive-audit/internal/filter"
	"gdrive-audit/internal/logger"
	"gdrive-audit/pkg/config"
	"gdrive-audit/pkg/gdrive"
)

// FileRevoker handles revocation of external permissions from files
type FileRevoker struct {
	driveClient gdrive.DriveClient
	config      *config.Config
	logger      logger.Logger
	dryRun      bool
}

// NewFileRevoker creates a new file revoker
func NewFileRevoker(driveClient gdrive.DriveClient, cfg *config.Config, log logger.Logger, dryRun bool) *FileRevoker {
	return &FileRevoker{
		driveClient: driveClient,
		config:      cfg,
		logger:      log,
		dryRun:      dryRun,
	}
}

// RevokeExternalPermissions revokes all external permissions from a file
func (fr *FileRevoker) RevokeExternalPermissions(ctx context.Context, fileID string) error {
	// Get all permissions for the file
	permissions, err := fr.driveClient.ListPermissions(ctx, fileID)
	if err != nil {
		return fmt.Errorf("failed to list permissions for file %s: %w", fileID, err)
	}

	// Filter to external permissions only
	externalPerms := make([]gdrive.Permission, 0)
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
		return nil
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
				fr.logger.Error("Failed to revoke permission: fileID=%s, permissionID=%s, error=%v", fileID, perm.ID, err)
				return fmt.Errorf("failed to revoke permission %s: %w", perm.ID, err)
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

	return nil
}

// RevokeUserFromFile revokes a specific user's access from a file
func (fr *FileRevoker) RevokeUserFromFile(ctx context.Context, fileID string, userEmail string) error {
	// Get all permissions for the file
	permissions, err := fr.driveClient.ListPermissions(ctx, fileID)
	if err != nil {
		return fmt.Errorf("failed to list permissions for file %s: %w", fileID, err)
	}

	// Find the user's permission
	var userPerm *gdrive.Permission
	for _, perm := range permissions {
		if perm.EmailAddress == userEmail {
			userPerm = &perm
			break
		}
	}

	if userPerm == nil {
		fr.logger.Info("User not found in file permissions: fileID=%s, user=%s", fileID, userEmail)
		fmt.Printf("\n⚠️  User %s not found in file permissions.\n", userEmail)
		return nil
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
		fmt.Printf("\n[DRY-RUN] Would revoke:\n")
		fmt.Printf("  File ID: %s\n", fileID)
		fmt.Printf("  User: %s\n", userEmail)
		fmt.Printf("  Role: %s\n", userPerm.Role)
	} else {
		if err := fr.driveClient.DeletePermission(ctx, fileID, userPerm.ID); err != nil {
			fr.logger.Error("Failed to revoke permission: fileID=%s, permissionID=%s, user=%s, error=%v",
				fileID, userPerm.ID, userEmail, err)
			return fmt.Errorf("failed to revoke permission: %w", err)
		}
		fr.logger.Info("Revoked permission: fileID=%s, permissionID=%s, user=%s, role=%s",
			fileID, userPerm.ID, userEmail, userPerm.Role)
		fmt.Printf("\n✅ Revoked:\n")
		fmt.Printf("  File ID: %s\n", fileID)
		fmt.Printf("  User: %s\n", userEmail)
		fmt.Printf("  Role: %s\n", userPerm.Role)
	}

	return nil
}
