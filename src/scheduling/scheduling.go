// Package scheduling (E06) — dominio de turnos sobre Postgres (SoT, P-10).
// Anti doble-reserva por EXCLUDE constraint (no lock a nivel app).
package scheduling

import (
	"context"
	"errors"
	"time"

	"github.com/hornosg/wa-agent-runtime/src/agent"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type resourceRow struct {
	id          string
	nombre      string
	slotMinutes int
	openMinute  int
	closeMinute int
}

// PgScheduler implementa agent.BookingScheduler y las operaciones de booking.
type PgScheduler struct {
	pool        *pgxpool.Pool
	horizonDays int
}

func NewPgScheduler(pool *pgxpool.Pool) *PgScheduler {
	return &PgScheduler{pool: pool, horizonDays: 7}
}

// NextAvailable devuelve el primer turno libre del tenant (primer recurso), dentro
// del horizonte, posterior a ahora. Disponibilidad = ventana del recurso menos
// bookings confirmados (todo desde Postgres).
func (s *PgScheduler) NextAvailable(ctx context.Context, tenantSlug string) (agent.SlotInfo, bool, error) {
	res, err := s.firstResource(ctx, tenantSlug)
	if err != nil {
		return agent.SlotInfo{}, false, err
	}
	if res == nil {
		return agent.SlotInfo{}, false, nil // tenant sin recursos cargados
	}

	now := time.Now().UTC()
	busy, err := s.confirmedRanges(ctx, res.id, now, now.AddDate(0, 0, s.horizonDays))
	if err != nil {
		return agent.SlotInfo{}, false, err
	}

	dur := time.Duration(res.slotMinutes) * time.Minute
	for d := 0; d < s.horizonDays; d++ {
		day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, d)
		for m := res.openMinute; m+res.slotMinutes <= res.closeMinute; m += res.slotMinutes {
			start := day.Add(time.Duration(m) * time.Minute)
			if !start.After(now) {
				continue
			}
			end := start.Add(dur)
			if overlapsAny(start, end, busy) {
				continue
			}
			return agent.SlotInfo{
				ResourceID: res.id, ResourceName: res.nombre,
				Start: start, End: end, Minutes: res.slotMinutes,
			}, true, nil
		}
	}
	return agent.SlotInfo{}, false, nil
}

// Book inserta un turno (implementa agent.BookingScheduler). Si se solapa con uno
// confirmado, la EXCLUDE constraint lo rechaza → agent.ErrSlotTaken. Safe por Postgres.
func (s *PgScheduler) Book(ctx context.Context, tenantSlug, resourceID, contact string, start time.Time, slotMinutes int) (string, error) {
	end := start.Add(time.Duration(slotMinutes) * time.Minute)
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO scheduling.booking (tenant_slug, resource_id, contact, during)
		VALUES ($1, $2, $3, tstzrange($4, $5, '[)'))
		RETURNING id`,
		tenantSlug, resourceID, contact, start, end,
	).Scan(&id)
	if err != nil {
		if isExclusionViolation(err) {
			return "", agent.ErrSlotTaken
		}
		return "", err
	}
	return id, nil
}

// CancelBooking marca el turno como cancelado (libera el slot).
func (s *PgScheduler) CancelBooking(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `UPDATE scheduling.booking SET status='cancelled' WHERE id=$1`, id)
	return err
}

// EnsureResource crea (si no existe) un recurso para el tenant. Util para seed/tests.
func (s *PgScheduler) EnsureResource(ctx context.Context, tenantSlug, nombre string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO scheduling.resource (tenant_slug, nombre) VALUES ($1, $2) RETURNING id`,
		tenantSlug, nombre,
	).Scan(&id)
	return id, err
}

// ── helpers ──────────────────────────────────────────────────────────────────

func (s *PgScheduler) firstResource(ctx context.Context, tenantSlug string) (*resourceRow, error) {
	var r resourceRow
	err := s.pool.QueryRow(ctx, `
		SELECT id, nombre, slot_minutes, open_minute, close_minute
		FROM scheduling.resource WHERE tenant_slug=$1 ORDER BY created_at LIMIT 1`, tenantSlug,
	).Scan(&r.id, &r.nombre, &r.slotMinutes, &r.openMinute, &r.closeMinute)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

type rng struct{ start, end time.Time }

func (s *PgScheduler) confirmedRanges(ctx context.Context, resourceID string, from, to time.Time) ([]rng, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT lower(during), upper(during) FROM scheduling.booking
		WHERE resource_id=$1 AND status='confirmed' AND during && tstzrange($2,$3,'[)')`,
		resourceID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []rng
	for rows.Next() {
		var r rng
		if err := rows.Scan(&r.start, &r.end); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func overlapsAny(start, end time.Time, busy []rng) bool {
	for _, b := range busy {
		if start.Before(b.end) && b.start.Before(end) {
			return true
		}
	}
	return false
}

func isExclusionViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23P01" // exclusion_violation
	}
	return false
}
