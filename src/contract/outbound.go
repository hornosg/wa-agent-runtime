package contract

// OutboundMessageArgs — respuesta saliente que el runtime encola (cola "outbound")
// y el messaging-gateway consume para enviar a Kapso. DEBE coincidir con el del
// gateway (Kind + JSON). Fuente de verdad: ADR-0003.
type OutboundMessageArgs struct {
	TenantSlug string `json:"tenant_slug"`
	To         string `json:"to"`
	Text       string `json:"text"`
	Handoff    bool   `json:"handoff"`
}

func (OutboundMessageArgs) Kind() string { return "outbound_message" }

const (
	QueueInbound  = "inbound"
	QueueOutbound = "outbound"
)
