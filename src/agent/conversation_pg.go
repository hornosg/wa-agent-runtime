package agent

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgConversations — adaptador de ConversationStore sobre core.conversation.
type PgConversations struct {
	pool *pgxpool.Pool
}

func NewPgConversations(pool *pgxpool.Pool) *PgConversations { return &PgConversations{pool: pool} }

func (p *PgConversations) Get(ctx context.Context, tenantSlug, contact string) (Conversation, error) {
	var state string
	var ctxJSON []byte
	err := p.pool.QueryRow(ctx,
		`SELECT state, context FROM core.conversation WHERE tenant_slug=$1 AND contact=$2`,
		tenantSlug, contact,
	).Scan(&state, &ctxJSON)
	if errors.Is(err, pgx.ErrNoRows) {
		return Conversation{TenantSlug: tenantSlug, Contact: contact, State: StateIdle}, nil
	}
	if err != nil {
		return Conversation{}, err
	}
	c := Conversation{TenantSlug: tenantSlug, Contact: contact, State: ConvState(state)}
	if c.State == StateAwaitingBookingConfirm && len(ctxJSON) > 0 {
		var bc BookingCtx
		if json.Unmarshal(ctxJSON, &bc) == nil {
			c.Booking = &bc
		}
	}
	return c, nil
}

func (p *PgConversations) Save(ctx context.Context, c Conversation) error {
	ctxJSON := []byte("{}")
	if c.Booking != nil {
		if b, err := json.Marshal(c.Booking); err == nil {
			ctxJSON = b
		}
	}
	_, err := p.pool.Exec(ctx, `
		INSERT INTO core.conversation (tenant_slug, contact, state, context, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (tenant_slug, contact)
		DO UPDATE SET state=EXCLUDED.state, context=EXCLUDED.context, updated_at=now()`,
		c.TenantSlug, c.Contact, string(c.State), ctxJSON)
	return err
}
