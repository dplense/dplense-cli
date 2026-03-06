package audit

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gdrive-audit/internal/drive"
	"gdrive-audit/internal/filter"
	"gdrive-audit/internal/logger"
	"gdrive-audit/pkg/config"
	"gdrive-audit/pkg/gdrive"
	"gdrive-audit/pkg/models"
)

// ScanOptions holds immutable options for a single scan invocation.
// Passed by value to avoid race conditions with concurrent worker goroutines.
type ScanOptions struct {
	FilterSharedWith string   // Only show files shared with this email/pattern
	FilterPublicOnly bool     // Only show files with public "anyone" links
	FilterRiskLevels []string // Only show issues at these risk levels (empty = all)
}

// Scanner performs security audits on Google Drive files
type Scanner struct {
	driveClient        gdrive.DriveClient
	dirClient          gdrive.DirectoryClient
	config             *config.Config
	logger             logger.Logger
	workerCount        int
	progressFunc       func(filesScanned, issuesFound int)
	targetProgressFunc func(targetIdx, targetTotal int, targetName string)
	resolver           *drive.MetadataResolver
	labelsAvailable    bool // true when Labels API is configured and working
}

// NewScanner creates a new scanner instance
func NewScanner(
	driveClient gdrive.DriveClient,
	dirClient gdrive.DirectoryClient,
	cfg *config.Config,
	log logger.Logger,
) *Scanner {
	// Create metadata resolver for drive name lookups
	resolver := drive.NewMetadataResolver(driveClient, log)

	return &Scanner{
		driveClient: driveClient,
		dirClient:   dirClient,
		config:      cfg,
		logger:      log,
		workerCount: 10, // Default worker count
		resolver:    resolver,
	}
}

// SetWorkerCount sets the number of concurrent workers
func (s *Scanner) SetWorkerCount(count int) {
	if count > 0 {
		s.workerCount = count
	}
}

// SetProgressFunc sets a callback function for progress updates
func (s *Scanner) SetProgressFunc(fn func(filesScanned, issuesFound int)) {
	s.progressFunc = fn
}

// SetTargetProgressFunc sets a callback function for target-level progress updates
func (s *Scanner) SetTargetProgressFunc(fn func(targetIdx, targetTotal int, targetName string)) {
	s.targetProgressFunc = fn
}

// SetLabelsAvailable marks that the Labels API is configured and label data
// will be present on files. When false, Label field shows "Cannot be retrieved".
func (s *Scanner) SetLabelsAvailable(available bool) {
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

// IsPublicPermission checks if a permission is public ("anyone with link")
func IsPublicPermission(perm gdrive.Permission) bool {
	return strings.ToLower(perm.Type) == "anyone"
}

// Scan performs a security scan based on the specified scope
func (s *Scanner) Scan(ctx context.Context, scope string, opts ScanOptions) (*models.ScanResult, error) {
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
			s.resolver.PreloadDriveName(target.ID, target.Name)
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
			s.logger.Warn("Failed to scan target %s: %v", targetName, err)
			// Continue with other targets
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

// scanTarget scans files for a specific target (user or drive)
func (s *Scanner) scanTarget(ctx context.Context, target Target, result *models.ScanResult, totalFileCount *int, opts ScanOptions) error {
	// Build query and driveID based on target type
	query := "trashed = false"
	var driveID string

	if target.Type == "drive" {
		// For shared drives, pass the driveId to limit results to that drive
		driveID = target.ID
		s.logger.Info("Scanning shared drive: %s (ID: %s)", target.Name, target.ID)
	} else {
		// For user scopes, scan their files (no driveId limit)
		driveID = ""
		s.logger.Info("Scanning user: %s", target.Email)
	}

	pageToken := ""
	pageCount := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		fileList, err := s.driveClient.ListFiles(ctx, query, pageToken, driveID)
		if err != nil {
			return fmt.Errorf("failed to list files: %w", err)
		}

		pageCount++
		if len(fileList.Files) == 0 {
			if pageCount == 1 {
				s.logger.Debug("No files found in this target")
			}
			break
		}

		s.logger.Info("Page %d: processing %d files (%d total so far)", pageCount, len(fileList.Files), *totalFileCount)

		// Process files concurrently
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

		if fileList.NextPageToken == "" {
			break
		}
		pageToken = fileList.NextPageToken
	}

	return nil
}

// processFiles processes files concurrently using a worker pool
func (s *Scanner) processFiles(ctx context.Context, files []gdrive.File, opts ScanOptions) []models.FileIssue {
	type workItem struct {
		file gdrive.File
	}

	workChan := make(chan workItem, len(files))
	resultChan := make(chan models.FileIssue, len(files))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < s.workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range workChan {
				select {
				case <-ctx.Done():
					return
				default:
					if issue := s.processFile(ctx, item.file, opts); issue != nil {
						resultChan <- *issue
					}
				}
			}
		}()
	}

	// Send work items
	go func() {
		defer close(workChan)
		for _, file := range files {
			select {
			case <-ctx.Done():
				return
			case workChan <- workItem{file: file}:
			}
		}
	}()

	// Collect results
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
func (s *Scanner) processFile(ctx context.Context, file gdrive.File, opts ScanOptions) *models.FileIssue {
	// Get permissions with timeout protection
	s.logger.Debug("Listing permissions for file: %s (%s)", file.ID, file.Name)
	permissions, err := s.driveClient.ListPermissions(ctx, file.ID)
	if err != nil {
		s.logger.Debug("Failed to list permissions for file %s: %v", file.ID, err)
		return nil
	}

	// Convert to model permissions and calculate risk
	modelPerms := make([]models.Permission, 0)
	hasExternal := false
	hasPublic := false

	for _, perm := range permissions {
		isPublic := IsPublicPermission(perm)

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

	// Count external permissions for logging
	extCount := 0
	for _, p := range modelPerms {
		if !p.IsInternal {
			extCount++
		}
	}
	s.logger.Info("Issue found: file=%s (%s), external_permissions=%d, public=%v", file.Name, file.ID, extCount, hasPublic)

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
		label = "Unclassified"
	}

	// Create file issue
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

	// Resolve drive name from DriveID
	if file.DriveID != "" {
		issue.DriveName = s.resolver.GetDriveName(ctx, &file)
	}

	// Resolve folder path
	issue.FolderPath = s.resolver.GetFolderPath(ctx, &file)

	// Apply filters
	if filter.ShouldExcludeInternalOnly(issue, s.config.InternalDomains) {
		return nil
	}
	if filter.ShouldExcludeTrustedOnly(issue, s.config.TrustedDomains) {
		return nil
	}

	// Apply --risk-level filter: only include issues with matching risk levels
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

	return &issue
}

