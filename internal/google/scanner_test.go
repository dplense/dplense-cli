package google

import (
	"context"
	"testing"

	"github.com/dplense/dplense-cli/internal/logger"
	"github.com/dplense/dplense-cli/pkg/config"
	"github.com/dplense/dplense-cli/pkg/models"
)

func Test_isPublicPermission(t *testing.T) {
	tests := []struct {
		name       string
		permission Permission
		expected   bool
	}{
		{
			name: "anyone permission",
			permission: Permission{
				Type: "anyone",
				Role: "reader",
			},
			expected: true,
		},
		{
			name: "user permission",
			permission: Permission{
				Type:         "user",
				EmailAddress: "user@example.com",
				Role:         "reader",
			},
			expected: false,
		},
		{
			name: "case insensitive",
			permission: Permission{
				Type: "ANYONE",
				Role: "reader",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPublicPermission(tt.permission)
			if result != tt.expected {
				t.Errorf("isPublicPermission() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestScanner_ProcessFile(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.InternalDomains = []string{"mycompany.com"}

	mockDrive := newInlineDriveMock()
	mockDrive.ListPermissionsFunc = func(ctx context.Context, fileID string) ([]Permission, error) {
		return []Permission{
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
	scanner := newScanner(mockDrive, nil, cfg, log)

	file := File{
		ID:          "file1",
		Name:        "test.pdf",
		WebViewLink: "https://drive.google.com/file1",
		Owners: []Owner{
			{EmailAddress: "owner@mycompany.com", DisplayName: "Owner", Me: false},
		},
	}

	issue := scanner.processFile(context.Background(), file, scanOptions{})
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

	mockDrive := newInlineDriveMock()
	mockDrive.ListPermissionsFunc = func(ctx context.Context, fileID string) ([]Permission, error) {
		return []Permission{
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
	scanner := newScanner(mockDrive, nil, cfg, log)

	file := File{
		ID:   "file1",
		Name: "test.pdf",
		Owners: []Owner{
			{EmailAddress: "owner@mycompany.com", DisplayName: "Owner", Me: false},
		},
	}

	issue := scanner.processFile(context.Background(), file, scanOptions{})
	if issue != nil {
		t.Error("Expected nil for internal-only permissions")
	}
}

func TestScanner_ProcessFile_Public(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.InternalDomains = []string{"mycompany.com"}

	mockDrive := newInlineDriveMock()
	mockDrive.ListPermissionsFunc = func(ctx context.Context, fileID string) ([]Permission, error) {
		return []Permission{
			{
				ID:   "perm1",
				Type: "anyone",
				Role: "reader",
			},
		}, nil
	}

	log := logger.New(logger.LevelDebug, false)
	scanner := newScanner(mockDrive, nil, cfg, log)

	file := File{
		ID:   "file1",
		Name: "test.pdf",
	}

	issue := scanner.processFile(context.Background(), file, scanOptions{})
	if issue == nil {
		t.Error("Expected issue for public permission")
	}
	if issue.Permissions[0].RiskLevel != models.RiskCritical {
		t.Errorf("Expected risk level Critical, got %s", issue.Permissions[0].RiskLevel)
	}
}
