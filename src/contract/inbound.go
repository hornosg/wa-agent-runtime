// Package contract — contrato de integración con el messaging-gateway (SVC-01).
// DEBE coincidir con services/messaging-gateway/src/contract (Kind + tags JSON).
// Fuente de verdad: ADR-0003. No modificar unilateralmente.
package contract

import "time"

type InboundMessageArgs struct {
	TenantSlug        string    `json:"tenant_slug"`
	ProviderMessageID string    `json:"provider_message_id"`
	From              string    `json:"from"`
	To                string    `json:"to"`
	Text              string    `json:"text"`
	ReceivedAt        time.Time `json:"received_at"`
}

func (InboundMessageArgs) Kind() string { return "inbound_message" }
