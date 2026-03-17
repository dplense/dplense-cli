package google

import (
	"strings"
	"testing"
)

func TestGetScopes(t *testing.T) {
	scopes := GetScopes()
	if len(scopes) == 0 {
		t.Error("Expected at least one scope")
	}
}

func TestDriveScopes(t *testing.T) {
	scopes := DriveScopes()
	if len(scopes) == 0 {
		t.Error("Expected at least one Drive scope")
	}
}

func TestDirectoryScopes(t *testing.T) {
	scopes := DirectoryScopes()
	if len(scopes) == 0 {
		t.Error("Expected at least one Directory scope")
	}
}

func TestFormatScopesForDisplay(t *testing.T) {
	scopes := []string{"scope1", "scope2"}
	display := FormatScopesForDisplay(scopes)

	if !strings.Contains(display, "scope1") {
		t.Error("Expected scope1 in display")
	}
	if !strings.Contains(display, "scope2") {
		t.Error("Expected scope2 in display")
	}
}

func TestGetDomainWideDelegationInstructions(t *testing.T) {
	instructions := GetDomainWideDelegationInstructions("test-client-id")
	if !strings.Contains(instructions, "test-client-id") {
		t.Error("Expected client ID in instructions")
	}
	if !strings.Contains(instructions, "admin.google.com") {
		t.Error("Expected admin console URL in instructions")
	}
}
