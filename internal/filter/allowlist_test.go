package filter

import (
	"testing"

	"github.com/dplense/dplense-cli/pkg/models"
)

func TestIsInAllowlist(t *testing.T) {
	trustedDomains := []string{"partner.com", "vendor.com"}

	tests := []struct {
		name     string
		email    string
		domain   string
		expected bool
	}{
		{"trusted email", "user@partner.com", "", true},
		{"trusted domain", "", "partner.com", true},
		{"untrusted email", "user@external.com", "", false},
		{"untrusted domain", "", "external.com", false},
		{"case insensitive email", "USER@PARTNER.COM", "", true},
		{"case insensitive domain", "", "PARTNER.COM", true},
		{"empty inputs", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsInAllowlist(tt.email, tt.domain, trustedDomains)
			if result != tt.expected {
				t.Errorf("IsInAllowlist(%q, %q) = %v, want %v", tt.email, tt.domain, result, tt.expected)
			}
		})
	}
}

func TestShouldExcludeTrustedOnly(t *testing.T) {
	trustedDomains := []string{"partner.com"}

	tests := []struct {
		name    string
		file    models.FileIssue
		exclude bool
	}{
		{
			name: "all trusted permissions",
			file: models.FileIssue{
				Permissions: []models.Permission{
					{Email: "user1@partner.com", Type: "user", Role: "reader"},
					{Domain: "partner.com", Type: "domain", Role: "reader"},
				},
			},
			exclude: true,
		},
		{
			name: "has untrusted permission",
			file: models.FileIssue{
				Permissions: []models.Permission{
					{Email: "user1@partner.com", Type: "user", Role: "reader"},
					{Email: "external@domain.com", Type: "user", Role: "reader"},
				},
			},
			exclude: false,
		},
		{
			name: "has public permission",
			file: models.FileIssue{
				Permissions: []models.Permission{
					{Email: "user1@partner.com", Type: "user", Role: "reader"},
					{Type: "anyone", Role: "reader"},
				},
			},
			exclude: false,
		},
		{
			name: "no permissions",
			file: models.FileIssue{
				Permissions: []models.Permission{},
			},
			exclude: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldExcludeTrustedOnly(tt.file, trustedDomains)
			if result != tt.exclude {
				t.Errorf("ShouldExcludeTrustedOnly() = %v, want %v", result, tt.exclude)
			}
		})
	}
}
