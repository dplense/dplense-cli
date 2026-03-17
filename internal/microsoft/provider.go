package microsoft

import (
	"context"
	"fmt"

	"github.com/dplense/dplense-cli/internal/logger"
	"github.com/dplense/dplense-cli/pkg/config"
	"github.com/dplense/dplense-cli/pkg/models"
	"github.com/dplense/dplense-cli/pkg/provider"
)

func init() {
	provider.Register(provider.Microsoft, NewProvider)
}

type microsoftProvider struct {
	cfg *config.Config
	log logger.Logger
	gc  *graphClient // lazily initialized
}

// NewProvider creates a new Microsoft 365 provider.
func NewProvider(cfg *config.Config, log logger.Logger) (provider.Provider, error) {
	return &microsoftProvider{cfg: cfg, log: log}, nil
}

func (m *microsoftProvider) Name() provider.Name { return provider.Microsoft }

func (m *microsoftProvider) ValidateConfig() error {
	mc := m.cfg.Microsoft
	if mc.TenantID == "" {
		return fmt.Errorf("microsoft.tenant_id is required")
	}
	if mc.ClientID == "" {
		return fmt.Errorf("microsoft.client_id is required")
	}
	if mc.ClientSecret == "" {
		return fmt.Errorf("microsoft.client_secret is required")
	}
	return nil
}

func (m *microsoftProvider) Scopes() []string {
	return []string{"sharepoint", "onedrive", "all", "site:<url>", "user:<email>"}
}

func (m *microsoftProvider) getClient() (*graphClient, error) {
	if m.gc != nil {
		return m.gc, nil
	}
	mc := m.cfg.Microsoft
	svc, adapter, err := newGraphService(mc.TenantID, mc.ClientID, mc.ClientSecret)
	if err != nil {
		return nil, fmt.Errorf("microsoft auth: %w", err)
	}
	m.gc = newClient(svc, adapter, m.log)
	return m.gc, nil
}

func (m *microsoftProvider) Scan(ctx context.Context, opts provider.ScanOptions) (*models.ScanResult, error) {
	gc, err := m.getClient()
	if err != nil {
		return nil, err
	}
	s := newScanner(gc, m.cfg, m.log)
	return s.scan(ctx, opts)
}

func (m *microsoftProvider) RevokePermission(ctx context.Context, resourceID, permissionID string, dryRun bool) (string, error) {
	gc, err := m.getClient()
	if err != nil {
		return "", err
	}
	r := newRevoker(gc, m.log)
	return r.revokePermission(ctx, resourceID, permissionID, dryRun)
}

func (m *microsoftProvider) RevokeUserAccess(ctx context.Context, email string, scanResult *models.ScanResult, dryRun bool) (int, int, error) {
	gc, err := m.getClient()
	if err != nil {
		return 0, 0, err
	}
	r := newRevoker(gc, m.log)
	return r.revokeUserAccess(ctx, email, scanResult, dryRun)
}
