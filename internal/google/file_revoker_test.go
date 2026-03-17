package google

import (
	"context"
	"testing"

	"github.com/dplense/dplense-cli/internal/logger"
	"github.com/dplense/dplense-cli/pkg/config"
)

func TestRevokeExternalPermissions_ExternalUserRevoked(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.InternalDomains = []string{"mycompany.com"}
	cfg.TrustedDomains = []string{}

	var deletedPerms []string
	mock := newInlineDriveMock()
	mock.ListPermissionsFunc = func(ctx context.Context, fileID string) ([]Permission, error) {
		return []Permission{
			{ID: "perm-internal", Type: "user", EmailAddress: "alice@mycompany.com", Role: "writer"},
			{ID: "perm-external", Type: "user", EmailAddress: "bob@external.com", Role: "reader"},
		}, nil
	}
	mock.DeletePermissionFunc = func(ctx context.Context, fileID string, permID string) error {
		deletedPerms = append(deletedPerms, permID)
		return nil
	}

	log := logger.New(logger.LevelDebug, false)
	revoker := newFileRevoker(mock, cfg, log, false)

	_, err := revoker.revokeExternalPermissions(context.Background(), "file-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deletedPerms) != 1 || deletedPerms[0] != "perm-external" {
		t.Errorf("expected [perm-external] to be deleted, got %v", deletedPerms)
	}
}

func TestRevokeExternalPermissions_InternalNotRevoked(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.InternalDomains = []string{"mycompany.com"}

	deleteCount := 0
	mock := newInlineDriveMock()
	mock.ListPermissionsFunc = func(ctx context.Context, fileID string) ([]Permission, error) {
		return []Permission{
			{ID: "perm1", Type: "user", EmailAddress: "alice@mycompany.com", Role: "writer"},
			{ID: "perm2", Type: "user", EmailAddress: "bob@mycompany.com", Role: "reader"},
		}, nil
	}
	mock.DeletePermissionFunc = func(ctx context.Context, fileID string, permID string) error {
		deleteCount++
		return nil
	}

	log := logger.New(logger.LevelDebug, false)
	revoker := newFileRevoker(mock, cfg, log, false)

	_, err := revoker.revokeExternalPermissions(context.Background(), "file-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleteCount != 0 {
		t.Errorf("expected 0 deletions for internal-only perms, got %d", deleteCount)
	}
}

func TestRevokeExternalPermissions_OwnerNotRevoked(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.InternalDomains = []string{"mycompany.com"}

	deleteCount := 0
	mock := newInlineDriveMock()
	mock.ListPermissionsFunc = func(ctx context.Context, fileID string) ([]Permission, error) {
		return []Permission{
			{ID: "perm-owner", Type: "user", EmailAddress: "external-owner@other.com", Role: "owner"},
		}, nil
	}
	mock.DeletePermissionFunc = func(ctx context.Context, fileID string, permID string) error {
		deleteCount++
		return nil
	}

	log := logger.New(logger.LevelDebug, false)
	revoker := newFileRevoker(mock, cfg, log, false)

	_, err := revoker.revokeExternalPermissions(context.Background(), "file-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleteCount != 0 {
		t.Errorf("expected 0 deletions for owner perm, got %d", deleteCount)
	}
}

func TestRevokeExternalPermissions_DryRun(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.InternalDomains = []string{"mycompany.com"}

	deleteCount := 0
	mock := newInlineDriveMock()
	mock.ListPermissionsFunc = func(ctx context.Context, fileID string) ([]Permission, error) {
		return []Permission{
			{ID: "perm-ext", Type: "user", EmailAddress: "ext@other.com", Role: "reader"},
		}, nil
	}
	mock.DeletePermissionFunc = func(ctx context.Context, fileID string, permID string) error {
		deleteCount++
		return nil
	}

	log := logger.New(logger.LevelDebug, false)
	revoker := newFileRevoker(mock, cfg, log, true) // dryRun = true

	_, err := revoker.revokeExternalPermissions(context.Background(), "file-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleteCount != 0 {
		t.Errorf("expected 0 deletions in dry-run mode, got %d", deleteCount)
	}
}

func TestRevokeExternalPermissions_PublicAnyoneRevoked(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.InternalDomains = []string{"mycompany.com"}

	var deletedPerms []string
	mock := newInlineDriveMock()
	mock.ListPermissionsFunc = func(ctx context.Context, fileID string) ([]Permission, error) {
		return []Permission{
			{ID: "perm-anyone", Type: "anyone", Role: "reader"},
		}, nil
	}
	mock.DeletePermissionFunc = func(ctx context.Context, fileID string, permID string) error {
		deletedPerms = append(deletedPerms, permID)
		return nil
	}

	log := logger.New(logger.LevelDebug, false)
	revoker := newFileRevoker(mock, cfg, log, false)

	_, err := revoker.revokeExternalPermissions(context.Background(), "file-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deletedPerms) != 1 || deletedPerms[0] != "perm-anyone" {
		t.Errorf("expected [perm-anyone] to be deleted, got %v", deletedPerms)
	}
}

func TestRevokeUserFromFile_SpecificUserRevoked(t *testing.T) {
	cfg := config.DefaultConfig()

	var deletedPerms []string
	mock := newInlineDriveMock()
	mock.ListPermissionsFunc = func(ctx context.Context, fileID string) ([]Permission, error) {
		return []Permission{
			{ID: "perm-alice", Type: "user", EmailAddress: "alice@external.com", Role: "writer"},
			{ID: "perm-bob", Type: "user", EmailAddress: "bob@external.com", Role: "reader"},
		}, nil
	}
	mock.DeletePermissionFunc = func(ctx context.Context, fileID string, permID string) error {
		deletedPerms = append(deletedPerms, permID)
		return nil
	}

	log := logger.New(logger.LevelDebug, false)
	revoker := newFileRevoker(mock, cfg, log, false)

	err := revoker.revokeUserFromFile(context.Background(), "file-1", "bob@external.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deletedPerms) != 1 || deletedPerms[0] != "perm-bob" {
		t.Errorf("expected [perm-bob] to be deleted, got %v", deletedPerms)
	}
}

func TestRevokeUserFromFile_OwnerCannotBeRevoked(t *testing.T) {
	cfg := config.DefaultConfig()

	mock := newInlineDriveMock()
	mock.ListPermissionsFunc = func(ctx context.Context, fileID string) ([]Permission, error) {
		return []Permission{
			{ID: "perm-owner", Type: "user", EmailAddress: "owner@company.com", Role: "owner"},
		}, nil
	}

	log := logger.New(logger.LevelDebug, false)
	revoker := newFileRevoker(mock, cfg, log, false)

	err := revoker.revokeUserFromFile(context.Background(), "file-1", "owner@company.com")
	if err == nil {
		t.Error("expected error when revoking owner, got nil")
	}
}

func TestRevokeUserFromFile_UserNotFound(t *testing.T) {
	cfg := config.DefaultConfig()

	deleteCount := 0
	mock := newInlineDriveMock()
	mock.ListPermissionsFunc = func(ctx context.Context, fileID string) ([]Permission, error) {
		return []Permission{
			{ID: "perm-alice", Type: "user", EmailAddress: "alice@external.com", Role: "writer"},
		}, nil
	}
	mock.DeletePermissionFunc = func(ctx context.Context, fileID string, permID string) error {
		deleteCount++
		return nil
	}

	log := logger.New(logger.LevelDebug, false)
	revoker := newFileRevoker(mock, cfg, log, false)

	err := revoker.revokeUserFromFile(context.Background(), "file-1", "nonexistent@other.com")
	if err == nil {
		t.Fatal("expected error for non-existent user, got nil")
	}
	if deleteCount != 0 {
		t.Errorf("expected 0 deletions for non-existent user, got %d", deleteCount)
	}
}

func TestRevokeUserFromFile_DryRun(t *testing.T) {
	cfg := config.DefaultConfig()

	deleteCount := 0
	mock := newInlineDriveMock()
	mock.ListPermissionsFunc = func(ctx context.Context, fileID string) ([]Permission, error) {
		return []Permission{
			{ID: "perm-bob", Type: "user", EmailAddress: "bob@external.com", Role: "reader"},
		}, nil
	}
	mock.DeletePermissionFunc = func(ctx context.Context, fileID string, permID string) error {
		deleteCount++
		return nil
	}

	log := logger.New(logger.LevelDebug, false)
	revoker := newFileRevoker(mock, cfg, log, true) // dryRun = true

	err := revoker.revokeUserFromFile(context.Background(), "file-1", "bob@external.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleteCount != 0 {
		t.Errorf("expected 0 deletions in dry-run mode, got %d", deleteCount)
	}
}
