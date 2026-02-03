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

// Scanner performs security audits on Google Drive files
type Scanner struct {
	driveClient  gdrive.DriveClient
	dirClient    gdrive.DirectoryClient
	config       *config.Config
	logger       logger.Logger
	workerCount  int
	progressFunc func(filesScanned, issuesFound int)
	resolver     *drive.MetadataResolver
	// Filters
	filterSharedWith string // Filter: only show files shared with this email
	filterPublicOnly bool   // Filter: only show files with public "anyone" links
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

// SetFilterSharedWith sets a filter to only show files shared with a specific email or pattern
// Supports wildcard patterns:
//   - *@example.com matches all emails from example.com domain
//   - user*@example.com matches emails starting with "user" from example.com
//   - *user@example.com matches emails ending with "user" from example.com
//   - exact@example.com matches exact email address
func (s *Scanner) SetFilterSharedWith(email string) {
	s.filterSharedWith = strings.ToLower(strings.TrimSpace(email))
}

// matchesSharedWithPattern checks if an email matches the shared-with filter pattern
func (s *Scanner) matchesSharedWithPattern(email string) bool {
	pattern := s.filterSharedWith
	
	// Exact match
	if email == pattern {
		return true
	}
	
	// Check if pattern contains wildcards
	if strings.Contains(pattern, "*") {
		// Convert wildcard pattern to glob pattern
		// Replace * with * for glob matching
		matched, err := filepath.Match(pattern, email)
		if err == nil && matched {
			return true
		}
	}
	
	return false
}

// SetFilterPublicOnly sets a filter to only show files with public "anyone" links
func (s *Scanner) SetFilterPublicOnly(publicOnly bool) {
	s.filterPublicOnly = publicOnly
}

// IsPublicPermission checks if a permission is public ("anyone with link")
func IsPublicPermission(perm gdrive.Permission) bool {
	return strings.ToLower(perm.Type) == "anyone"
}

// Scan performs a security scan based on the specified scope
func (s *Scanner) Scan(ctx context.Context, scope string) (*models.ScanResult, error) {
	startTime := time.Now()
	result := models.NewScanResult(scope)

	// Resolve scope to targets
	targets, err := ResolveScope(scope, s.dirClient, s.driveClient, s.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve scope: %w", err)
	}

	// Filter excluded drives
	filteredTargets := make([]Target, 0, len(targets))
	excludedCount := 0
	for _, target := range targets {
		if target.Type == "drive" && s.shouldExcludeDrive(target) {
			s.logger.Debug("Excluding drive: %s (ID: %s)", target.Name, target.ID)
			excludedCount++
			continue
		}
		filteredTargets = append(filteredTargets, target)
	}

	if excludedCount > 0 {
		s.logger.Info("Excluded %d drive(s) based on configuration", excludedCount)
	}

	s.logger.Info("Starting scan with scope '%s', found %d targets", scope, len(filteredTargets))
	if len(filteredTargets) > 0 {
		s.logger.Info("Scanning %d targets... This may take a while", len(filteredTargets))
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

		// Check if drive should be included/excluded
		if !s.shouldIncludeDrive(target) {
			s.logger.Debug("Skipping drive (not in included_drives): %s", targetName)
			continue
		}
		if s.shouldExcludeDrive(target) {
			s.logger.Debug("Skipping drive (in excluded_drives): %s", targetName)
			continue
		}

		s.logger.Info("Scanning target %d/%d: %s", idx+1, len(targets), targetName)

		// Track files and issues before scanning this target
		filesBeforeScan := totalFileCount
		issuesBeforeScan := len(result.Issues)

		if err := s.scanTarget(ctx, target, result, &totalFileCount); err != nil {
			s.logger.Warn("Failed to scan target %s: %v", targetName, err)
			// Continue with other targets
		}

		// Calculate files and issues for this target
		filesThisTarget := totalFileCount - filesBeforeScan
		issuesThisTarget := len(result.Issues) - issuesBeforeScan

		s.logger.Info("Completed target %d/%d (Files: %d, Issues: %d)",
			idx+1, len(filteredTargets), filesThisTarget, issuesThisTarget)
	}

	// Update metadata with final counts
	result.SetFilesScanned(totalFileCount)
	result.Metadata.IssuesFound = len(result.Issues)

	// Set duration
	result.SetDuration(time.Since(startTime))

	s.logger.Info("Scan complete: %d files scanned, %d issues found", result.Metadata.TotalFilesScanned, result.Metadata.IssuesFound)
	return result, nil
}

// scanTarget scans files for a specific target (user or drive)
func (s *Scanner) scanTarget(ctx context.Context, target Target, result *models.ScanResult, totalFileCount *int) error {
	// Build query and driveID based on target type
	query := "trashed = false"
	var driveID string

	if target.Type == "drive" {
		// For shared drives, pass the driveId to limit results to that drive
		driveID = target.ID
		s.logger.Debug("Scanning shared drive %s (ID: %s)", target.Name, target.ID)
	} else {
		// For user scopes, scan their files (no driveId limit)
		driveID = ""
		s.logger.Debug("Scanning user %s", target.Email)
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

		s.logger.Debug("Page %d: Processing %d files", pageCount, len(fileList.Files))

		// Log progress for every page
		if pageCount%10 == 0 || pageCount == 1 {
			s.logger.Info("Processing page %d... (%d files so far)", pageCount, *totalFileCount)
		}

		// Process files concurrently
		issues := s.processFiles(ctx, fileList.Files)

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
func (s *Scanner) processFiles(ctx context.Context, files []gdrive.File) []models.FileIssue {
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
					if issue := s.processFile(ctx, item.file); issue != nil {
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
func (s *Scanner) processFile(ctx context.Context, file gdrive.File) *models.FileIssue {
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

	// Apply --public filter: only show files with public "anyone" links
	if s.filterPublicOnly && !hasPublic {
		return nil
	}

	// Apply --shared-with filter: only show files shared with specific email or pattern
	if s.filterSharedWith != "" {
		matchesFilter := false
		for _, perm := range permissions {
			emailLower := strings.ToLower(perm.EmailAddress)
			if s.matchesSharedWithPattern(emailLower) {
				matchesFilter = true
				break
			}
		}
		if !matchesFilter {
			return nil
		}
	}

	// Create file issue
	issue := models.FileIssue{
		FileID:      file.ID,
		FileName:    file.Name,
		WebViewLink: file.WebViewLink,
		DriveID:     file.DriveID,
		Permissions: modelPerms,
	}

	// Get owner info
	if len(file.Owners) > 0 {
		issue.OwnerEmail = file.Owners[0].EmailAddress
		issue.OwnerName = file.Owners[0].DisplayName
	}

	// Resolve drive name from DriveID
	if file.DriveID != "" {
		issue.DriveName = s.resolver.GetDriveName(&file)
	}

	// Resolve folder path
	issue.FolderPath = s.resolver.GetFolderPath(&file)

	// Apply filters
	if filter.ShouldExcludeInternalOnly(issue, s.config.InternalDomains) {
		return nil
	}
	if filter.ShouldExcludeTrustedOnly(issue, s.config.TrustedDomains) {
		return nil
	}

	return &issue
}

// shouldIncludeDrive checks if a drive should be included based on configuration
// If included_drives is empty, all drives are included by default
// If included_drives is specified, only those drives are included
func (s *Scanner) shouldIncludeDrive(target Target) bool {
	// If no whitelist specified, include all drives
	if len(s.config.IncludedDrives) == 0 {
		return true
	}

	// Only scan if target type is "drive" (skip for user scopes)
	if target.Type != "drive" {
		return true
	}

	// Check if drive is in the whitelist
	for _, included := range s.config.IncludedDrives {
		// Match by ID or name
		if target.ID == included || target.Name == included {
			return true
		}
	}

	return false
}

// shouldExcludeDrive checks if a drive should be excluded based on configuration
func (s *Scanner) shouldExcludeDrive(target Target) bool {
	if len(s.config.ExcludedDrives) == 0 {
		return false
	}

	// Only apply to drive targets
	if target.Type != "drive" {
		return false
	}

	for _, excluded := range s.config.ExcludedDrives {
		// Match by ID or name
		if target.ID == excluded || target.Name == excluded {
			return true
		}
	}

	return false
}
