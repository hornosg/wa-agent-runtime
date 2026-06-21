package agent

import (
	"context"
	"strings"
	"time"
)

// ConvState — estado del flujo conversacional (E07).
type ConvState string

const (
	StateIdle                   ConvState = "idle"
	StateAwaitingBookingConfirm ConvState = "awaiting_booking_confirm"
)

// BookingCtx — turno propuesto, a la espera de confirmación.
type BookingCtx struct {
	ResourceID  string    `json:"resource_id"`
	SlotStart   time.Time `json:"slot_start"`
	SlotMinutes int       `json:"slot_minutes"`
}

// Conversation — estado por (tenant, contacto).
type Conversation struct {
	TenantSlug string
	Contact    string
	State      ConvState
	Booking    *BookingCtx // set cuando State == awaiting_booking_confirm
}

// ConversationStore — puerto de persistencia del estado conversacional.
type ConversationStore interface {
	Get(ctx context.Context, tenantSlug, contact string) (Conversation, error)
	Save(ctx context.Context, c Conversation) error
}

// isAffirmative / isNegative — detección simple de confirmación (es-AR).
func isAffirmative(text string) bool {
	return matchesAny(text, "si", "sí", "dale", "ok", "oka", "okay", "listo", "perfecto", "confirmo", "confirmar", "de una", "obvio", "reserva", "reservalo", "reservá")
}

func isNegative(text string) bool {
	return matchesAny(text, "no", "otro", "otra", "cambiar", "mejor no", "después", "despues", "más tarde", "mas tarde")
}

func matchesAny(text string, words ...string) bool {
	t := " " + strings.ToLower(strings.TrimSpace(text)) + " "
	for _, w := range words {
		if strings.Contains(t, " "+w+" ") || t == " "+w+" " {
			return true
		}
	}
	return false
}
