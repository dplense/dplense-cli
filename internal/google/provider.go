package google

import (
	"context"
	"fmt"
	"strings"

	"github.com/dplense/dplense-cli/internal/logger"
	"github.com/dplense/dplense-cli/pkg/config"
	"github.com/dplense/dplense-cli/pkg/models"
	"github.com/dplense/dplense-cli/pkg/provider"
)

func init() {
	provider.Register(provider.Google, NewProvider)
}

type googleProvider struct {
	cfg       *config.Config
	log       logger.Logger
	dc        DriveClient // lazily initialized
	legacyCfg *config.Config
}

// NewProvider creates a new Google Drive provider.
func NewProvider(cfg *config.Config, log logger.Logger) (provider.Provider, error) {
	return &googleProvider{cfg: cfg, log: log}, nil
}

func (g *googleProvider) Name() provider.Name { return provider.Google }

func (g *googleProvider) ValidateConfig() error {
	creds := g.resolveCredentialsPath()
	if creds == "" {
		return fmt.Errorf("google credentials_path is required (set in config or via --credentials flag)")
	}
	return nil
}

func (g *googleProvider) Scopes() []string {
	return []string{"active", "suspended", "shared-drives", "user:<email>"}
}

func (g *googleProvider) getDriveClient(ctx context.Context) (DriveClient, *config.Config, error) {
	lcfg := g.getLegacyConfig()
	if g.dc != nil {
		return g.dc, lcfg, nil
	}
	dc, err := NewDriveClient(ctx, lcfg, g.log)
	if err != nil {
		return nil, nil, fmt.Errorf("google drive client: %w", err)
	}
	g.dc = dc
	return dc, lcfg, nil
}

func (g *googleProvider) getLegacyConfig() *config.Config {
	if g.legacyCfg != nil {
		return g.legacyCfg
	}
	g.legacyCfg = g.buildLegacyConfig()
	return g.legacyCfg
}

func (g *googleProvider) Scan(ctx context.Context, opts provider.ScanOptions) (*models.ScanResult, error) {
	dc, legacyCfg, err := g.getDriveClient(ctx)
	if err != nil {
		return nil, err
	}

	// Initialize Directory client (for user enumeration)
	var dirClient DirectoryClient
	scope := opts.Scope
	if scope == "active" || scope == "suspended" || strings.HasPrefix(scope, "user:") {
		dirClient, err = newDirectoryClient(ctx, legacyCfg, g.log)
		if err != nil {
			g.log.Warn("Directory service not available: %v. User enumeration scopes may not work.", err)
		}
	}

	// Initialize label resolver (optional)
	labelsAvailable := false
	lr, err := newLabelResolver(ctx, legacyCfg, g.log)
	if err != nil {
		g.log.Info("Labels API not available: %v", err)
	} else {
		labelsAvailable = true
		dc.SetIncludeLabels(lr.labelIDs())
		configureLabelResolver(dc, lr.resolve)
	}

	// Create scanner
	sc := newScanner(dc, dirClient, legacyCfg, g.log)
	sc.setLabelsAvailable(labelsAvailable)

	// Set up progress callbacks
	if opts.ProgressFunc != nil {
		sc.setProgressFunc(opts.ProgressFunc)
	}
	if opts.TargetProgressFunc != nil {
		sc.setTargetProgressFunc(opts.TargetProgressFunc)
	}
	if opts.StatusFunc != nil {
		sc.setStatusFunc(opts.StatusFunc)
	}

	// Map provider scan options to local scan options
	scanOpts := scanOptions{
		FilterSharedWith: opts.FilterSharedWith,
		FilterPublicOnly: opts.FilterPublicOnly,
		FilterRiskLevels: opts.FilterRiskLevels,
	}

	result, err := sc.scan(ctx, scope, scanOpts)
	if err != nil {
		return nil, err
	}

	result.Metadata.Provider = string(provider.Google)
	return result, nil
}

func (g *googleProvider) RevokePermission(ctx context.Context, resourceID, permissionID string, dryRun bool) (string, error) {
	dc, legacyCfg, err := g.getDriveClient(ctx)
	if err != nil {
		return "", err
	}

	fr := newFileRevoker(dc, legacyCfg, g.log, dryRun)

	if permissionID != "" {
		// Revoke specific user from file
		if err := fr.revokeUserFromFile(ctx, resourceID, permissionID); err != nil {
			return "", err
		}
		if dryRun {
			return fmt.Sprintf("[DRY-RUN] Would revoke %s from file %s", permissionID, resourceID), nil
		}
		return fmt.Sprintf("Revoked %s from file %s", permissionID, resourceID), nil
	}

	// Revoke all external permissions
	count, err := fr.revokeExternalPermissions(ctx, resourceID)
	if err != nil {
		return "", err
	}
	if dryRun {
		return fmt.Sprintf("[DRY-RUN] Would revoke %d external permissions from file %s", count, resourceID), nil
	}
	return fmt.Sprintf("Revoked %d external permissions from file %s", count, resourceID), nil
}

func (g *googleProvider) RevokeUserAccess(ctx context.Context, email string, scanResult *models.ScanResult, dryRun bool) (int, int, error) {
	dc, legacyCfg, err := g.getDriveClient(ctx)
	if err != nil {
		return 0, 0, err
	}

	ur := newUserRevoker(dc, legacyCfg, g.log, dryRun)
	return ur.revokeUserFromFiles(ctx, email, scanResult)
}

// buildLegacyConfig creates a Config that the existing Google-specific code
// expects, with credentials and drive filters resolved from both the nested
// google config and the deprecated top-level fields.
func (g *googleProvider) buildLegacyConfig() *config.Config {
	creds := g.cfg.Google.CredentialsPath
	if creds == "" {
		creds = g.cfg.CredentialsPath
	}
	impersonate := g.cfg.Google.ImpersonateUser
	if impersonate == "" {
		impersonate = g.cfg.ImpersonateUser
	}
	included := g.cfg.Google.IncludedDrives
	if len(included) == 0 {
		included = g.cfg.IncludedDrives
	}
	excluded := g.cfg.Google.ExcludedDrives
	if len(excluded) == 0 {
		excluded = g.cfg.ExcludedDrives
	}

	return &config.Config{
		InternalDomains: g.cfg.InternalDomains,
		TrustedDomains:  g.cfg.TrustedDomains,
		IncludedDrives:  included,
		ExcludedDrives:  excluded,
		DefaultScope:    g.cfg.DefaultScope,
		DryRun:          g.cfg.DryRun,
		CredentialsPath: creds,
		ImpersonateUser: impersonate,
		Logging:         g.cfg.Logging,
	}
}

func (g *googleProvider) resolveCredentialsPath() string {
	if g.cfg.Google.CredentialsPath != "" {
		return g.cfg.Google.CredentialsPath
	}
	return g.cfg.CredentialsPath
}
