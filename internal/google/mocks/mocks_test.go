package mocks

import (
	"context"
	"testing"

	google "github.com/dplense/dplense-cli/internal/google"
)

func TestDriveMock_ListFiles(t *testing.T) {
	mock := NewDriveMock()
	result, err := mock.ListFiles(context.Background(), "test", "", "")
	if err != nil {
		t.Errorf("ListFiles() error = %v", err)
	}
	if result == nil {
		t.Error("ListFiles() returned nil")
	}
}

func TestDriveMock_GetFile(t *testing.T) {
	mock := NewDriveMock()
	_, err := mock.GetFile(context.Background(), "test-id")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestDirectoryMock_ListUsers(t *testing.T) {
	mock := NewDirectoryMock()
	users, err := mock.ListUsers(context.Background(), false)
	if err != nil {
		t.Errorf("ListUsers() error = %v", err)
	}
	if users == nil {
		t.Error("ListUsers() returned nil")
	}
}

func TestDirectoryMock_GetUser(t *testing.T) {
	mock := NewDirectoryMock()
	_, err := mock.GetUser(context.Background(), "test@example.com")
	if err == nil {
		t.Error("Expected error for non-existent user")
	}
}

// Test that mocks implement the interfaces
func TestDriveMock_ImplementsInterface(t *testing.T) {
	var _ google.DriveClient = NewDriveMock()
}

func TestDirectoryMock_ImplementsInterface(t *testing.T) {
	var _ google.DirectoryClient = NewDirectoryMock()
}
