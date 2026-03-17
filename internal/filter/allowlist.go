package filter

import (
	"strings"

	"github.com/dplense/dplense-cli/pkg/models"
)

// IsInAllowlist checks if an email or domain is in the trusted allowlist
func IsInAllowlist(email, domain string, trustedDomains []string) bool {
	// Check email
	if email != "" {
		emailLower := strings.ToLower(email)
		for _, trustedDomain := range trustedDomains {
			trustedDomainLower := strings.ToLower(strings.TrimSpace(trustedDomain))
			if trustedDomainLower == "" {
				continue
			}
			// Remove @ prefix if present
			if strings.HasPrefix(trustedDomainLower, "@") {
				trustedDomainLower = trustedDomainLower[1:]
			}
			if strings.HasSuffix(emailLower, "@"+trustedDomainLower) {
				return true
			}
		}
	}

	// Check domain
	if domain != "" {
		domainLower := strings.ToLower(domain)
		for _, trustedDomain := range trustedDomains {
			trustedDomainLower := strings.ToLower(strings.TrimSpace(trustedDomain))
			if trustedDomainLower == "" {
				continue
			}
			// Remove @ prefix if present
			if strings.HasPrefix(trustedDomainLower, "@") {
				trustedDomainLower = trustedDomainLower[1:]
			}
			if domainLower == trustedDomainLower {
				return true
			}
		}
	}

	return false
}

// ShouldExcludeTrustedOnly checks if a file should be excluded because all permissions are trusted
func ShouldExcludeTrustedOnly(file models.FileIssue, trustedDomains []string) bool {
	if len(file.Permissions) == 0 {
		return false // No permissions means we can't determine, so include it
	}

	// Check if ALL permissions are trusted (or internal)
	for _, perm := range file.Permissions {
		isTrusted := false

		// Public permissions are never trusted
		if perm.Type == "anyone" {
			return false
		}

		// Check email-based permissions
		if perm.Email != "" {
			isTrusted = IsInAllowlist(perm.Email, "", trustedDomains)
		}

		// Check domain-based permissions
		if !isTrusted && perm.Domain != "" {
			isTrusted = IsInAllowlist("", perm.Domain, trustedDomains)
		}

		// If any permission is not trusted, don't exclude
		if !isTrusted {
			return false
		}
	}

	// All permissions are trusted, exclude this file
	return true
}
