package drive

import (
	"context"
	"fmt"

	"gdrive-audit/internal/logger"
	"gdrive-audit/pkg/gdrive"

	driveapi "google.golang.org/api/drive/v3"
)

// client implements the DriveClient interface
type client struct {
	service *driveapi.Service
	logger  logger.Logger
}

// NewClient creates a new Drive client
func NewClient(service *driveapi.Service, log logger.Logger) gdrive.DriveClient {
	return &client{
		service: service,
		logger:  log,
	}
}

// ListFiles lists files matching the query with pagination
func (c *client) ListFiles(ctx context.Context, query string, pageToken string, driveID string) (*gdrive.FileList, error) {
	c.logger.Debug("Listing files with query: %s, pageToken: %s, driveID: %s", query, pageToken, driveID)

	call := c.service.Files.List().
		Q(query).
		PageSize(100).
		Fields("nextPageToken, files(id, name, webViewLink, driveId, parents, owners(displayName, emailAddress, me))").
		SupportsAllDrives(true).
		IncludeItemsFromAllDrives(true)

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

	files := make([]gdrive.File, 0, len(r.Files))
	for _, f := range r.Files {
		owners := make([]gdrive.Owner, 0, len(f.Owners))
		for _, o := range f.Owners {
			owners = append(owners, gdrive.Owner{
				DisplayName:  o.DisplayName,
				EmailAddress: o.EmailAddress,
				Me:           o.Me,
			})
		}

		files = append(files, gdrive.File{
			ID:          f.Id,
			Name:        f.Name,
			WebViewLink: f.WebViewLink,
			DriveID:     f.DriveId,
			Parents:     f.Parents,
			Owners:      owners,
		})
	}

	return &gdrive.FileList{
		Files:         files,
		NextPageToken: r.NextPageToken,
	}, nil
}

// GetFile retrieves a file by ID
func (c *client) GetFile(ctx context.Context, fileID string) (*gdrive.File, error) {
	c.logger.Debug("Getting file: %s", fileID)

	f, err := c.service.Files.Get(fileID).
		Fields("id, name, webViewLink, driveId, parents, owners(displayName, emailAddress, me)").
		SupportsAllDrives(true).
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get file %s: %w", fileID, err)
	}

	owners := make([]gdrive.Owner, 0, len(f.Owners))
	for _, o := range f.Owners {
		owners = append(owners, gdrive.Owner{
			DisplayName:  o.DisplayName,
			EmailAddress: o.EmailAddress,
			Me:           o.Me,
		})
	}

	return &gdrive.File{
		ID:          f.Id,
		Name:        f.Name,
		WebViewLink: f.WebViewLink,
		DriveID:     f.DriveId,
		Parents:     f.Parents,
		Owners:      owners,
	}, nil
}

// ListPermissions lists all permissions for a file
func (c *client) ListPermissions(ctx context.Context, fileID string) ([]gdrive.Permission, error) {
	c.logger.Debug("Listing permissions for file: %s", fileID)

	r, err := c.service.Permissions.List(fileID).
		Fields("permissions(id, type, emailAddress, domain, role, permissionDetails)").
		SupportsAllDrives(true).
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list permissions for file %s: %w", fileID, err)
	}

	permissions := make([]gdrive.Permission, 0, len(r.Permissions))
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

		permissions = append(permissions, gdrive.Permission{
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
func (c *client) DeletePermission(ctx context.Context, fileID string, permissionID string) error {
	c.logger.Debug("Deleting permission %s from file %s", permissionID, fileID)

	err := c.service.Permissions.Delete(fileID, permissionID).
		SupportsAllDrives(true).
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("failed to delete permission %s from file %s: %w", permissionID, fileID, err)
	}

	return nil
}

// ListDrives lists all Shared Drives
func (c *client) ListDrives(ctx context.Context, adminAccess bool) ([]gdrive.Drive, error) {
	c.logger.Debug("Listing drives, adminAccess: %v", adminAccess)

	call := c.service.Drives.List().
		PageSize(100).
		Fields("nextPageToken, drives(id, name, createdTime, restrictions)")

	if adminAccess {
		call = call.UseDomainAdminAccess(true)
	}

	drives := make([]gdrive.Drive, 0)
	pageToken := ""

	for {
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		r, err := call.Context(ctx).Do()
		if err != nil {
			return nil, fmt.Errorf("failed to list drives: %w", err)
		}

		for _, d := range r.Drives {
			drive := gdrive.Drive{
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
func (c *client) GetDrive(ctx context.Context, driveID string, adminAccess bool) (*gdrive.Drive, error) {
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

	drive := &gdrive.Drive{
		ID:          d.Id,
		Name:        d.Name,
		CreatedTime: d.CreatedTime,
	}

	// Note: The createdBy field is not available in the standard Drive API v3
	// It may be available through Admin SDK or audit logs, but not through
	// the standard Drive API endpoints.

	return drive, nil
}
