package slack

import (
	"testing"

	"github.com/dplense/dplense-cli/pkg/models"

	slackapi "github.com/slack-go/slack"
)

func TestMapPublicFile(t *testing.T) {
	file := slackapi.File{
		ID:        "F123",
		Name:      "report.pdf",
		Permalink: "https://slack.com/files/F123",
		User:      "U456",
	}

	issue := mapPublicFile(file)

	if issue.FileID != "F123" {
		t.Errorf("FileID = %q, want F123", issue.FileID)
	}
	if issue.FileName != "report.pdf" {
		t.Errorf("FileName = %q, want report.pdf", issue.FileName)
	}
	if issue.OwnerEmail != "U456" {
		t.Errorf("OwnerEmail = %q, want U456", issue.OwnerEmail)
	}
	if issue.DriveName != "Slack Files" {
		t.Errorf("DriveName = %q, want 'Slack Files'", issue.DriveName)
	}
	if len(issue.Permissions) != 1 {
		t.Fatalf("expected 1 permission, got %d", len(issue.Permissions))
	}
	perm := issue.Permissions[0]
	if perm.Type != "anyone" {
		t.Errorf("permission type = %q, want anyone", perm.Type)
	}
	if perm.RiskLevel != models.RiskCritical {
		t.Errorf("risk level = %v, want Critical", perm.RiskLevel)
	}
}

func TestMapFileInExtChannel(t *testing.T) {
	file := slackapi.File{
		ID:        "F789",
		Name:      "design.fig",
		Permalink: "https://slack.com/files/F789",
		User:      "U111",
	}

	issue := mapFileInExtChannel(file, "C222", "ext-partners")

	if issue.FolderPath != "#ext-partners" {
		t.Errorf("FolderPath = %q, want #ext-partners", issue.FolderPath)
	}
	if len(issue.Permissions) != 1 {
		t.Fatalf("expected 1 permission, got %d", len(issue.Permissions))
	}
	perm := issue.Permissions[0]
	if perm.Type != "group" {
		t.Errorf("permission type = %q, want group", perm.Type)
	}
	if perm.RiskLevel != models.RiskHigh {
		t.Errorf("risk level = %v, want High", perm.RiskLevel)
	}
}

func TestMapExternalChannel(t *testing.T) {
	ch := slackapi.Channel{}
	ch.ID = "C333"
	ch.Name = "shared-channel"

	issue := mapExternalChannel(ch)

	if issue.FileID != "C333" {
		t.Errorf("FileID = %q, want C333", issue.FileID)
	}
	if issue.FileName != "#shared-channel" {
		t.Errorf("FileName = %q, want #shared-channel", issue.FileName)
	}
	if issue.DriveName != "Slack Channels" {
		t.Errorf("DriveName = %q, want 'Slack Channels'", issue.DriveName)
	}
}

func TestMapGuestUser(t *testing.T) {
	user := slackapi.User{
		ID:                "U999",
		RealName:          "External Guest",
		IsRestricted:      true,
		IsUltraRestricted: false,
	}
	user.Profile.Email = "guest@external.com"

	ch := slackapi.Channel{}
	ch.ID = "C555"
	ch.Name = "general"

	issue := mapGuestUser(user, ch)

	if issue.OwnerEmail != "guest@external.com" {
		t.Errorf("OwnerEmail = %q, want guest@external.com", issue.OwnerEmail)
	}
	if issue.OwnerName != "External Guest" {
		t.Errorf("OwnerName = %q, want External Guest", issue.OwnerName)
	}
	perm := issue.Permissions[0]
	if perm.Role != "multi-channel" {
		t.Errorf("Role = %q, want multi-channel", perm.Role)
	}
	if perm.RiskLevel != models.RiskHigh {
		t.Errorf("RiskLevel = %v, want High", perm.RiskLevel)
	}

	// Test ultra-restricted (single-channel) guest
	user.IsUltraRestricted = true
	issue2 := mapGuestUser(user, ch)
	perm2 := issue2.Permissions[0]
	if perm2.Role != "single-channel" {
		t.Errorf("Role = %q, want single-channel", perm2.Role)
	}
	if perm2.RiskLevel != models.RiskMedium {
		t.Errorf("RiskLevel = %v, want Medium", perm2.RiskLevel)
	}
}

func TestProviderScopes(t *testing.T) {
	p := &slackProvider{}
	scopes := p.Scopes()
	expected := []string{"files", "channels", "guests", "all"}

	if len(scopes) != len(expected) {
		t.Fatalf("len(scopes) = %d, want %d", len(scopes), len(expected))
	}
	for i, s := range scopes {
		if s != expected[i] {
			t.Errorf("scope[%d] = %q, want %q", i, s, expected[i])
		}
	}
}
