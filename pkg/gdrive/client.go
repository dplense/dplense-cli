package gdrive

import "context"

// File represents a Google Drive file
type File struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	WebViewLink string   `json:"webViewLink"`
	DriveID     string   `json:"driveId"`
	Parents     []string `json:"parents"`
	Owners      []Owner  `json:"owners"`
	Labels      []string `json:"labels,omitempty"` // Human-readable label names (resolved by labels.Resolver)
}

// Owner represents a file owner
type Owner struct {
	DisplayName  string `json:"displayName"`
	EmailAddress string `json:"emailAddress"`
	Me           bool   `json:"me"`
}

// Permission represents a Google Drive permission
type Permission struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	EmailAddress  string `json:"emailAddress,omitempty"`
	Domain        string `json:"domain,omitempty"`
	Role          string `json:"role"`
	Inherited     bool   `json:"inherited,omitempty"`     // True if inherited from parent
	InheritedFrom string `json:"inheritedFrom,omitempty"` // ID of the parent folder/drive
}

// Drive represents a Shared Drive
type Drive struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	CreatedTime string  `json:"createdTime,omitempty"`
	CreatedBy   *Owner  `json:"createdBy,omitempty"` // Creator of the drive
}

// FileList represents a paginated list of files
type FileList struct {
	Files         []File `json:"files"`
	NextPageToken string `json:"nextPageToken"`
}

// DriveClient interface for Google Drive operations
type DriveClient interface {
	// ListFiles lists files matching the query with pagination
	// If driveID is specified, limits search to that specific drive
	ListFiles(ctx context.Context, query string, pageToken string, driveID string) (*FileList, error)

	// GetFile retrieves a file by ID
	GetFile(ctx context.Context, fileID string) (*File, error)

	// ListPermissions lists all permissions for a file
	ListPermissions(ctx context.Context, fileID string) ([]Permission, error)

	// DeletePermission removes a permission from a file
	DeletePermission(ctx context.Context, fileID string, permissionID string) error

	// ListDrives lists all Shared Drives
	ListDrives(ctx context.Context, adminAccess bool) ([]Drive, error)

	// GetDrive retrieves a Shared Drive by ID
	GetDrive(ctx context.Context, driveID string, adminAccess bool) (*Drive, error)

	// SetIncludeLabels sets label IDs to request in Files.List calls
	SetIncludeLabels(labelIDs string)
}
