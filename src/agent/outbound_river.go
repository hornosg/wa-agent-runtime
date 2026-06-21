package agent

import (
	"context"

	"github.com/hornosg/wa-agent-runtime/src/contract"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

// RiverOutbound — adaptador del puerto Outbound: encola la respuesta en la cola
// "outbound" de River; el messaging-gateway la consume y la envía a Kapso.
type RiverOutbound struct {
	client *river.Client[pgx.Tx]
}

func NewRiverOutbound(client *river.Client[pgx.Tx]) *RiverOutbound {
	return &RiverOutbound{client: client}
}

func (o *RiverOutbound) Send(ctx context.Context, tenantSlug, to string, r Reply) error {
	_, err := o.client.Insert(ctx, contract.OutboundMessageArgs{
		TenantSlug: tenantSlug,
		To:         to,
		Text:       r.Text,
		Handoff:    r.Handoff,
	}, &river.InsertOpts{Queue: contract.QueueOutbound})
	return err
}
