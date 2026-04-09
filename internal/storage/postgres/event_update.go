package postgres

import (
	"context"
	"time"

	"fresnel/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EventUpdateStore struct {
	pool *pgxpool.Pool
}

func NewEventUpdateStore(pool *pgxpool.Pool) *EventUpdateStore {
	return &EventUpdateStore{pool: pool}
}

func (s *EventUpdateStore) Create(ctx context.Context, u *domain.EventUpdate) error {
	var impactChange, statusChange *string
	if u.ImpactChange != nil {
		v := string(*u.ImpactChange)
		impactChange = &v
	}
	if u.StatusChange != nil {
		v := string(*u.StatusChange)
		statusChange = &v
	}
	_, err := s.pool.Exec(ctx, `
INSERT INTO fresnel.event_updates (id, event_id, author_id, body, tlp, impact_change, status_change, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		u.ID, u.EventID, u.AuthorID, u.Body, string(u.TLP), impactChange, statusChange, u.CreatedAt,
	)
	return err
}

func (s *EventUpdateStore) ListByEvent(ctx context.Context, eventID uuid.UUID) ([]*domain.EventUpdate, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, event_id, author_id, body, tlp, impact_change, status_change, created_at
FROM fresnel.event_updates WHERE event_id = $1 ORDER BY created_at`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.EventUpdate
	for rows.Next() {
		var u domain.EventUpdate
		var impactChange, statusChange *string
		if err := rows.Scan(&u.ID, &u.EventID, &u.AuthorID, &u.Body, &u.TLP, &impactChange, &statusChange, &u.CreatedAt); err != nil {
			return nil, err
		}
		if impactChange != nil {
			v := domain.Impact(*impactChange)
			u.ImpactChange = &v
		}
		if statusChange != nil {
			v := domain.EventStatus(*statusChange)
			u.StatusChange = &v
		}
		out = append(out, &u)
	}
	return out, rows.Err()
}

func (s *EventUpdateStore) LatestCreatedAt(ctx context.Context, eventID uuid.UUID) (time.Time, bool, error) {
	var t time.Time
	err := s.pool.QueryRow(ctx, `
SELECT created_at FROM fresnel.event_updates WHERE event_id = $1 ORDER BY created_at DESC LIMIT 1`,
		eventID,
	).Scan(&t)
	if err != nil {
		if err == pgx.ErrNoRows {
			return time.Time{}, false, nil
		}
		return time.Time{}, false, err
	}
	return t, true, nil
}
