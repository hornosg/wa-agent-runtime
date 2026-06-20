package agent

import (
	"context"
	"errors"

	"github.com/hornosg/wa-agent-runtime/src/logging"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgTenants — adaptador de TenantConfigProvider sobre core.tenants (Postgres).
type PgTenants struct {
	pool *pgxpool.Pool
	log  *logging.Logger
}

func NewPgTenants(pool *pgxpool.Pool, log *logging.Logger) *PgTenants {
	return &PgTenants{pool: pool, log: log}
}

func (p *PgTenants) Get(ctx context.Context, slug string) (TenantConfig, error) {
	var mode, status string
	err := p.pool.QueryRow(ctx,
		`SELECT mode, status FROM core.tenants WHERE slug = $1`, slug,
	).Scan(&mode, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		// Tenant desconocido: default seguro rag_chat/pending (no rompe el cable).
		p.log.Warn("tenant.not_found_defaulting", map[string]any{"tenant_slug": slug, "default_mode": "rag_chat"})
		return TenantConfig{Slug: slug, Mode: ModeRagChat, Status: "pending"}, nil
	}
	if err != nil {
		return TenantConfig{}, err
	}
	return TenantConfig{Slug: slug, Mode: TenantMode(mode), Status: status}, nil
}
