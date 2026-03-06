package revoke

import (
	"context"
	"fmt"
	"testing"

	"gdrive-audit/internal/logger"
	"gdrive-audit/pkg/config"
	"gdrive-audit/pkg/gdrive"
	"gdrive-audit/pkg/gdrive/mocks"
	"gdrive-audit/pkg/models"
)

func newScanResultWithIssues(issues []models.FileIssue) *models.ScanResult {
	result := models.NewScanResult("test")
	for _, issue := range issues {
		result.AddIssue(issue)
	}
	return result
}

func TestRevokeUserFromFiles_MultipleFiles(t *testing.T) {
	cfg := config.DefaultConfig()

	type deletion struct {
		fileID string
		permID string
	}
	var deletions []deletion

	mock := mocks.NewDriveMock()
	mock.ListPermissionsFunc = func(ctx context.Context, fileID string) ([]gdrive.Permission, error) {
		return []gdrive.Permission{
			{ID: "perm-" + fileID, Type: "user", EmailAddress: "external@other.com", Role: "reader"},
		}, nil
	}
	mock.DeletePermissionFunc = func(ctx context.Context, fileID string, permID string) error {
		deletions = append(deletions, deletion{fileID: fileID, permID: permID})
		return nil
	}

	log := logger.New(logger.LevelDebug, false)
	revoker := NewUserRevoker(mock, cfg, log, false)

	scanResult := newScanResultWithIssues([]models.FileIssue{
		{
			FileID:   "file-1",
			FileName: "doc1.pdf",
			Permissions: []models.Permission{
				{ID: "perm-file-1", Email: "external@other.com", Role: "reader"},
			},
		},
		{
			FileID:   "file-2",
			FileName: "doc2.pdf",
			Permissions: []models.Permission{
				{ID: "perm-file-2", Email: "external@other.com", Role: "writer"},
			},
		},
	})

	_, _, err := revoker.RevokeUserFromFiles(context.Background(), "external@other.com", scanResult)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deletions) != 2 {
		t.Errorf("expected 2 deletions, got %d", len(deletions))
	}
}

func TestRevokeUserFromFiles_DryRun(t *testing.T) {
	cfg := config.DefaultConfig()

	deleteCount := 0
	mock := mocks.NewDriveMock()
	mock.ListPermissionsFunc = func(ctx context.Context, fileID string) ([]gdrive.Permission, error) {
		return []gdrive.Permission{
			{ID: "perm-1", Type: "user", EmailAddress: "external@other.com", Role: "reader"},
		}, nil
	}
	mock.DeletePermissionFunc = func(ctx context.Context, fileID string, permID string) error {
		deleteCount++
		return nil
	}

	log := logger.New(logger.LevelDebug, false)
	revoker := NewUserRevoker(mock, cfg, log, true) // dryRun = true

	scanResult := newScanResultWithIssues([]models.FileIssue{
		{
			FileID:   "file-1",
			FileName: "doc1.pdf",
			Permissions: []models.Permission{
				{ID: "perm-1", Email: "external@other.com", Role: "reader"},
			},
		},
	})

	_, _, err := revoker.RevokeUserFromFiles(context.Background(), "external@other.com", scanResult)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleteCount != 0 {
		t.Errorf("expected 0 deletions in dry-run, got %d", deleteCount)
	}
}

func TestRevokeUserFromFiles_ListPermissionsError(t *testing.T) {
	cfg := config.DefaultConfig()

	callCount := 0
	mock := mocks.NewDriveMock()
	mock.ListPermissionsFunc = func(ctx context.Context, fileID string) ([]gdrive.Permission, error) {
		callCount++
		if fileID == "file-1" {
			return nil, fmt.Errorf("access denied")
		}
		return []gdrive.Permission{
			{ID: "perm-2", Type: "user", EmailAddress: "external@other.com", Role: "reader"},
		}, nil
	}

	var deletedFiles []string
	mock.DeletePermissionFunc = func(ctx context.Context, fileID string, permID string) error {
		deletedFiles = append(deletedFiles, fileID)
		return nil
	}

	log := logger.New(logger.LevelDebug, false)
	revoker := NewUserRevoker(mock, cfg, log, false)

	scanResult := newScanResultWithIssues([]models.FileIssue{
		{
			FileID:   "file-1",
			FileName: "doc1.pdf",
			Permissions: []models.Permission{
				{ID: "perm-1", Email: "external@other.com", Role: "reader"},
			},
		},
		{
			FileID:   "file-2",
			FileName: "doc2.pdf",
			Permissions: []models.Permission{
				{ID: "perm-2", Email: "external@other.com", Role: "reader"},
			},
		},
	})

	_, _, err := revoker.RevokeUserFromFiles(context.Background(), "external@other.com", scanResult)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should continue processing file-2 despite file-1 error
	if callCount != 2 {
		t.Errorf("expected 2 ListPermissions calls, got %d", callCount)
	}
	if len(deletedFiles) != 1 || deletedFiles[0] != "file-2" {
		t.Errorf("expected [file-2] to be deleted, got %v", deletedFiles)
	}
}

func TestRevokeUserFromFiles_NoMatchingFiles(t *testing.T) {
	cfg := config.DefaultConfig()

	listCallCount := 0
	mock := mocks.NewDriveMock()
	mock.ListPermissionsFunc = func(ctx context.Context, fileID string) ([]gdrive.Permission, error) {
		listCallCount++
		return nil, nil
	}

	log := logger.New(logger.LevelDebug, false)
	revoker := NewUserRevoker(mock, cfg, log, false)

	scanResult := newScanResultWithIssues([]models.FileIssue{
		{
			FileID:   "file-1",
			FileName: "doc1.pdf",
			Permissions: []models.Permission{
				{ID: "perm-1", Email: "someone-else@other.com", Role: "reader"},
			},
		},
	})

	_, _, err := revoker.RevokeUserFromFiles(context.Background(), "external@other.com", scanResult)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if listCallCount != 0 {
		t.Errorf("expected 0 ListPermissions calls when no files match, got %d", listCallCount)
	}
}
