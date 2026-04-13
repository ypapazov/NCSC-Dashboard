package postgres

import (
	"context"

	"fresnel/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CorrelationStore struct {
	pool *pgxpool.Pool
}

func NewCorrelationStore(pool *pgxpool.Pool) *CorrelationStore {
	return &CorrelationStore{pool: pool}
}

func (s *CorrelationStore) Create(ctx context.Context, c *domain.Correlation) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO fresnel.correlations (id, event_a_id, event_b_id, label, correlation_type, created_by_user, created_by_agent, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		c.ID, c.EventAID, c.EventBID, c.Label, string(c.CorrelationType),
		uuidToNullable(c.CreatedByUser), nilIfEmpty(c.CreatedByAgent), c.CreatedAt,
	)
	return err
}

func (s *CorrelationStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.Correlation, error) {
	var c domain.Correlation
	var createdByUser pgtype.UUID
	var createdByAgent *string
	err := s.pool.QueryRow(ctx, `
SELECT id, event_a_id, event_b_id, label, correlation_type, created_by_user, created_by_agent, created_at
FROM fresnel.correlations WHERE id = $1`, id,
	).Scan(&c.ID, &c.EventAID, &c.EventBID, &c.Label, &c.CorrelationType,
		&createdByUser, &createdByAgent, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	c.CreatedByUser = nullableToUUID(createdByUser)
	if createdByAgent != nil {
		c.CreatedByAgent = *createdByAgent
	}
	return &c, nil
}

func (s *CorrelationStore) ListByEvent(ctx context.Context, eventID uuid.UUID) ([]*domain.Correlation, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, event_a_id, event_b_id, label, correlation_type, created_by_user, created_by_agent, created_at
FROM fresnel.correlations WHERE event_a_id = $1 OR event_b_id = $1
ORDER BY created_at`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Correlation
	for rows.Next() {
		var c domain.Correlation
		var createdByUser pgtype.UUID
		var createdByAgent *string
		if err := rows.Scan(&c.ID, &c.EventAID, &c.EventBID, &c.Label, &c.CorrelationType,
			&createdByUser, &createdByAgent, &c.CreatedAt); err != nil {
			return nil, err
		}
		c.CreatedByUser = nullableToUUID(createdByUser)
		if createdByAgent != nil {
			c.CreatedByAgent = *createdByAgent
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

func (s *CorrelationStore) ListByEventIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.Correlation, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, `
SELECT id, event_a_id, event_b_id, label, correlation_type, created_by_user, created_by_agent, created_at
FROM fresnel.correlations
WHERE event_a_id = ANY($1) AND event_b_id = ANY($1)
ORDER BY created_at`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Correlation
	for rows.Next() {
		var c domain.Correlation
		var createdByUser pgtype.UUID
		var createdByAgent *string
		if err := rows.Scan(&c.ID, &c.EventAID, &c.EventBID, &c.Label, &c.CorrelationType,
			&createdByUser, &createdByAgent, &c.CreatedAt); err != nil {
			return nil, err
		}
		c.CreatedByUser = nullableToUUID(createdByUser)
		if createdByAgent != nil {
			c.CreatedByAgent = *createdByAgent
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

func (s *CorrelationStore) Delete(ctx context.Context, id uuid.UUID) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM fresnel.correlations WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *CorrelationStore) UpdateType(ctx context.Context, id uuid.UUID, corrType domain.CorrelationType) error {
	ct, err := s.pool.Exec(ctx, `
UPDATE fresnel.correlations SET correlation_type = $2 WHERE id = $1`, id, string(corrType))
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
