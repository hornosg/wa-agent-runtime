package agent

import (
	"context"

	"github.com/hornosg/wa-agent-runtime/src/logging"
)

// AgentRuntime — orquestador (ADR-0003): tenant → clasificar → responder → outbound.
type AgentRuntime struct {
	tenants    TenantConfigProvider
	classifier IntentClassifier
	replier    Replier
	outbound   Outbound
	log        *logging.Logger
}

func New(t TenantConfigProvider, c IntentClassifier, r Replier, o Outbound, log *logging.Logger) *AgentRuntime {
	return &AgentRuntime{tenants: t, classifier: c, replier: r, outbound: o, log: log}
}

func (a *AgentRuntime) Process(ctx context.Context, m InboundMessage) error {
	tc, err := a.tenants.Get(ctx, m.TenantSlug)
	if err != nil {
		a.log.Error("runtime.tenant_lookup_failed", map[string]any{"tenant_slug": m.TenantSlug, "error": err.Error()})
		return err
	}

	intent, err := a.classifier.Classify(ctx, m, tc)
	if err != nil {
		a.log.Error("runtime.classify_failed", map[string]any{"tenant_slug": m.TenantSlug, "error": err.Error()})
		return err
	}

	reply, err := a.replier.Reply(ctx, m, intent, tc)
	if err != nil {
		a.log.Error("runtime.reply_failed", map[string]any{"tenant_slug": m.TenantSlug, "intent": string(intent), "error": err.Error()})
		return err
	}

	if err := a.outbound.Send(ctx, m.From, reply); err != nil {
		a.log.Error("runtime.outbound_failed", map[string]any{"tenant_slug": m.TenantSlug, "error": err.Error()})
		return err
	}

	a.log.Info("runtime.processed", map[string]any{
		"tenant_slug": m.TenantSlug, "mode": string(tc.Mode),
		"intent": string(intent), "handoff": reply.Handoff,
		"provider_message_id": m.ProviderMessageID,
	})
	return nil
}
