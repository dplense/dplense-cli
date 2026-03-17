package mocks

import (
	"context"
	"fmt"

	google "github.com/dplense/dplense-cli/internal/google"
)

// DirectoryMock is a mock implementation of DirectoryClient for testing
type DirectoryMock struct {
	ListUsersFunc func(ctx context.Context, suspended bool) ([]google.User, error)
	GetUserFunc   func(ctx context.Context, email string) (*google.User, error)
}

// NewDirectoryMock creates a new DirectoryMock with default implementations
func NewDirectoryMock() *DirectoryMock {
	return &DirectoryMock{
		ListUsersFunc: func(ctx context.Context, suspended bool) ([]google.User, error) {
			return []google.User{}, nil
		},
		GetUserFunc: func(ctx context.Context, email string) (*google.User, error) {
			return nil, fmt.Errorf("user not found: %s", email)
		},
	}
}

// ListUsers implements DirectoryClient.ListUsers
func (m *DirectoryMock) ListUsers(ctx context.Context, suspended bool) ([]google.User, error) {
	if m.ListUsersFunc != nil {
		return m.ListUsersFunc(ctx, suspended)
	}
	return []google.User{}, nil
}

// GetUser implements DirectoryClient.GetUser
func (m *DirectoryMock) GetUser(ctx context.Context, email string) (*google.User, error) {
	if m.GetUserFunc != nil {
		return m.GetUserFunc(ctx, email)
	}
	return nil, fmt.Errorf("user not found: %s", email)
}
