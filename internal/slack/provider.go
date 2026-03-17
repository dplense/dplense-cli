package slack

import (
	"context"
	"fmt"

	"github.com/dplense/dplense-cli/internal/logger"
	"github.com/dplense/dplense-cli/pkg/config"
	"github.com/dplense/dplense-cli/pkg/models"
	"github.com/dplense/dplense-cli/pkg/provider"
)

func init() {
	provider.Register(provider.Slack, NewProvider)
}

type slackProvider struct {
	cfg *config.Config
	log logger.Logger
	sc  *slackClient // lazily initialized
}

// NewProvider creates a new Slack provider.
func NewProvider(cfg *config.Config, log logger.Logger) (provider.Provider, error) {
	return &slackProvider{cfg: cfg, log: log}, nil
}

func (p *slackProvider) Name() provider.Name { return provider.Slack }

func (p *slackProvider) ValidateConfig() error {
	if p.cfg.Slack.BotToken == "" {
		return fmt.Errorf("slack.bot_token is required")
	}
	return nil
}

func (p *slackProvider) Scopes() []string {
	return []string{"files", "channels", "guests", "all"}
}

func (p *slackProvider) getClient() *slackClient {
	if p.sc != nil {
		return p.sc
	}
	p.sc = newSlackClient(p.cfg.Slack.BotToken, p.log)
	return p.sc
}

func (p *slackProvider) Scan(ctx context.Context, opts provider.ScanOptions) (*models.ScanResult, error) {
	s := newScanner(p.getClient(), p.cfg, p.log)
	return s.scan(ctx, opts)
}

func (p *slackProvider) RevokePermission(ctx context.Context, resourceID, permissionID string, dryRun bool) (string, error) {
	r := newRevoker(p.getClient(), p.log)
	return r.revokePermission(ctx, resourceID, permissionID, dryRun)
}

func (p *slackProvider) RevokeUserAccess(ctx context.Context, email string, scanResult *models.ScanResult, dryRun bool) (int, int, error) {
	r := newRevoker(p.getClient(), p.log)
	return r.revokeUserAccess(ctx, email, scanResult, dryRun)
}
