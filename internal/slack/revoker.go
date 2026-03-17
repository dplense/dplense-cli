package slack

import (
	"context"
	"fmt"
	"strings"

	"github.com/dplense/dplense-cli/internal/logger"
	"github.com/dplense/dplense-cli/pkg/models"
)

type revoker struct {
	sc  *slackClient
	log logger.Logger
}

func newRevoker(sc *slackClient, log logger.Logger) *revoker {
	return &revoker{sc: sc, log: log}
}

// revokePermission handles revocation for a Slack resource.
// resourceID is the file ID or "guest-<userID>-<channelID>".
// permissionID encodes the action: "public-<fileID>", "guest-<userID>".
func (r *revoker) revokePermission(ctx context.Context, resourceID, permissionID string, dryRun bool) (string, error) {
	switch {
	case strings.HasPrefix(permissionID, "public-"):
		return r.revokePublicFile(ctx, resourceID, dryRun)
	case strings.HasPrefix(permissionID, "guest-"):
		return r.revokeGuestAccess(ctx, resourceID, permissionID, dryRun)
	default:
		return "", fmt.Errorf("unsupported Slack permission type: %s", permissionID)
	}
}

func (r *revoker) revokePublicFile(ctx context.Context, fileID string, dryRun bool) (string, error) {
	if dryRun {
		r.log.Info("[DRY-RUN] Would revoke public URL for file %s", fileID)
		return fmt.Sprintf("[DRY-RUN] Would revoke public URL for file %s", fileID), nil
	}

	if err := r.sc.revokeFilePublicURL(ctx, fileID); err != nil {
		return "", fmt.Errorf("revoke public URL for %s: %w", fileID, err)
	}
	return fmt.Sprintf("Revoked public URL for file %s", fileID), nil
}

func (r *revoker) revokeGuestAccess(ctx context.Context, resourceID, permissionID string, dryRun bool) (string, error) {
	// resourceID = "guest-<userID>-<channelID>"
	parts := strings.SplitN(resourceID, "-", 3)
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid guest resource ID: %s (expected guest-<userID>-<channelID>)", resourceID)
	}
	userID := parts[1]
	channelID := parts[2]

	if dryRun {
		r.log.Info("[DRY-RUN] Would remove guest %s from channel %s", userID, channelID)
		return fmt.Sprintf("[DRY-RUN] Would remove guest %s from channel %s", userID, channelID), nil
	}

	if err := r.sc.kickUserFromConversation(ctx, channelID, userID); err != nil {
		return "", fmt.Errorf("kick guest %s from %s: %w", userID, channelID, err)
	}
	return fmt.Sprintf("Removed guest %s from channel %s", userID, channelID), nil
}

// revokeUserAccess removes a user's access from all resources in scan results.
func (r *revoker) revokeUserAccess(ctx context.Context, email string, scanResult *models.ScanResult, dryRun bool) (int, int, error) {
	revoked := 0
	resourcesAffected := 0

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

		for _, perm := range issue.Permissions {
			if !strings.EqualFold(perm.Email, email) {
				continue
			}

			msg, err := r.revokePermission(ctx, issue.FileID, perm.ID, dryRun)
			if err != nil {
				r.log.Warn("Failed to revoke %s from %s: %v", email, issue.FileName, err)
				continue
			}
			r.log.Info("%s", msg)
			revoked++
		}
		resourcesAffected++
	}

	return revoked, resourcesAffected, nil
}
