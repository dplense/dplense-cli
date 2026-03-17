package microsoft

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dplense/dplense-cli/internal/filter"
	"github.com/dplense/dplense-cli/internal/logger"
	"github.com/dplense/dplense-cli/pkg/config"
	"github.com/dplense/dplense-cli/pkg/models"
	"github.com/dplense/dplense-cli/pkg/provider"
)

type scanner struct {
	gc  *graphClient
	cfg *config.Config
	log logger.Logger
}

func newScanner(gc *graphClient, cfg *config.Config, log logger.Logger) *scanner {
	return &scanner{gc: gc, cfg: cfg, log: log}
}

func (s *scanner) scan(ctx context.Context, opts provider.ScanOptions) (*models.ScanResult, error) {
	startTime := time.Now()
	result := models.NewScanResult(opts.Scope)
	result.Metadata.Provider = string(provider.Microsoft)

	totalFiles := 0

	switch {
	case opts.Scope == "sharepoint":
		if err := s.scanSharePoint(ctx, opts, result, &totalFiles); err != nil {
			return nil, err
		}
	case opts.Scope == "onedrive":
		if err := s.scanOneDrive(ctx, opts, result, &totalFiles); err != nil {
			return nil, err
		}
	case opts.Scope == "all":
		if err := s.scanSharePoint(ctx, opts, result, &totalFiles); err != nil {
			s.log.Warn("SharePoint scan failed: %v", err)
			result.AddError(fmt.Sprintf("SharePoint: %v", err))
		}
		if err := s.scanOneDrive(ctx, opts, result, &totalFiles); err != nil {
			s.log.Warn("OneDrive scan failed: %v", err)
			result.AddError(fmt.Sprintf("OneDrive: %v", err))
		}
	case strings.HasPrefix(opts.Scope, "site:"):
		siteURL := strings.TrimPrefix(opts.Scope, "site:")
		if err := s.scanSite(ctx, siteURL, opts, result, &totalFiles); err != nil {
			return nil, err
		}
	case strings.HasPrefix(opts.Scope, "user:"):
		email := strings.TrimPrefix(opts.Scope, "user:")
		if err := s.scanUserOneDrive(ctx, email, opts, result, &totalFiles); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown scope: %s (valid: sharepoint, onedrive, all, site:<url>, user:<email>)", opts.Scope)
	}

	result.SetFilesScanned(totalFiles)
	result.SetDuration(time.Since(startTime))
	s.log.Info("Microsoft scan complete: %d files scanned, %d issues found in %s",
		totalFiles, len(result.Issues), time.Since(startTime).Round(time.Millisecond))
	return result, nil
}

func (s *scanner) scanSharePoint(ctx context.Context, opts provider.ScanOptions, result *models.ScanResult, totalFiles *int) error {
	s.log.Info("Scanning SharePoint sites...")
	sites, err := s.gc.listAllSites(ctx)
	if err != nil {
		return fmt.Errorf("list SharePoint sites: %w", err)
	}
	s.log.Info("Found %d SharePoint sites", len(sites))

	for idx, site := range sites {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		siteID := deref(site.GetId())
		siteName := deref(site.GetDisplayName())
		if siteName == "" {
			siteName = deref(site.GetName())
		}

		if opts.TargetProgressFunc != nil {
			opts.TargetProgressFunc(idx+1, len(sites), siteName)
		}

		s.log.Info("Scanning site %d/%d: %s", idx+1, len(sites), siteName)

		drives, err := s.gc.listSiteDrives(ctx, siteID)
		if err != nil {
			s.log.Warn("Failed to list drives for site %s: %v", siteName, err)
			result.AddError(fmt.Sprintf("site %s: %v", siteName, err))
			continue
		}

		for _, drv := range drives {
			driveID := deref(drv.GetId())
			driveName := deref(drv.GetName())

			s.scanDrive(ctx, driveID, driveName, siteName, opts, result, totalFiles)
		}
	}
	return nil
}

func (s *scanner) scanOneDrive(ctx context.Context, opts provider.ScanOptions, result *models.ScanResult, totalFiles *int) error {
	s.log.Info("Scanning OneDrive for all users...")
	users, err := s.gc.listUsers(ctx)
	if err != nil {
		return fmt.Errorf("list users: %w", err)
	}
	s.log.Info("Found %d users", len(users))

	for idx, user := range users {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		userID := deref(user.GetId())
		userName := deref(user.GetDisplayName())
		userEmail := deref(user.GetMail())
		if userEmail == "" {
			userEmail = deref(user.GetUserPrincipalName())
		}

		if opts.TargetProgressFunc != nil {
			opts.TargetProgressFunc(idx+1, len(users), userName)
		}

		s.log.Info("Scanning OneDrive %d/%d: %s (%s)", idx+1, len(users), userName, userEmail)

		drv, err := s.gc.listUserDrive(ctx, userID)
		if err != nil {
			s.log.Debug("No OneDrive for user %s: %v", userName, err)
			continue
		}

		driveID := deref(drv.GetId())
		s.scanDrive(ctx, driveID, "OneDrive", userName, opts, result, totalFiles)
	}
	return nil
}

func (s *scanner) scanSite(ctx context.Context, siteURL string, opts provider.ScanOptions, result *models.ScanResult, totalFiles *int) error {
	s.log.Info("Scanning site: %s", siteURL)
	// For a specific site URL, we search for it
	sites, err := s.gc.listAllSites(ctx)
	if err != nil {
		return err
	}

	for _, site := range sites {
		webURL := deref(site.GetWebUrl())
		if strings.Contains(strings.ToLower(webURL), strings.ToLower(siteURL)) {
			siteID := deref(site.GetId())
			siteName := deref(site.GetDisplayName())

			drives, err := s.gc.listSiteDrives(ctx, siteID)
			if err != nil {
				return fmt.Errorf("list drives for site %s: %w", siteName, err)
			}

			for _, drv := range drives {
				driveID := deref(drv.GetId())
				driveName := deref(drv.GetName())
				s.scanDrive(ctx, driveID, driveName, siteName, opts, result, totalFiles)
			}
			return nil
		}
	}

	return fmt.Errorf("site not found: %s", siteURL)
}

func (s *scanner) scanUserOneDrive(ctx context.Context, email string, opts provider.ScanOptions, result *models.ScanResult, totalFiles *int) error {
	s.log.Info("Scanning OneDrive for user: %s", email)

	// List users to find the one matching the email
	users, err := s.gc.listUsers(ctx)
	if err != nil {
		return err
	}

	for _, user := range users {
		userEmail := deref(user.GetMail())
		if userEmail == "" {
			userEmail = deref(user.GetUserPrincipalName())
		}
		if strings.EqualFold(userEmail, email) {
			userID := deref(user.GetId())
			drv, err := s.gc.listUserDrive(ctx, userID)
			if err != nil {
				return fmt.Errorf("get OneDrive for %s: %w", email, err)
			}
			driveID := deref(drv.GetId())
			s.scanDrive(ctx, driveID, "OneDrive", email, opts, result, totalFiles)
			return nil
		}
	}

	return fmt.Errorf("user not found: %s", email)
}

func (s *scanner) scanDrive(ctx context.Context, driveID, driveName, ownerName string, opts provider.ScanOptions, result *models.ScanResult, totalFiles *int) {
	items, err := s.gc.listDriveRootChildren(ctx, driveID)
	if err != nil {
		s.log.Warn("Failed to list items in drive %s: %v", driveName, err)
		result.AddError(fmt.Sprintf("drive %s (%s): %v", driveName, ownerName, err))
		return
	}

	s.log.Info("Found %d items in drive %s (%s)", len(items), driveName, ownerName)
	*totalFiles += len(items)

	for _, item := range items {
		if ctx.Err() != nil {
			return
		}

		perms, err := s.gc.listItemPermissions(ctx, driveID, item.ID)
		if err != nil {
			s.log.Debug("Failed to get permissions for %s: %v", item.Name, err)
			continue
		}

		// Convert to model permissions
		var modelPerms []models.Permission
		hasExternal := false

		for _, gp := range perms {
			if isInherited(gp) {
				continue // Skip inherited permissions
			}

			mp := mapPermission(gp, s.cfg.InternalDomains)

			if !mp.IsInternal {
				hasExternal = true
			}
			modelPerms = append(modelPerms, mp)
		}

		if !hasExternal {
			continue
		}

		issue := models.FileIssue{
			FileID:      item.ID,
			FileName:    item.Name,
			DriveName:   driveName,
			DriveID:     driveID,
			OwnerEmail:  ownerName,
			WebViewLink: item.WebURL,
			Permissions: modelPerms,
		}

		// Apply filters
		if !provider.MatchesFilters(issue, opts) {
			continue
		}
		if filter.ShouldExcludeInternalOnly(issue, s.cfg.InternalDomains) {
			continue
		}
		if filter.ShouldExcludeTrustedOnly(issue, s.cfg.TrustedDomains) {
			continue
		}

		result.AddIssue(issue)

		if opts.ProgressFunc != nil {
			opts.ProgressFunc(*totalFiles, len(result.Issues))
		}
	}
}
