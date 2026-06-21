package agent

import (
	"context"
	"errors"
	"fmt"

	"github.com/hornosg/wa-agent-runtime/src/logging"
)

// AgentRuntime — orquestador con memoria de conversación (ADR-0003, E07):
// tenant → estado conversacional → (flujo de booking | clasificar+responder) → outbound.
type AgentRuntime struct {
	tenants    TenantConfigProvider
	conv       ConversationStore
	classifier IntentClassifier
	replier    Replier
	scheduler  BookingScheduler
	outbound   Outbound
	log        *logging.Logger
}

func New(tenants TenantConfigProvider, conv ConversationStore, classifier IntentClassifier, replier Replier, scheduler BookingScheduler, outbound Outbound, log *logging.Logger) *AgentRuntime {
	return &AgentRuntime{tenants: tenants, conv: conv, classifier: classifier, replier: replier, scheduler: scheduler, outbound: outbound, log: log}
}

func (a *AgentRuntime) Process(ctx context.Context, m InboundMessage) error {
	tc, err := a.tenants.Get(ctx, m.TenantSlug)
	if err != nil {
		a.log.Error("runtime.tenant_lookup_failed", map[string]any{"tenant_slug": m.TenantSlug, "error": err.Error()})
		return err
	}

	conv, err := a.conv.Get(ctx, m.TenantSlug, m.From)
	if err != nil {
		a.log.Error("runtime.conv_lookup_failed", map[string]any{"tenant_slug": m.TenantSlug, "error": err.Error()})
		return err
	}

	reply, conv, intent, err := a.decide(ctx, m, tc, conv)
	if err != nil {
		a.log.Error("runtime.decide_failed", map[string]any{"tenant_slug": m.TenantSlug, "error": err.Error()})
		return err
	}

	if err := a.conv.Save(ctx, conv); err != nil {
		a.log.Error("runtime.conv_save_failed", map[string]any{"tenant_slug": m.TenantSlug, "error": err.Error()})
		return err
	}

	if err := a.outbound.Send(ctx, m.TenantSlug, m.From, reply); err != nil {
		a.log.Error("runtime.outbound_failed", map[string]any{"tenant_slug": m.TenantSlug, "error": err.Error()})
		return err
	}

	a.log.Info("runtime.processed", map[string]any{
		"tenant_slug": m.TenantSlug, "mode": string(tc.Mode), "intent": intent,
		"state": string(conv.State), "handoff": reply.Handoff, "provider_message_id": m.ProviderMessageID,
	})
	return nil
}

// decide aplica la state machine. Devuelve la respuesta, la conversación actualizada
// y una etiqueta de intención (para logs).
func (a *AgentRuntime) decide(ctx context.Context, m InboundMessage, tc TenantConfig, conv Conversation) (Reply, Conversation, string, error) {
	// En medio de un flujo de confirmación de turno.
	if conv.State == StateAwaitingBookingConfirm && conv.Booking != nil {
		reply, conv, err := a.handleConfirm(ctx, m, tc, conv)
		return reply, conv, "booking_confirm", err
	}

	intent, err := a.classifier.Classify(ctx, m, tc)
	if err != nil {
		return Reply{}, conv, string(intent), err
	}

	// Booking en tenant con agenda → flujo stateful (propone turno real).
	if (intent == IntentBooking || intent == IntentReschedule) && tc.Mode == ModeAgenda {
		reply, conv := a.proposeBooking(ctx, m, conv)
		return reply, conv, string(intent), nil
	}

	// Resto (FAQ con RAG, cancel, handoff, booking en rag_chat → handoff, etc.).
	reply, err := a.replier.Reply(ctx, m, intent, tc)
	return reply, conv, string(intent), err
}

func (a *AgentRuntime) proposeBooking(ctx context.Context, m InboundMessage, conv Conversation) (Reply, Conversation) {
	slot, ok, err := a.scheduler.NextAvailable(ctx, m.TenantSlug)
	if err != nil || !ok {
		return Reply{Text: "¡Dale! Te ayudo a agendar. ¿Qué día y horario te queda cómodo?"}, conv
	}
	conv.State = StateAwaitingBookingConfirm
	conv.Booking = &BookingCtx{ResourceID: slot.ResourceID, SlotStart: slot.Start, SlotMinutes: slot.Minutes}
	return Reply{Text: fmt.Sprintf("Tengo un turno el %s a las %s. ¿Te lo reservo? 🙂",
		slot.Start.Format("02/01"), slot.Start.Format("15:04"))}, conv
}

func (a *AgentRuntime) handleConfirm(ctx context.Context, m InboundMessage, _ TenantConfig, conv Conversation) (Reply, Conversation, error) {
	switch {
	case isAffirmative(m.Text):
		start := conv.Booking.SlotStart
		_, err := a.scheduler.Book(ctx, m.TenantSlug, conv.Booking.ResourceID, m.From, start, conv.Booking.SlotMinutes)
		if errors.Is(err, ErrSlotTaken) {
			// Se ocupó entre la propuesta y el "sí": ofrecer el siguiente.
			if slot, ok, e2 := a.scheduler.NextAvailable(ctx, m.TenantSlug); e2 == nil && ok {
				conv.Booking = &BookingCtx{ResourceID: slot.ResourceID, SlotStart: slot.Start, SlotMinutes: slot.Minutes}
				return Reply{Text: fmt.Sprintf("Uy, justo se ocupó ese horario 😅. Te ofrezco el %s a las %s, ¿te lo reservo?",
					slot.Start.Format("02/01"), slot.Start.Format("15:04"))}, conv, nil
			}
			return Reply{Text: "Uy, se ocupó y no tengo otro cercano. Te paso con alguien del equipo. 🙌", Handoff: true}, idle(conv), nil
		}
		if err != nil {
			return Reply{}, conv, err
		}
		return Reply{Text: fmt.Sprintf("¡Listo! Te reservé el turno el %s a las %s. Te esperamos 🙌",
			start.Format("02/01"), start.Format("15:04"))}, idle(conv), nil

	case isNegative(m.Text):
		return Reply{Text: "Dale, sin problema. ¿Qué día y horario te queda mejor?"}, idle(conv), nil

	default:
		return Reply{Text: "¿Te confirmo ese turno? Respondé *sí* o *no* 🙂"}, conv, nil
	}
}

func idle(c Conversation) Conversation {
	c.State = StateIdle
	c.Booking = nil
	return c
}
