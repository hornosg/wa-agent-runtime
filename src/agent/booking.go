package agent

import (
	"context"
	"errors"
	"time"
)

// ErrSlotTaken — el turno se solapa con uno confirmado (lo rechaza Postgres).
// Vive en agent para que el flujo lo detecte sin importar el paquete scheduling.
var ErrSlotTaken = errors.New("slot ya reservado")

// SlotInfo — un turno disponible (lo necesario para proponerlo y luego reservarlo).
type SlotInfo struct {
	ResourceID   string
	ResourceName string
	Start        time.Time
	End          time.Time
	Minutes      int
}

// BookingScheduler — puerto: disponibilidad y reserva (SoT Postgres, P-05/P-10).
type BookingScheduler interface {
	NextAvailable(ctx context.Context, tenantSlug string) (SlotInfo, bool, error)
	Book(ctx context.Context, tenantSlug, resourceID, contact string, start time.Time, slotMinutes int) (bookingID string, err error)
}
