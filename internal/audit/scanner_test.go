package audit

import (
	"context"
	"testing"

	"gdrive-audit/internal/logger"
	"gdrive-audit/pkg/config"
	"gdrive-audit/pkg/gdrive"
	"gdrive-audit/pkg/gdrive/mocks"
	"gdrive-audit/pkg/models"
)

func TestIsPublicPermission(t *testing.T) {
	tests := []struct {
		name       string
		permission gdrive.Permission
		expected   bool
	}{
		{
			name: "anyone permission",
			permission: gdrive.Permission{
				Type: "anyone",
				Role: "reader",
			},
			expected: true,
		},
		{
			name: "user permission",
			permission: gdrive.Permission{
				Type:         "user",
				EmailAddress: "user@example.com",
				Role:         "reader",
			},
			expected: false,
		},
		{
			name: "case insensitive",
			permission: gdrive.Permission{
				Type: "ANYONE",
				Role: "reader",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPublicPermission(tt.permission)
			if result != tt.expected {
				t.Errorf("IsPublicPermission() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestScanner_ProcessFile(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.InternalDomains = []string{"mycompany.com"}

	mockDrive := mocks.NewDriveMock()
	mockDrive.ListPermissionsFunc = func(ctx context.Context, fileID string) ([]gdrive.Permission, error) {
		return []gdrive.Permission{
			{
				ID:           "perm1",
				Type:         "user",
				EmailAddress: "external@domain.com",
				Domain:       "",
				Role:         "reader",
			},
		}, nil
	}

	log := logger.New(logger.LevelDebug, false)
	scanner := NewScanner(mockDrive, nil, cfg, log)

	file := gdrive.File{
		ID:          "file1",
		Name:        "test.pdf",
		WebViewLink: "https://drive.google.com/file1",
		Owners: []gdrive.Owner{
			{EmailAddress: "owner@mycompany.com", DisplayName: "Owner", Me: false},
		},
	}

	issue := scanner.processFile(context.Background(), file)
	if issue == nil {
		t.Error("Expected issue for external permission")
	}
	if issue.FileID != "file1" {
		t.Errorf("Expected file ID 'file1', got '%s'", issue.FileID)
	}
	if len(issue.Permissions) != 1 {
		t.Errorf("Expected 1 permission, got %d", len(issue.Permissions))
	}
	if issue.Permissions[0].RiskLevel != models.RiskMedium {
		t.Errorf("Expected risk level Medium, got %s", issue.Permissions[0].RiskLevel)
	}
}

func TestScanner_ProcessFile_InternalOnly(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.InternalDomains = []string{"mycompany.com"}

	mockDrive := mocks.NewDriveMock()
	mockDrive.ListPermissionsFunc = func(ctx context.Context, fileID string) ([]gdrive.Permission, error) {
		return []gdrive.Permission{
			{
				ID:           "perm1",
				Type:         "user",
				EmailAddress: "internal@mycompany.com",
				Domain:       "",
				Role:         "reader",
			},
		}, nil
	}

	log := logger.New(logger.LevelDebug, false)
	scanner := NewScanner(mockDrive, nil, cfg, log)

	file := gdrive.File{
		ID:   "file1",
		Name: "test.pdf",
		Owners: []gdrive.Owner{
			{EmailAddress: "owner@mycompany.com", DisplayName: "Owner", Me: false},
		},
	}

	issue := scanner.processFile(context.Background(), file)
	if issue != nil {
		t.Error("Expected nil for internal-only permissions")
	}
}

func TestScanner_ProcessFile_Public(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.InternalDomains = []string{"mycompany.com"}

	mockDrive := mocks.NewDriveMock()
	mockDrive.ListPermissionsFunc = func(ctx context.Context, fileID string) ([]gdrive.Permission, error) {
		return []gdrive.Permission{
			{
				ID:   "perm1",
				Type: "anyone",
				Role: "reader",
			},
		}, nil
	}

	log := logger.New(logger.LevelDebug, false)
	scanner := NewScanner(mockDrive, nil, cfg, log)

	file := gdrive.File{
		ID:   "file1",
		Name: "test.pdf",
	}

	issue := scanner.processFile(context.Background(), file)
	if issue == nil {
		t.Error("Expected issue for public permission")
	}
	if issue.Permissions[0].RiskLevel != models.RiskCritical {
		t.Errorf("Expected risk level Critical, got %s", issue.Permissions[0].RiskLevel)
	}
}
