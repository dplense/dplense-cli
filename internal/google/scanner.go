package google

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dplense/dplense-cli/internal/filter"
	"github.com/dplense/dplense-cli/internal/logger"
	"github.com/dplense/dplense-cli/pkg/config"
	"github.com/dplense/dplense-cli/pkg/models"
)

// scanOptions holds immutable options for a single scan invocation.
// Passed by value to avoid race conditions with concurrent worker goroutines.
type scanOptions struct {
	FilterSharedWith string   // Only show files shared with this email/pattern
	FilterPublicOnly bool     // Only show files with public "anyone" links
	FilterRiskLevels []string // Only show issues at these risk levels (empty = all)
}

// scanner performs security audits on Google Drive files
type scanner struct {
	driveClient        DriveClient
	dirClient          DirectoryClient
	config             *config.Config
	logger             logger.Logger
	workerCount        int
	progressFunc       func(filesScanned, issuesFound int)
	targetProgressFunc func(targetIdx, targetTotal int, targetName string)
	statusFunc         func(msg string) // prints a line to user output (no log prefix)
	resolver           *metadataResolver
	labelsAvailable    bool // true when Labels API is configured and working
}

// newScanner creates a new scanner instance
func newScanner(
	driveClient DriveClient,
	dirClient DirectoryClient,
	cfg *config.Config,
	log logger.Logger,
) *scanner {
	// Create metadata resolver for drive name lookups
	resolver := newMetadataResolver(driveClient, log)

	return &scanner{
		driveClient: driveClient,
		dirClient:   dirClient,
		config:      cfg,
		logger:      log,
		workerCount: 50, // Default worker count (I/O-bound: Permissions.List + GetFile API calls)
		resolver:    resolver,
	}
}

// setWorkerCount sets the number of concurrent workers
func (s *scanner) setWorkerCount(count int) {
	if count > 0 {
		s.workerCount = count
	}
}

// setProgressFunc sets a callback function for progress updates
func (s *scanner) setProgressFunc(fn func(filesScanned, issuesFound int)) {
	s.progressFunc = fn
}

// setTargetProgressFunc sets a callback function for target-level progress updates
func (s *scanner) setTargetProgressFunc(fn func(targetIdx, targetTotal int, targetName string)) {
	s.targetProgressFunc = fn
}

// setStatusFunc sets a callback for plain status messages (no log prefix).
func (s *scanner) setStatusFunc(fn func(msg string)) {
	s.statusFunc = fn
}

// setLabelsAvailable marks that the Labels API is configured and label data
// will be present on files. When false, Label field shows "Cannot be retrieved".
func (s *scanner) setLabelsAvailable(available bool) {
	s.labelsAvailable = available
}

// matchesSharedWithPattern checks if an email matches the shared-with filter pattern.
// Supports wildcard patterns:
//   - *@example.com matches all emails from example.com domain
//   - user*@example.com matches emails starting with "user" from example.com
//   - *user@example.com matches emails ending with "user" from example.com
//   - exact@example.com matches exact email address
func matchesSharedWithPattern(email, pattern string) bool {
	// Exact match
	if email == pattern {
		return true
	}

	// Check if pattern contains wildcards
	if strings.Contains(pattern, "*") {
		matched, err := filepath.Match(pattern, email)
		if err == nil && matched {
			return true
		}
	}

	return false
}

// classifyAccessError returns a short human-readable reason for known access
// errors, or "" if the error is unknown and should be logged in full.
func classifyAccessError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "teamDriveMembershipRequired"):
		return "Access denied (not a member)"
	case strings.Contains(msg, "insufficientPermissions") || strings.Contains(msg, "insufficientFilePermissions"):
		return "Access denied (insufficient permissions)"
	case strings.Contains(msg, "notFound") || strings.Contains(msg, "404"):
		return "Not found (deleted or inaccessible)"
	case strings.Contains(msg, "forbidden") || strings.Contains(msg, "403"):
		return "Access denied"
	default:
		return ""
	}
}

// isPublicPermission checks if a permission is public ("anyone with link")
func isPublicPermission(perm Permission) bool {
	return strings.ToLower(perm.Type) == "anyone"
}

// scan performs a security scan based on the specified scope
func (s *scanner) scan(ctx context.Context, scope string, opts scanOptions) (*models.ScanResult, error) {
	// Normalize filter values once, before any concurrent access
	opts.FilterSharedWith = strings.ToLower(strings.TrimSpace(opts.FilterSharedWith))
	startTime := time.Now()
	result := models.NewScanResult(scope)

	// Resolve scope to targets
	s.logger.Info("Starting scan: scope=%s, workers=%d", scope, s.workerCount)
	targets, err := ResolveScope(scope, s.dirClient, s.driveClient, s.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve scope: %w", err)
	}
	s.logger.Info("Scope resolved: %d targets found", len(targets))

	// Filter targets by included_drives and excluded_drives
	filteredTargets := make([]Target, 0, len(targets))
	excludedCount := 0
	outOfScopeCount := 0
	for _, target := range targets {
		if !ShouldIncludeDrive(target, s.config.IncludedDrives) {
			s.logger.Debug("Skipping drive (not in included_drives): %s (ID: %s)", target.Name, target.ID)
			outOfScopeCount++
			continue
		}
		if ShouldExcludeDrive(target, s.config.ExcludedDrives) {
			s.logger.Debug("Excluding drive: %s (ID: %s)", target.Name, target.ID)
			excludedCount++
			continue
		}
		filteredTargets = append(filteredTargets, target)
	}

	skippedTotal := excludedCount + outOfScopeCount
	if skippedTotal > 0 {
		s.logger.Info("Found %d shared drives (%d excluded, %d to scan)", len(targets), skippedTotal, len(filteredTargets))
	} else {
		s.logger.Info("Found %d shared drives to scan", len(filteredTargets))
	}

	// Pre-populate drive name cache with known drive names from targets
	for _, target := range filteredTargets {
		if target.Type == "drive" && target.ID != "" && target.Name != "" {
			s.resolver.preloadDriveName(target.ID, target.Name)
		}
	}

	// Scan each target
	totalFileCount := 0
	for idx, target := range filteredTargets {
		targetName := target.Name
		if targetName == "" {
			targetName = target.Email
		}
		if targetName == "" {
			targetName = target.ID
		}

		s.logger.Info("Scanning target %d/%d: %s (type=%s)", idx+1, len(filteredTargets), targetName, target.Type)

		// Notify target progress callback (for progress reporter)
		if s.targetProgressFunc != nil {
			s.targetProgressFunc(idx+1, len(filteredTargets), targetName)
		}

		// Track files and issues before scanning this target
		filesBeforeScan := totalFileCount
		issuesBeforeScan := len(result.Issues)

		if err := s.scanTarget(ctx, target, result, &totalFileCount, opts); err != nil {
			if reason := classifyAccessError(err); reason != "" {
				if s.statusFunc != nil {
					s.statusFunc(fmt.Sprintf("Drive: %s — %s", targetName, reason))
				}
			} else {
				s.logger.Warn("Failed to scan target %s: %v", targetName, err)
			}
			result.AddError(fmt.Sprintf("target %s: %v", targetName, err))
		}

		// Calculate files and issues for this target
		filesThisTarget := totalFileCount - filesBeforeScan
		issuesThisTarget := len(result.Issues) - issuesBeforeScan

		s.logger.Info("Completed target %d/%d: %s — %d files, %d issues",
			idx+1, len(filteredTargets), targetName, filesThisTarget, issuesThisTarget)
	}

	// Update metadata with final counts
	result.SetFilesScanned(totalFileCount)
	result.Metadata.IssuesFound = len(result.Issues)

	// Set duration
	result.SetDuration(time.Since(startTime))

	s.logger.Info("Scan complete: %d files scanned, %d issues found in %s",
		result.Metadata.TotalFilesScanned, result.Metadata.IssuesFound, time.Since(startTime).Round(time.Millisecond))
	return result, nil
}

// scanTarget scans files for a specific target (user or drive).
// Uses pipeline parallelism: prefetches the next page from the API while
// processing the current page, so network I/O overlaps with CPU work.
func (s *scanner) scanTarget(ctx context.Context, target Target, result *models.ScanResult, totalFileCount *int, opts scanOptions) error {
	// Build query and driveID based on target type
	query := "trashed = false"
	var driveID string

	if target.Type == "drive" {
		driveID = target.ID
		s.logger.Info("Scanning shared drive: %s (ID: %s)", target.Name, target.ID)
	} else {
		driveID = ""
		s.logger.Info("Scanning user: %s", target.Email)
	}

	// Fetch first page
	fileList, err := s.driveClient.ListFiles(ctx, query, "", driveID)
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}

	type prefetchResult struct {
		list *FileList
		err  error
	}

	pageCount := 0
	for fileList != nil && len(fileList.Files) > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		pageCount++
		s.logger.Info("Page %d: processing %d files (%d total so far)", pageCount, len(fileList.Files), *totalFileCount)

		// Start prefetching next page while we process the current one
		var nextPageCh <-chan prefetchResult
		if fileList.NextPageToken != "" {
			ch := make(chan prefetchResult, 1)
			nextPageCh = ch
			go func(token string) {
				list, fetchErr := s.driveClient.ListFiles(ctx, query, token, driveID)
				ch <- prefetchResult{list, fetchErr}
			}(fileList.NextPageToken)
		}

		// Process current page concurrently (overlaps with next-page fetch)
		issues := s.processFiles(ctx, fileList.Files, opts)

		// Add issues to result
		for _, issue := range issues {
			result.AddIssue(issue)
		}

		// Update total file count
		*totalFileCount += len(fileList.Files)

		// Report progress
		if s.progressFunc != nil {
			s.progressFunc(*totalFileCount, len(result.Issues))
		}

		// Wait for prefetched next page
		if nextPageCh == nil {
			break
		}
		next := <-nextPageCh
		if next.err != nil {
			return fmt.Errorf("failed to list files: %w", next.err)
		}
		fileList = next.list
	}

	if pageCount == 0 {
		s.logger.Debug("No files found in this target")
	}

	return nil
}

// processFiles processes files concurrently using a worker pool.
// When permissions are fetched inline (no per-file API calls needed),
// workers handle CPU-bound filtering + occasional metadata resolution.
func (s *scanner) processFiles(ctx context.Context, files []File, opts scanOptions) []models.FileIssue {
	resultChan := make(chan models.FileIssue, len(files))
	var wg sync.WaitGroup

	// Use a semaphore channel to limit concurrency
	sem := make(chan struct{}, s.workerCount)

	for _, file := range files {
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		sem <- struct{}{} // acquire
		go func(f File) {
			defer wg.Done()
			defer func() { <-sem }() // release
			if issue := s.processFile(ctx, f, opts); issue != nil {
				resultChan <- *issue
			}
		}(file)
	}

	// Close resultChan after all goroutines finish
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	issues := make([]models.FileIssue, 0)
	for issue := range resultChan {
		issues = append(issues, issue)
	}

	return issues
}

// processFile processes a single file and returns an issue if security problems are found
func (s *scanner) processFile(ctx context.Context, file File, opts scanOptions) *models.FileIssue {
	// Use inline permissions from Files.List when available (avoids per-file API call)
	permissions := file.Permissions
	if len(permissions) == 0 {
		// Fallback: fetch permissions separately (e.g. if inline permissions were empty)
		var err error
		permissions, err = s.driveClient.ListPermissions(ctx, file.ID)
		if err != nil {
			s.logger.Debug("Failed to list permissions for file %s: %v", file.ID, err)
			return nil
		}
	}

	// Convert to model permissions and calculate risk
	modelPerms := make([]models.Permission, 0)
	hasExternal := false
	hasPublic := false

	for _, perm := range permissions {
		isPublic := isPublicPermission(perm)

		// External means: NOT in internal domains (regardless of allowlist)
		// Domain-based permissions should also be checked
		isExternal := false
		if perm.EmailAddress != "" {
			isExternal = !filter.IsInternalDomain(perm.EmailAddress, s.config.InternalDomains)
		} else if perm.Domain != "" {
			// For domain-based permissions, check if the domain is internal
			isExternal = !filter.IsInternalDomain("user@"+perm.Domain, s.config.InternalDomains)
		} else if perm.Type == "anyone" {
			// "Anyone" permissions are always external (and public)
			isExternal = true
		}

		if isPublic {
			hasPublic = true
		}
		if isExternal {
			hasExternal = true
		}

		riskLevel := filter.CalculateRiskLevel(models.Permission{
			ID:     perm.ID,
			Type:   perm.Type,
			Email:  perm.EmailAddress,
			Domain: perm.Domain,
			Role:   perm.Role,
		}, isPublic, isExternal)

		modelPerms = append(modelPerms, models.Permission{
			ID:         perm.ID,
			Type:       perm.Type,
			Email:      perm.EmailAddress,
			Domain:     perm.Domain,
			Role:       perm.Role,
			RiskLevel:  riskLevel,
			IsInternal: !isExternal, // Mark as internal if not external
		})
	}

	// If no external or public permissions, skip this file
	if !hasExternal && !hasPublic {
		return nil
	}

	// Apply --public filter: only show files with public "anyone" links
	if opts.FilterPublicOnly && !hasPublic {
		return nil
	}

	// Apply --shared-with filter: only show files shared with specific email or pattern
	if opts.FilterSharedWith != "" {
		matchesFilter := false
		for _, perm := range permissions {
			emailLower := strings.ToLower(perm.EmailAddress)
			if matchesSharedWithPattern(emailLower, opts.FilterSharedWith) {
				matchesFilter = true
				break
			}
		}
		if !matchesFilter {
			return nil
		}
	}

	// Determine label value
	var label string
	if !s.labelsAvailable {
		label = "Cannot be retrieved"
	} else if len(file.Labels) > 0 {
		label = strings.Join(file.Labels, "; ")
	} else {
		label = "Unlabeled"
	}

	// Create file issue (with permissions, but without expensive metadata yet)
	issue := models.FileIssue{
		FileID:      file.ID,
		FileName:    file.Name,
		WebViewLink: file.WebViewLink,
		DriveID:     file.DriveID,
		Label:       label,
		Permissions: modelPerms,
	}

	// Get owner info
	if len(file.Owners) > 0 {
		issue.OwnerEmail = file.Owners[0].EmailAddress
		issue.OwnerName = file.Owners[0].DisplayName
	}

	// Apply permission-based filters BEFORE expensive metadata resolution (getDriveName, getFolderPath).
	// These filters only need issue.Permissions which is already populated.
	if filter.ShouldExcludeInternalOnly(issue, s.config.InternalDomains) {
		return nil
	}
	if filter.ShouldExcludeTrustedOnly(issue, s.config.TrustedDomains) {
		return nil
	}
	if len(opts.FilterRiskLevels) > 0 {
		hasMatchingRisk := false
		for _, perm := range issue.Permissions {
			if perm.IsInternal {
				continue
			}
			for _, level := range opts.FilterRiskLevels {
				if strings.EqualFold(string(perm.RiskLevel), level) {
					hasMatchingRisk = true
					break
				}
			}
			if hasMatchingRisk {
				break
			}
		}
		if !hasMatchingRisk {
			return nil
		}
	}

	// Only resolve expensive metadata for files that passed all filters
	if file.DriveID != "" {
		issue.DriveName = s.resolver.getDriveName(ctx, &file)
	}
	issue.FolderPath = s.resolver.getFolderPath(ctx, &file)

	s.logger.Debug("Issue found: file=%s (%s), public=%v", file.Name, file.ID, hasPublic)

	return &issue
}
