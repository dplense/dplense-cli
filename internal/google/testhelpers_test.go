package google

import (
	"context"
	"fmt"
)

// inlineDriveMock is an inline mock for DriveClient used by tests in this package.
// This avoids the import cycle that would occur from importing internal/google/mocks.
type inlineDriveMock struct {
	ListFilesFunc        func(ctx context.Context, query string, pageToken string, driveID string) (*FileList, error)
	GetFileFunc          func(ctx context.Context, fileID string) (*File, error)
	ListPermissionsFunc  func(ctx context.Context, fileID string) ([]Permission, error)
	DeletePermissionFunc func(ctx context.Context, fileID string, permissionID string) error
	ListDrivesFunc       func(ctx context.Context, adminAccess bool) ([]Drive, error)
	GetDriveFunc         func(ctx context.Context, driveID string, adminAccess bool) (*Drive, error)
}

func newInlineDriveMock() *inlineDriveMock {
	return &inlineDriveMock{
		ListFilesFunc: func(ctx context.Context, query string, pageToken string, driveID string) (*FileList, error) {
			return &FileList{Files: []File{}, NextPageToken: ""}, nil
		},
		GetFileFunc: func(ctx context.Context, fileID string) (*File, error) {
			return nil, fmt.Errorf("file not found: %s", fileID)
		},
		ListPermissionsFunc: func(ctx context.Context, fileID string) ([]Permission, error) {
			return []Permission{}, nil
		},
		DeletePermissionFunc: func(ctx context.Context, fileID string, permissionID string) error {
			return nil
		},
		ListDrivesFunc: func(ctx context.Context, adminAccess bool) ([]Drive, error) {
			return []Drive{}, nil
		},
		GetDriveFunc: func(ctx context.Context, driveID string, adminAccess bool) (*Drive, error) {
			return nil, fmt.Errorf("drive not found: %s", driveID)
		},
	}
}

func (m *inlineDriveMock) ListFiles(ctx context.Context, query string, pageToken string, driveID string) (*FileList, error) {
	return m.ListFilesFunc(ctx, query, pageToken, driveID)
}
func (m *inlineDriveMock) GetFile(ctx context.Context, fileID string) (*File, error) {
	return m.GetFileFunc(ctx, fileID)
}
func (m *inlineDriveMock) ListPermissions(ctx context.Context, fileID string) ([]Permission, error) {
	return m.ListPermissionsFunc(ctx, fileID)
}
func (m *inlineDriveMock) DeletePermission(ctx context.Context, fileID string, permissionID string) error {
	return m.DeletePermissionFunc(ctx, fileID, permissionID)
}
func (m *inlineDriveMock) ListDrives(ctx context.Context, adminAccess bool) ([]Drive, error) {
	return m.ListDrivesFunc(ctx, adminAccess)
}
func (m *inlineDriveMock) GetDrive(ctx context.Context, driveID string, adminAccess bool) (*Drive, error) {
	return m.GetDriveFunc(ctx, driveID, adminAccess)
}
func (m *inlineDriveMock) SetIncludeLabels(labelIDs string) {}

// inlineDirectoryMock is an inline mock for DirectoryClient used by tests in this package.
type inlineDirectoryMock struct {
	ListUsersFunc func(ctx context.Context, suspended bool) ([]User, error)
	GetUserFunc   func(ctx context.Context, email string) (*User, error)
}

func newInlineDirectoryMock() *inlineDirectoryMock {
	return &inlineDirectoryMock{
		ListUsersFunc: func(ctx context.Context, suspended bool) ([]User, error) {
			return []User{}, nil
		},
		GetUserFunc: func(ctx context.Context, email string) (*User, error) {
			return nil, fmt.Errorf("user not found: %s", email)
		},
	}
}

func (m *inlineDirectoryMock) ListUsers(ctx context.Context, suspended bool) ([]User, error) {
	return m.ListUsersFunc(ctx, suspended)
}
func (m *inlineDirectoryMock) GetUser(ctx context.Context, email string) (*User, error) {
	return m.GetUserFunc(ctx, email)
}
