package mocks

import (
	"context"
	"fmt"

	"gdrive-audit/pkg/gdrive"
)

// DirectoryMock is a mock implementation of DirectoryClient for testing
type DirectoryMock struct {
	ListUsersFunc func(ctx context.Context, suspended bool) ([]gdrive.User, error)
	GetUserFunc   func(ctx context.Context, email string) (*gdrive.User, error)
}

// NewDirectoryMock creates a new DirectoryMock with default implementations
func NewDirectoryMock() *DirectoryMock {
	return &DirectoryMock{
		ListUsersFunc: func(ctx context.Context, suspended bool) ([]gdrive.User, error) {
			return []gdrive.User{}, nil
		},
		GetUserFunc: func(ctx context.Context, email string) (*gdrive.User, error) {
			return nil, fmt.Errorf("user not found: %s", email)
		},
	}
}

// ListUsers implements DirectoryClient.ListUsers
func (m *DirectoryMock) ListUsers(ctx context.Context, suspended bool) ([]gdrive.User, error) {
	if m.ListUsersFunc != nil {
		return m.ListUsersFunc(ctx, suspended)
	}
	return []gdrive.User{}, nil
}

// GetUser implements DirectoryClient.GetUser
func (m *DirectoryMock) GetUser(ctx context.Context, email string) (*gdrive.User, error) {
	if m.GetUserFunc != nil {
		return m.GetUserFunc(ctx, email)
	}
	return nil, fmt.Errorf("user not found: %s", email)
}
