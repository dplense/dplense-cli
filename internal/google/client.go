package google

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dplense/dplense-cli/internal/logger"
	"github.com/dplense/dplense-cli/pkg/config"

	driveapi "google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
)

// driveClient implements the DriveClient interface
type driveClient struct {
	service       *driveapi.Service
	logger        logger.Logger
	rateLimiter   *RateLimiter
	includeLabels string                                   // comma-separated label IDs for Files.List
	labelResolver func([]*driveapi.Label) []string // converts raw API labels to human-readable strings
}

// NewDriveClient creates a new Drive client with built-in rate limiting.
// Handles authentication internally using credentials from config.
func NewDriveClient(ctx context.Context, cfg *config.Config, log logger.Logger) (DriveClient, error) {
	service, err := newDriveService(ctx, cfg.CredentialsPath, cfg.ImpersonateUser)
	if err != nil {
		if strings.Contains(err.Error(), "unauthorized_client") || strings.Contains(err.Error(), "401") {
			return nil, fmt.Errorf("failed to initialize Drive service: %w\n\n"+
				"TIP: This usually means domain-wide delegation is not configured.\n"+
				"1. Make sure you're using --impersonate flag with a user email\n"+
				"2. Configure domain-wide delegation in Google Admin Console:\n"+
				"   - Go to: https://admin.google.com\n"+
				"   - Security > API Controls > Domain-wide Delegation\n"+
				"   - Add/Edit your service account\n"+
				"   - Authorize these scopes:\n"+
				"     * https://www.googleapis.com/auth/drive.readonly\n"+
				"     * https://www.googleapis.com/auth/drive.metadata.readonly\n"+
				"3. Wait a few minutes for changes to propagate", err)
		}
		return nil, fmt.Errorf("failed to initialize Drive service: %w", err)
	}
	return &driveClient{
		service:     service,
		logger:      log,
		rateLimiter: NewRateLimiter(5, 1*time.Second, 30*time.Second),
	}, nil
}

// SetIncludeLabels sets label IDs to request in Files.List calls.
func (c *driveClient) SetIncludeLabels(labelIDs string) {
	c.includeLabels = labelIDs
}

// SetLabelResolver sets a callback that converts raw API labels to human-readable strings.
func (c *driveClient) SetLabelResolver(fn func([]*driveapi.Label) []string) {
	c.labelResolver = fn
}

// configureLabelResolver sets the label resolver on a DriveClient.
// This is a package-level helper so callers don't need to import the Drive API types.
func configureLabelResolver(dc DriveClient, resolver func([]*driveapi.Label) []string) {
	if c, ok := dc.(*driveClient); ok {
		c.labelResolver = resolver
	}
}

// waitForQuota waits if the API rate limit quota is exhausted
func (c *driveClient) waitForQuota(ctx context.Context) error {
	return c.rateLimiter.WaitIfNeeded(ctx)
}

// ListFiles lists files matching the query with pagination
func (c *driveClient) ListFiles(ctx context.Context, query string, pageToken string, driveID string) (*FileList, error) {
	if err := c.waitForQuota(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}
	c.logger.Debug("Listing files with query: %s, pageToken: %s, driveID: %s", query, pageToken, driveID)

	// Note: permissions are NOT included here because Google API caps page size
	// to 100 when permissions are requested, and shared drive files return empty
	// permissions anyway. Permissions are fetched per-file via ListPermissions.
	fileFields := "id, name, webViewLink, driveId, parents, owners(displayName, emailAddress, me)"
	if c.includeLabels != "" {
		fileFields += ", labelInfo"
	}

	call := c.service.Files.List().
		Q(query).
		PageSize(1000).
		Fields(googleapi.Field("nextPageToken, files(" + fileFields + ")")).
		SupportsAllDrives(true).
		IncludeItemsFromAllDrives(true)

	if c.includeLabels != "" {
		call = call.IncludeLabels(c.includeLabels)
	}

	// If driveID is specified, limit search to that specific drive
	if driveID != "" {
		call = call.DriveId(driveID).Corpora("drive")
		c.logger.Debug("Limiting search to drive: %s", driveID)
	} else {
		call = call.Corpora("allDrives")
	}

	if pageToken != "" {
		call = call.PageToken(pageToken)
	}

	r, err := call.Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	c.logger.Info("Files.List API: returned %d files, hasNextPage=%v, driveID=%s", len(r.Files), r.NextPageToken != "", driveID)

	files := make([]File, 0, len(r.Files))
	for _, f := range r.Files {
		owners := make([]Owner, 0, len(f.Owners))
		for _, o := range f.Owners {
			owners = append(owners, Owner{
				DisplayName:  o.DisplayName,
				EmailAddress: o.EmailAddress,
				Me:           o.Me,
			})
		}

		gfile := File{
			ID:          f.Id,
			Name:        f.Name,
			WebViewLink: f.WebViewLink,
			DriveID:     f.DriveId,
			Parents:     f.Parents,
			Owners:      owners,
		}

		// Resolve labels if resolver is configured and file has label info
		if c.labelResolver != nil && f.LabelInfo != nil && len(f.LabelInfo.Labels) > 0 {
			gfile.Labels = c.labelResolver(f.LabelInfo.Labels)
		}

		files = append(files, gfile)
	}

	return &FileList{
		Files:         files,
		NextPageToken: r.NextPageToken,
	}, nil
}

// GetFile retrieves a file by ID
func (c *driveClient) GetFile(ctx context.Context, fileID string) (*File, error) {
	if err := c.waitForQuota(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}
	c.logger.Debug("Getting file: %s", fileID)

	f, err := c.service.Files.Get(fileID).
		Fields("id, name, webViewLink, driveId, parents, owners(displayName, emailAddress, me)").
		SupportsAllDrives(true).
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get file %s: %w", fileID, err)
	}

	c.logger.Debug("Files.Get API: fileID=%s, name=%s", fileID, f.Name)

	owners := make([]Owner, 0, len(f.Owners))
	for _, o := range f.Owners {
		owners = append(owners, Owner{
			DisplayName:  o.DisplayName,
			EmailAddress: o.EmailAddress,
			Me:           o.Me,
		})
	}

	return &File{
		ID:          f.Id,
		Name:        f.Name,
		WebViewLink: f.WebViewLink,
		DriveID:     f.DriveId,
		Parents:     f.Parents,
		Owners:      owners,
	}, nil
}

// ListPermissions lists all permissions for a file
func (c *driveClient) ListPermissions(ctx context.Context, fileID string) ([]Permission, error) {
	if err := c.waitForQuota(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}
	c.logger.Debug("Listing permissions for file: %s", fileID)

	r, err := c.service.Permissions.List(fileID).
		Fields("permissions(id, type, emailAddress, domain, role, permissionDetails)").
		SupportsAllDrives(true).
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list permissions for file %s: %w", fileID, err)
	}

	c.logger.Debug("Permissions.List API: fileID=%s, count=%d", fileID, len(r.Permissions))

	permissions := make([]Permission, 0, len(r.Permissions))
	for _, p := range r.Permissions {
		// Check if this permission is inherited
		isInherited := false
		inheritedFrom := ""
		if len(p.PermissionDetails) > 0 {
			for _, detail := range p.PermissionDetails {
				if detail.Inherited {
					isInherited = true
					inheritedFrom = detail.InheritedFrom
					break
				}
			}
		}

		permissions = append(permissions, Permission{
			ID:            p.Id,
			Type:          p.Type,
			EmailAddress:  p.EmailAddress,
			Domain:        p.Domain,
			Role:          p.Role,
			Inherited:     isInherited,
			InheritedFrom: inheritedFrom,
		})
	}

	return permissions, nil
}

// DeletePermission removes a permission from a file
func (c *driveClient) DeletePermission(ctx context.Context, fileID string, permissionID string) error {
	if err := c.waitForQuota(ctx); err != nil {
		return fmt.Errorf("rate limiter: %w", err)
	}
	c.logger.Debug("Deleting permission %s from file %s", permissionID, fileID)

	err := c.service.Permissions.Delete(fileID, permissionID).
		SupportsAllDrives(true).
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("failed to delete permission %s from file %s: %w", permissionID, fileID, err)
	}

	c.logger.Info("Permissions.Delete API: fileID=%s, permissionID=%s — revoked", fileID, permissionID)
	return nil
}

// ListDrives lists all Shared Drives
func (c *driveClient) ListDrives(ctx context.Context, adminAccess bool) ([]Drive, error) {
	if err := c.waitForQuota(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}
	c.logger.Debug("Listing drives, adminAccess: %v", adminAccess)

	call := c.service.Drives.List().
		PageSize(100). // Google API max for Drives.List is 100
		Fields("nextPageToken, drives(id, name, createdTime, restrictions)")

	if adminAccess {
		call = call.UseDomainAdminAccess(true)
	}

	drives := make([]Drive, 0)
	pageToken := ""

	for {
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		r, err := call.Context(ctx).Do()
		if err != nil {
			return nil, fmt.Errorf("failed to list drives: %w", err)
		}

		c.logger.Info("Drives.List API: returned %d drives, hasNextPage=%v", len(r.Drives), r.NextPageToken != "")

		for _, d := range r.Drives {
			drive := Drive{
				ID:          d.Id,
				Name:        d.Name,
				CreatedTime: d.CreatedTime,
			}

			// Note: The Google Drive API v3 does not expose 'createdBy' field in drives.list() or drives.get()
			// responses, even with domain admin access. Creator information may be available through:
			// 1. Admin SDK Reports API (audit logs) - requires additional scopes and API calls
			// 2. Google Workspace Admin Console (manual lookup)
			// For now, CreatedBy will remain nil and the code will fall back to showing creation date.

			drives = append(drives, drive)
		}

		if r.NextPageToken == "" {
			break
		}
		pageToken = r.NextPageToken
	}

	return drives, nil
}

// GetDrive retrieves a Shared Drive by ID
func (c *driveClient) GetDrive(ctx context.Context, driveID string, adminAccess bool) (*Drive, error) {
	if err := c.waitForQuota(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}
	c.logger.Debug("Getting drive: %s, adminAccess: %v", driveID, adminAccess)

	call := c.service.Drives.Get(driveID).
		Fields("id, name, createdTime, restrictions")

	if adminAccess {
		call = call.UseDomainAdminAccess(true)
	}

	d, err := call.Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get drive %s: %w", driveID, err)
	}

	c.logger.Info("Drives.Get API: driveID=%s, name=%s", driveID, d.Name)

	drive := &Drive{
		ID:          d.Id,
		Name:        d.Name,
		CreatedTime: d.CreatedTime,
	}

	// Note: The createdBy field is not available in the standard Drive API v3
	// It may be available through Admin SDK or audit logs, but not through
	// the standard Drive API endpoints.

	return drive, nil
}
