package postgres

import (
	"context"
	"fmt"
	"strings"

	"fresnel/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EventStore struct {
	pool *pgxpool.Pool
}

func NewEventStore(pool *pgxpool.Pool) *EventStore {
	return &EventStore{pool: pool}
}

func (s *EventStore) Create(ctx context.Context, e *domain.Event) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO fresnel.events
    (id, source_instance, sector_context, title, description, event_type, submitter_id, organization_id, tlp, impact, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		e.ID, e.SourceInstance, e.SectorContext, e.Title, e.Description,
		string(e.EventType), e.SubmitterID, e.OrganizationID,
		string(e.TLP), string(e.Impact), string(e.Status),
		e.CreatedAt, e.UpdatedAt,
	)
	return err
}

func (s *EventStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.Event, error) {
	var e domain.Event
	err := s.pool.QueryRow(ctx, `
SELECT id, source_instance, sector_context, title, description, event_type, submitter_id, organization_id,
       tlp, impact, status, created_at, updated_at
FROM fresnel.events WHERE id = $1`, id,
	).Scan(&e.ID, &e.SourceInstance, &e.SectorContext, &e.Title, &e.Description,
		&e.EventType, &e.SubmitterID, &e.OrganizationID,
		&e.TLP, &e.Impact, &e.Status,
		&e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (s *EventStore) List(ctx context.Context, f domain.EventFilter) (*domain.ListResult[*domain.Event], error) {
	f.Normalize()

	var conditions []string
	var args []any
	idx := 1

	if f.SectorContextID != nil {
		conditions = append(conditions, fmt.Sprintf("sector_context = $%d", idx))
		args = append(args, *f.SectorContextID)
		idx++
	}
	if f.OrganizationID != nil {
		conditions = append(conditions, fmt.Sprintf("organization_id = $%d", idx))
		args = append(args, *f.OrganizationID)
		idx++
	}
	if f.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", idx))
		args = append(args, string(*f.Status))
		idx++
	}
	if f.Impact != nil {
		conditions = append(conditions, fmt.Sprintf("impact = $%d", idx))
		args = append(args, string(*f.Impact))
		idx++
	}
	if f.EventType != nil {
		conditions = append(conditions, fmt.Sprintf("event_type = $%d", idx))
		args = append(args, string(*f.EventType))
		idx++
	}
	if f.TLP != nil {
		conditions = append(conditions, fmt.Sprintf("tlp = $%d", idx))
		args = append(args, string(*f.TLP))
		idx++
	}
	if f.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(title ILIKE '%%' || $%d || '%%' OR description ILIKE '%%' || $%d || '%%')", idx, idx))
		args = append(args, f.Search)
		idx++
	}
	if f.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", idx))
		args = append(args, *f.DateFrom)
		idx++
	}
	if f.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", idx))
		args = append(args, *f.DateTo)
		idx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	orderCol := "created_at"
	if f.SortBy == "updated_at" {
		orderCol = "updated_at"
	}

	q := fmt.Sprintf(`
SELECT id, source_instance, sector_context, title, description, event_type, submitter_id, organization_id,
       tlp, impact, status, created_at, updated_at,
       COUNT(*) OVER() AS total
FROM fresnel.events
%s
ORDER BY %s DESC
LIMIT $%d OFFSET $%d`, where, orderCol, idx, idx+1)
	args = append(args, f.Limit, f.Offset)

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := &domain.ListResult[*domain.Event]{}
	for rows.Next() {
		var e domain.Event
		var total int
		if err := rows.Scan(
			&e.ID, &e.SourceInstance, &e.SectorContext, &e.Title, &e.Description,
			&e.EventType, &e.SubmitterID, &e.OrganizationID,
			&e.TLP, &e.Impact, &e.Status,
			&e.CreatedAt, &e.UpdatedAt,
			&total,
		); err != nil {
			return nil, err
		}
		result.TotalCount = total
		result.Items = append(result.Items, &e)
	}
	return result, rows.Err()
}

func (s *EventStore) Update(ctx context.Context, e *domain.Event, changedBy uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var revNum int
	err = tx.QueryRow(ctx, `
SELECT COALESCE(MAX(revision_number), 0) FROM fresnel.event_revisions WHERE event_id = $1`, e.ID).Scan(&revNum)
	if err != nil {
		return err
	}
	revNum++

	_, err = tx.Exec(ctx, `
INSERT INTO fresnel.event_revisions
    (id, event_id, revision_number, title, description, event_type, tlp, impact, status, changed_by)
SELECT gen_random_uuid(), id, $2, title, description, event_type, tlp, impact, status, $3
FROM fresnel.events WHERE id = $1`, e.ID, revNum, changedBy)
	if err != nil {
		return err
	}

	ct, err := tx.Exec(ctx, `
UPDATE fresnel.events SET
    title = $2, description = $3, event_type = $4, tlp = $5, impact = $6, status = $7, updated_at = now()
WHERE id = $1`,
		e.ID, e.Title, e.Description, string(e.EventType), string(e.TLP), string(e.Impact), string(e.Status),
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return tx.Commit(ctx)
}

func (s *EventStore) Delete(ctx context.Context, id uuid.UUID) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM fresnel.events WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *EventStore) GetRevisions(ctx context.Context, eventID uuid.UUID) ([]*domain.EventRevision, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, event_id, revision_number, title, description, event_type, tlp, impact, status, changed_by, changed_at
FROM fresnel.event_revisions WHERE event_id = $1 ORDER BY revision_number`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.EventRevision
	for rows.Next() {
		var r domain.EventRevision
		if err := rows.Scan(&r.ID, &r.EventID, &r.RevisionNumber, &r.Title, &r.Description,
			&r.EventType, &r.TLP, &r.Impact, &r.Status, &r.ChangedBy, &r.ChangedAt); err != nil {
			return nil, err
		}
		out = append(out, &r)
	}
	return out, rows.Err()
}

func (s *EventStore) CountByOrg(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM fresnel.events WHERE organization_id = $1`, orgID).Scan(&count)
	return count, err
}
