package agent

import (
	"context"
	"strings"
)

// StubClassifier — clasificador L1 por palabras clave. Sin API key; para dev/CI.
// El real es AnthropicClassifier (Haiku, ADR-0002).
type StubClassifier struct{}

func NewStubClassifier() *StubClassifier { return &StubClassifier{} }

func (StubClassifier) Classify(_ context.Context, m InboundMessage, _ TenantConfig) (Intent, error) {
	t := strings.ToLower(m.Text)
	switch {
	case containsAny(t, "cancel", "anular"):
		return IntentCancel, nil
	case containsAny(t, "reprogram", "reagend", "cambiar el turno", "mover el turno"):
		return IntentReschedule, nil
	case containsAny(t, "turno", "reserva", "agendar", "cita", "disponib"):
		return IntentBooking, nil
	case containsAny(t, "humano", "persona", "hablar con"):
		return IntentHandoff, nil
	case containsAny(t, "?", "precio", "horario", "atienden", "donde", "dónde", "cuanto", "cuánto"):
		return IntentFAQ, nil
	default:
		return IntentOther, nil
	}
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
