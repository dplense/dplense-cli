package mocks

import (
	"context"
	"fmt"

	"gdrive-audit/pkg/gdrive"
)

// DriveMock is a mock implementation of DriveClient for testing
type DriveMock struct {
	FilesFunc           func(ctx context.Context, query string, pageToken string, driveID string) (*gdrive.FileList, error)
	GetFileFunc         func(ctx context.Context, fileID string) (*gdrive.File, error)
	ListPermissionsFunc func(ctx context.Context, fileID string) ([]gdrive.Permission, error)
	DeletePermissionFunc func(ctx context.Context, fileID string, permissionID string) error
	ListDrivesFunc      func(ctx context.Context, adminAccess bool) ([]gdrive.Drive, error)
	GetDriveFunc        func(ctx context.Context, driveID string, adminAccess bool) (*gdrive.Drive, error)
}

// NewDriveMock creates a new DriveMock with default implementations
func NewDriveMock() *DriveMock {
	return &DriveMock{
		FilesFunc: func(ctx context.Context, query string, pageToken string, driveID string) (*gdrive.FileList, error) {
			return &gdrive.FileList{Files: []gdrive.File{}, NextPageToken: ""}, nil
		},
		GetFileFunc: func(ctx context.Context, fileID string) (*gdrive.File, error) {
			return nil, fmt.Errorf("file not found: %s", fileID)
		},
		ListPermissionsFunc: func(ctx context.Context, fileID string) ([]gdrive.Permission, error) {
			return []gdrive.Permission{}, nil
		},
		DeletePermissionFunc: func(ctx context.Context, fileID string, permissionID string) error {
			return nil
		},
		ListDrivesFunc: func(ctx context.Context, adminAccess bool) ([]gdrive.Drive, error) {
			return []gdrive.Drive{}, nil
		},
		GetDriveFunc: func(ctx context.Context, driveID string, adminAccess bool) (*gdrive.Drive, error) {
			return nil, fmt.Errorf("drive not found: %s", driveID)
		},
		// Note: CreatedBy field is supported but defaults to nil in mock responses
	}
}

// ListFiles implements DriveClient.ListFiles
func (m *DriveMock) ListFiles(ctx context.Context, query string, pageToken string, driveID string) (*gdrive.FileList, error) {
	if m.FilesFunc != nil {
		return m.FilesFunc(ctx, query, pageToken, driveID)
	}
	return &gdrive.FileList{Files: []gdrive.File{}, NextPageToken: ""}, nil
}

// GetFile implements DriveClient.GetFile
func (m *DriveMock) GetFile(ctx context.Context, fileID string) (*gdrive.File, error) {
	if m.GetFileFunc != nil {
		return m.GetFileFunc(ctx, fileID)
	}
	return nil, fmt.Errorf("file not found: %s", fileID)
}

// ListPermissions implements DriveClient.ListPermissions
func (m *DriveMock) ListPermissions(ctx context.Context, fileID string) ([]gdrive.Permission, error) {
	if m.ListPermissionsFunc != nil {
		return m.ListPermissionsFunc(ctx, fileID)
	}
	return []gdrive.Permission{}, nil
}

// DeletePermission implements DriveClient.DeletePermission
func (m *DriveMock) DeletePermission(ctx context.Context, fileID string, permissionID string) error {
	if m.DeletePermissionFunc != nil {
		return m.DeletePermissionFunc(ctx, fileID, permissionID)
	}
	return nil
}

// ListDrives implements DriveClient.ListDrives
func (m *DriveMock) ListDrives(ctx context.Context, adminAccess bool) ([]gdrive.Drive, error) {
	if m.ListDrivesFunc != nil {
		return m.ListDrivesFunc(ctx, adminAccess)
	}
	return []gdrive.Drive{}, nil
}

// GetDrive implements DriveClient.GetDrive
func (m *DriveMock) GetDrive(ctx context.Context, driveID string, adminAccess bool) (*gdrive.Drive, error) {
	if m.GetDriveFunc != nil {
		return m.GetDriveFunc(ctx, driveID, adminAccess)
	}
	return nil, fmt.Errorf("drive not found: %s", driveID)
}

// SetIncludeLabels implements DriveClient.SetIncludeLabels (no-op in mock)
func (m *DriveMock) SetIncludeLabels(labelIDs string) {}
