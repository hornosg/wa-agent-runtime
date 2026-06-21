package agent

import (
	"context"
	"time"
)

// SlotInfo — un turno disponible propuesto al contacto.
type SlotInfo struct {
	Start        time.Time
	End          time.Time
	ResourceName string
}

// BookingScheduler — puerto: el runtime consulta disponibilidad para proponer
// el próximo turno (la disponibilidad sale SOLO de Postgres — P-05/P-10).
type BookingScheduler interface {
	NextAvailable(ctx context.Context, tenantSlug string) (SlotInfo, bool, error)
}
