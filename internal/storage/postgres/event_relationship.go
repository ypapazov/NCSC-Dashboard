package postgres

import (
	"context"

	"fresnel/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EventRelationshipStore struct {
	pool *pgxpool.Pool
}

func NewEventRelationshipStore(pool *pgxpool.Pool) *EventRelationshipStore {
	return &EventRelationshipStore{pool: pool}
}

func (s *EventRelationshipStore) Create(ctx context.Context, r *domain.EventRelationship) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO fresnel.event_relationships (id, source_event_id, target_event_id, label, created_by_user, created_by_agent, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		r.ID, r.SourceEventID, r.TargetEventID, r.Label,
		uuidToNullable(r.CreatedByUser), nilIfEmpty(r.CreatedByAgent), r.CreatedAt,
	)
	return err
}

func (s *EventRelationshipStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.EventRelationship, error) {
	var r domain.EventRelationship
	var createdByUser pgtype.UUID
	var createdByAgent *string
	err := s.pool.QueryRow(ctx, `
SELECT id, source_event_id, target_event_id, label, created_by_user, created_by_agent, created_at
FROM fresnel.event_relationships WHERE id = $1`, id,
	).Scan(&r.ID, &r.SourceEventID, &r.TargetEventID, &r.Label,
		&createdByUser, &createdByAgent, &r.CreatedAt)
	if err != nil {
		return nil, err
	}
	r.CreatedByUser = nullableToUUID(createdByUser)
	if createdByAgent != nil {
		r.CreatedByAgent = *createdByAgent
	}
	return &r, nil
}

func (s *EventRelationshipStore) ListByEvent(ctx context.Context, eventID uuid.UUID) ([]*domain.EventRelationship, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, source_event_id, target_event_id, label, created_by_user, created_by_agent, created_at
FROM fresnel.event_relationships
WHERE source_event_id = $1 OR target_event_id = $1
ORDER BY created_at`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.EventRelationship
	for rows.Next() {
		var r domain.EventRelationship
		var createdByUser pgtype.UUID
		var createdByAgent *string
		if err := rows.Scan(&r.ID, &r.SourceEventID, &r.TargetEventID, &r.Label,
			&createdByUser, &createdByAgent, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.CreatedByUser = nullableToUUID(createdByUser)
		if createdByAgent != nil {
			r.CreatedByAgent = *createdByAgent
		}
		out = append(out, &r)
	}
	return out, rows.Err()
}

func (s *EventRelationshipStore) ListByEventIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.EventRelationship, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, `
SELECT id, source_event_id, target_event_id, label, created_by_user, created_by_agent, created_at
FROM fresnel.event_relationships
WHERE source_event_id = ANY($1) AND target_event_id = ANY($1)
ORDER BY created_at`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.EventRelationship
	for rows.Next() {
		var r domain.EventRelationship
		var createdByUser pgtype.UUID
		var createdByAgent *string
		if err := rows.Scan(&r.ID, &r.SourceEventID, &r.TargetEventID, &r.Label,
			&createdByUser, &createdByAgent, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.CreatedByUser = nullableToUUID(createdByUser)
		if createdByAgent != nil {
			r.CreatedByAgent = *createdByAgent
		}
		out = append(out, &r)
	}
	return out, rows.Err()
}

func (s *EventRelationshipStore) Delete(ctx context.Context, id uuid.UUID) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM fresnel.event_relationships WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
