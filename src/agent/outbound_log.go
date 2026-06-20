package agent

import (
	"context"

	"github.com/hornosg/wa-agent-runtime/src/logging"
)

// LogOutbound — adaptador temporal del puerto Outbound: loguea la respuesta.
// El envío real a Kapso lo hará el outbox del messaging-gateway (E03 pendiente).
type LogOutbound struct {
	log *logging.Logger
}

func NewLogOutbound(log *logging.Logger) *LogOutbound { return &LogOutbound{log: log} }

func (o *LogOutbound) Send(_ context.Context, to string, r Reply) error {
	o.log.Info("outbound.reply", map[string]any{
		"to": to, "handoff": r.Handoff, "text": r.Text,
		"note": "log-driver (outbound real = outbox del gateway, E03)",
	})
	return nil
}
