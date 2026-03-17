package microsoft

import (
	"context"
	"fmt"
	"strings"

	"github.com/dplense/dplense-cli/internal/logger"
	"github.com/dplense/dplense-cli/pkg/models"
)

type revoker struct {
	gc  *graphClient
	log logger.Logger
}

func newRevoker(gc *graphClient, log logger.Logger) *revoker {
	return &revoker{gc: gc, log: log}
}

// revokePermission removes a specific permission from a resource.
// resourceID format: "driveID/itemID"
// permissionID is the Graph permission ID, or a user email to revoke.
func (r *revoker) revokePermission(ctx context.Context, resourceID, permissionID string, dryRun bool) (string, error) {
	// Parse resourceID as "driveID/itemID"
	parts := strings.SplitN(resourceID, "/", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("resourceID must be in format 'driveID/itemID', got: %s", resourceID)
	}
	driveID, itemID := parts[0], parts[1]

	if permissionID == "" {
		// Revoke all external permissions
		return r.revokeAllExternal(ctx, driveID, itemID, dryRun)
	}

	// Check if permissionID is an email (user revoke) or actual perm ID
	if strings.Contains(permissionID, "@") {
		return r.revokeUserFromItem(ctx, driveID, itemID, permissionID, dryRun)
	}

	// Direct permission ID revoke
	if dryRun {
		r.log.Info("[DRY-RUN] Would revoke permission %s from %s/%s", permissionID, driveID, itemID)
		return fmt.Sprintf("[DRY-RUN] Would revoke permission %s", permissionID), nil
	}

	if err := r.gc.deletePermission(ctx, driveID, itemID, permissionID); err != nil {
		return "", err
	}
	return fmt.Sprintf("Revoked permission %s from item %s", permissionID, itemID), nil
}

func (r *revoker) revokeAllExternal(ctx context.Context, driveID, itemID string, dryRun bool) (string, error) {
	perms, err := r.gc.listItemPermissions(ctx, driveID, itemID)
	if err != nil {
		return "", err
	}

	count := 0
	for _, gp := range perms {
		if isInherited(gp) {
			continue
		}

		mp := mapPermission(gp, nil) // No internal domains check for revoke
		if mp.Type == "anyone" || !mp.IsInternal {
			if dryRun {
				r.log.Info("[DRY-RUN] Would revoke permission %s (type=%s, email=%s)", deref(gp.GetId()), mp.Type, mp.Email)
				count++
			} else {
				if err := r.gc.deletePermission(ctx, driveID, itemID, deref(gp.GetId())); err != nil {
					r.log.Warn("Failed to revoke permission %s: %v", deref(gp.GetId()), err)
					continue
				}
				count++
			}
		}
	}

	if dryRun {
		return fmt.Sprintf("[DRY-RUN] Would revoke %d external permissions", count), nil
	}
	return fmt.Sprintf("Revoked %d external permissions", count), nil
}

func (r *revoker) revokeUserFromItem(ctx context.Context, driveID, itemID, email string, dryRun bool) (string, error) {
	perms, err := r.gc.listItemPermissions(ctx, driveID, itemID)
	if err != nil {
		return "", err
	}

	for _, gp := range perms {
		mp := mapPermission(gp, nil)
		if strings.EqualFold(mp.Email, email) {
			if dryRun {
				return fmt.Sprintf("[DRY-RUN] Would revoke %s's access (role=%s)", email, mp.Role), nil
			}
			if err := r.gc.deletePermission(ctx, driveID, itemID, deref(gp.GetId())); err != nil {
				return "", err
			}
			return fmt.Sprintf("Revoked %s's access (role=%s)", email, mp.Role), nil
		}
	}

	return fmt.Sprintf("User %s not found in permissions", email), nil
}

// revokeUserAccess removes a user's access from all files in scan results.
func (r *revoker) revokeUserAccess(ctx context.Context, email string, scanResult *models.ScanResult, dryRun bool) (int, int, error) {
	revoked := 0
	filesAffected := 0

	for _, issue := range scanResult.Issues {
		hasUser := false
		for _, perm := range issue.Permissions {
			if strings.EqualFold(perm.Email, email) {
				hasUser = true
				break
			}
		}
		if !hasUser {
			continue
		}

		// Need driveID to revoke — stored in issue.DriveID
		if issue.DriveID == "" {
			r.log.Warn("Missing DriveID for file %s, skipping", issue.FileName)
			continue
		}

		perms, err := r.gc.listItemPermissions(ctx, issue.DriveID, issue.FileID)
		if err != nil {
			r.log.Warn("Failed to list permissions for %s: %v", issue.FileName, err)
			continue
		}

		for _, gp := range perms {
			mp := mapPermission(gp, nil)
			if strings.EqualFold(mp.Email, email) {
				if dryRun {
					r.log.Info("[DRY-RUN] Would revoke %s from %s", email, issue.FileName)
					revoked++
				} else {
					if err := r.gc.deletePermission(ctx, issue.DriveID, issue.FileID, deref(gp.GetId())); err != nil {
						r.log.Warn("Failed to revoke: %v", err)
						continue
					}
					revoked++
				}
			}
		}

		if revoked > 0 {
			filesAffected++
		}
	}

	return revoked, filesAffected, nil
}
