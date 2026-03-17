package filter

import (
	"strings"

	"github.com/dplense/dplense-cli/pkg/models"
)

// IsInternalDomain checks if an email belongs to an internal domain
func IsInternalDomain(email string, domains []string) bool {
	if email == "" {
		return false
	}

	emailLower := strings.ToLower(email)
	for _, domain := range domains {
		domainLower := strings.ToLower(strings.TrimSpace(domain))
		if domainLower == "" {
			continue
		}
		// Remove @ prefix if present
		if strings.HasPrefix(domainLower, "@") {
			domainLower = domainLower[1:]
		}
		if strings.HasSuffix(emailLower, "@"+domainLower) {
			return true
		}
	}
	return false
}

// ShouldExcludeInternalOnly checks if a file should be excluded because all permissions are internal
func ShouldExcludeInternalOnly(file models.FileIssue, internalDomains []string) bool {
	if len(file.Permissions) == 0 {
		return false // No permissions means we can't determine, so include it
	}

	// Check if ALL permissions are internal
	for _, perm := range file.Permissions {
		// Public permissions are never internal, so don't exclude
		if strings.ToLower(perm.Type) == "anyone" {
			return false
		}

		isInternal := false

		// Check email-based permissions
		if perm.Email != "" {
			isInternal = IsInternalDomain(perm.Email, internalDomains)
		}

		// Check domain-based permissions
		if !isInternal && perm.Domain != "" {
			domainLower := strings.ToLower(perm.Domain)
			for _, internalDomain := range internalDomains {
				internalDomainLower := strings.ToLower(strings.TrimSpace(internalDomain))
				if strings.HasPrefix(internalDomainLower, "@") {
					internalDomainLower = internalDomainLower[1:]
				}
				if domainLower == internalDomainLower {
					isInternal = true
					break
				}
			}
		}

		// If any permission is external, don't exclude
		if !isInternal {
			return false
		}
	}

	// All permissions are internal, exclude this file
	return true
}
