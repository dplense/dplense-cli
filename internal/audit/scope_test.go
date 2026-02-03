package audit

import (
	"context"
	"testing"

	"gdrive-audit/internal/logger"
	"gdrive-audit/pkg/gdrive"
	"gdrive-audit/pkg/gdrive/mocks"
)

func TestResolveScope_Active(t *testing.T) {
	mockDir := mocks.NewDirectoryMock()
	mockDir.ListUsersFunc = func(ctx context.Context, suspended bool) ([]gdrive.User, error) {
		return []gdrive.User{
			{ID: "1", Email: "user1@example.com", DisplayName: "User 1"},
			{ID: "2", Email: "user2@example.com", DisplayName: "User 2"},
		}, nil
	}

	log := logger.New(logger.LevelDebug, false)
	targets, err := ResolveScope("active", mockDir, nil, log)
	if err != nil {
		t.Fatalf("ResolveScope() error = %v", err)
	}
	if len(targets) != 2 {
		t.Errorf("Expected 2 targets, got %d", len(targets))
	}
	if targets[0].Type != "user" {
		t.Errorf("Expected type 'user', got '%s'", targets[0].Type)
	}
}

func TestResolveScope_User(t *testing.T) {
	mockDir := mocks.NewDirectoryMock()
	mockDir.GetUserFunc = func(ctx context.Context, email string) (*gdrive.User, error) {
		return &gdrive.User{
			ID:          "1",
			Email:       email,
			DisplayName: "Test User",
		}, nil
	}

	log := logger.New(logger.LevelDebug, false)
	targets, err := ResolveScope("user:test@example.com", mockDir, nil, log)
	if err != nil {
		t.Fatalf("ResolveScope() error = %v", err)
	}
	if len(targets) != 1 {
		t.Errorf("Expected 1 target, got %d", len(targets))
	}
	if targets[0].Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", targets[0].Email)
	}
}

func TestResolveScope_SharedDrives(t *testing.T) {
	mockDrive := mocks.NewDriveMock()
	mockDrive.ListDrivesFunc = func(ctx context.Context, adminAccess bool) ([]gdrive.Drive, error) {
		return []gdrive.Drive{
			{ID: "drive1", Name: "Drive 1"},
			{ID: "drive2", Name: "Drive 2"},
		}, nil
	}

	log := logger.New(logger.LevelDebug, false)
	targets, err := ResolveScope("shared-drives", nil, mockDrive, log)
	if err != nil {
		t.Fatalf("ResolveScope() error = %v", err)
	}
	if len(targets) != 2 {
		t.Errorf("Expected 2 targets, got %d", len(targets))
	}
	if targets[0].Type != "drive" {
		t.Errorf("Expected type 'drive', got '%s'", targets[0].Type)
	}
	if targets[0].ID != "drive1" {
		t.Errorf("Expected drive ID 'drive1', got '%s'", targets[0].ID)
	}
	if targets[0].Name != "Drive 1" {
		t.Errorf("Expected drive name 'Drive 1', got '%s'", targets[0].Name)
	}
}

func TestResolveScope_SharedDrives_WithCreator(t *testing.T) {
	mockDrive := mocks.NewDriveMock()
	mockDrive.ListDrivesFunc = func(ctx context.Context, adminAccess bool) ([]gdrive.Drive, error) {
		return []gdrive.Drive{
			{
				ID:   "drive1",
				Name: "Drive 1",
				CreatedBy: &gdrive.Owner{
					EmailAddress: "creator@example.com",
					DisplayName:  "Creator Name",
				},
			},
			{
				ID:          "drive2",
				Name:        "Drive 2",
				CreatedTime: "2024-01-15T10:30:00Z",
				// No CreatedBy - should fallback to creation date
			},
		}, nil
	}

	log := logger.New(logger.LevelDebug, false)
	targets, err := ResolveScope("shared-drives", nil, mockDrive, log)
	if err != nil {
		t.Fatalf("ResolveScope() error = %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("Expected 2 targets, got %d", len(targets))
	}

	// First drive should use creator email
	if targets[0].Owner != "creator@example.com" {
		t.Errorf("Expected owner 'creator@example.com', got '%s'", targets[0].Owner)
	}

	// Second drive should fallback to creation date
	if targets[1].Owner != "2024-01-15" {
		t.Errorf("Expected owner '2024-01-15', got '%s'", targets[1].Owner)
	}
}

func TestResolveScope_SharedDrives_NoCreatorFallback(t *testing.T) {
	mockDrive := mocks.NewDriveMock()
	mockDrive.ListDrivesFunc = func(ctx context.Context, adminAccess bool) ([]gdrive.Drive, error) {
		return []gdrive.Drive{
			{
				ID:   "drive1",
				Name: "Drive 1",
				// No CreatedBy and no CreatedTime - should use "Unknown"
			},
		}, nil
	}

	log := logger.New(logger.LevelDebug, false)
	targets, err := ResolveScope("shared-drives", nil, mockDrive, log)
	if err != nil {
		t.Fatalf("ResolveScope() error = %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("Expected 1 target, got %d", len(targets))
	}

	// Should fallback to "Unknown" when no creator or creation time
	if targets[0].Owner != "Unknown" {
		t.Errorf("Expected owner 'Unknown', got '%s'", targets[0].Owner)
	}
}

func TestResolveScope_Invalid(t *testing.T) {
	log := logger.New(logger.LevelDebug, false)
	_, err := ResolveScope("invalid", nil, nil, log)
	if err == nil {
		t.Error("Expected error for invalid scope")
	}
}
