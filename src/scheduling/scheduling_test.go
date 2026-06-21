package scheduling

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/hornosg/wa-agent-runtime/src/agent"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Integration test contra lab-postgres. Correr con:
//
//	SCHEDULING_TEST_DSN="postgres://whatsapp_agent:whatsapp_agent@localhost:5432/whatsapp_agent?sslmode=disable" \
//	  go test ./src/scheduling/ -run TestAntiDoubleBooking -v
func TestAntiDoubleBooking(t *testing.T) {
	dsn := os.Getenv("SCHEDULING_TEST_DSN")
	if dsn == "" {
		t.Skip("SCHEDULING_TEST_DSN no seteado — test de integración omitido")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	const tenant = "test_scheduling"
	cleanup := func() {
		_, _ = pool.Exec(ctx, `DELETE FROM scheduling.booking WHERE tenant_slug=$1`, tenant)
		_, _ = pool.Exec(ctx, `DELETE FROM scheduling.resource WHERE tenant_slug=$1`, tenant)
	}
	cleanup()
	defer cleanup()

	s := NewPgScheduler(pool)
	rid, err := s.EnsureResource(ctx, tenant, "Recurso Test")
	if err != nil {
		t.Fatalf("EnsureResource: %v", err)
	}

	slot, ok, err := s.NextAvailable(ctx, tenant)
	if err != nil || !ok {
		t.Fatalf("NextAvailable inicial: ok=%v err=%v", ok, err)
	}

	// 1) Reservar el slot.
	id1, err := s.Book(ctx, tenant, rid, "5491100000001", slot.Start, 30)
	if err != nil {
		t.Fatalf("CreateBooking 1: %v", err)
	}

	// 2) ANTI DOBLE-RESERVA: el mismo slot otra vez → ErrSlotTaken.
	if _, err := s.Book(ctx, tenant, rid, "5491100000002", slot.Start, 30); !errors.Is(err, agent.ErrSlotTaken) {
		t.Fatalf("doble reserva debería fallar con ErrSlotTaken, got: %v", err)
	}

	// 3) NextAvailable ahora propone otro slot (excluye el reservado).
	slot2, ok2, err := s.NextAvailable(ctx, tenant)
	if err != nil || !ok2 || slot2.Start.Equal(slot.Start) {
		t.Fatalf("NextAvailable tras reserva debería diferir: ok=%v start=%v err=%v", ok2, slot2.Start, err)
	}

	// 4) Cancelar libera el slot → se puede volver a reservar.
	if err := s.CancelBooking(ctx, id1); err != nil {
		t.Fatalf("CancelBooking: %v", err)
	}
	if _, err := s.Book(ctx, tenant, rid, "5491100000003", slot.Start, 30); err != nil {
		t.Fatalf("CreateBooking tras cancelar (slot libre): %v", err)
	}
}
