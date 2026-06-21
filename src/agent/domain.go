package agent

import (
	"context"
	"time"
)

// InboundMessage — mensaje entrante normalizado (espejo del contrato del job).
type InboundMessage struct {
	TenantSlug        string
	ProviderMessageID string
	From              string
	To                string
	Text              string
	ReceivedAt        time.Time
}

// Intent — clasificación L1 del mensaje.
type Intent string

const (
	IntentFAQ        Intent = "faq"
	IntentBooking    Intent = "booking"
	IntentCancel     Intent = "cancel"
	IntentReschedule Intent = "reschedule"
	IntentHandoff    Intent = "handoff"
	IntentOther      Intent = "other"
)

// TenantMode — G-10: agenda (FAQ + booking) | rag_chat (FAQ-only).
type TenantMode string

const (
	ModeAgenda  TenantMode = "agenda"
	ModeRagChat TenantMode = "rag_chat"
)

// TenantConfig — config mínima del tenant que condiciona la respuesta.
type TenantConfig struct {
	Slug   string
	Mode   TenantMode
	Status string // pending | active | suspended (RULE-06)
}

// Reply — respuesta del agente. Handoff = derivar a humano (P-05/G-09).
type Reply struct {
	Text    string
	Handoff bool
}

// ── Puertos de salida (ADR-0003) ─────────────────────────────────────────────

type TenantConfigProvider interface {
	Get(ctx context.Context, slug string) (TenantConfig, error)
}

type IntentClassifier interface {
	Classify(ctx context.Context, m InboundMessage, tc TenantConfig) (Intent, error)
}

type Replier interface {
	Reply(ctx context.Context, m InboundMessage, intent Intent, tc TenantConfig) (Reply, error)
}

type Outbound interface {
	Send(ctx context.Context, tenantSlug, to string, r Reply) error
}
