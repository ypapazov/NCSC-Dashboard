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

type StatusReportStore struct {
	pool *pgxpool.Pool
}

func NewStatusReportStore(pool *pgxpool.Pool) *StatusReportStore {
	return &StatusReportStore{pool: pool}
}

func (s *StatusReportStore) Create(ctx context.Context, sr *domain.StatusReport) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO fresnel.status_reports
    (id, source_instance, sector_context, scope_type, scope_ref, title, body,
     period_covered_start, period_covered_end, as_of, published_at,
     assessed_status, impact, tlp, author_id, organization_id, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`,
		sr.ID, sr.SourceInstance, sr.SectorContext, sr.ScopeType, sr.ScopeRef,
		sr.Title, sr.Body,
		sr.PeriodCoveredStart, sr.PeriodCoveredEnd, sr.AsOf, sr.PublishedAt,
		string(sr.AssessedStatus), string(sr.Impact), string(sr.TLP),
		sr.AuthorID, sr.OrganizationID, sr.CreatedAt, sr.UpdatedAt,
	)
	return err
}

func (s *StatusReportStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.StatusReport, error) {
	var sr domain.StatusReport
	err := s.pool.QueryRow(ctx, `
SELECT id, source_instance, sector_context, scope_type, scope_ref, title, body,
       period_covered_start, period_covered_end, as_of, published_at,
       assessed_status, impact, tlp, author_id, organization_id, created_at, updated_at
FROM fresnel.status_reports WHERE id = $1`, id,
	).Scan(&sr.ID, &sr.SourceInstance, &sr.SectorContext, &sr.ScopeType, &sr.ScopeRef,
		&sr.Title, &sr.Body,
		&sr.PeriodCoveredStart, &sr.PeriodCoveredEnd, &sr.AsOf, &sr.PublishedAt,
		&sr.AssessedStatus, &sr.Impact, &sr.TLP,
		&sr.AuthorID, &sr.OrganizationID, &sr.CreatedAt, &sr.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &sr, nil
}

func (s *StatusReportStore) List(ctx context.Context, f domain.StatusReportFilter) (*domain.ListResult[*domain.StatusReport], error) {
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
	if f.ScopeType != "" {
		conditions = append(conditions, fmt.Sprintf("scope_type = $%d", idx))
		args = append(args, f.ScopeType)
		idx++
	}
	if f.ScopeRef != nil {
		conditions = append(conditions, fmt.Sprintf("scope_ref = $%d", idx))
		args = append(args, *f.ScopeRef)
		idx++
	}
	if f.AssessedStatus != nil {
		conditions = append(conditions, fmt.Sprintf("assessed_status = $%d", idx))
		args = append(args, string(*f.AssessedStatus))
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

	q := fmt.Sprintf(`
SELECT id, source_instance, sector_context, scope_type, scope_ref, title, body,
       period_covered_start, period_covered_end, as_of, published_at,
       assessed_status, impact, tlp, author_id, organization_id, created_at, updated_at,
       COUNT(*) OVER() AS total
FROM fresnel.status_reports
%s
ORDER BY created_at DESC
LIMIT $%d OFFSET $%d`, where, idx, idx+1)
	args = append(args, f.Limit, f.Offset)

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := &domain.ListResult[*domain.StatusReport]{}
	for rows.Next() {
		var sr domain.StatusReport
		var total int
		if err := rows.Scan(
			&sr.ID, &sr.SourceInstance, &sr.SectorContext, &sr.ScopeType, &sr.ScopeRef,
			&sr.Title, &sr.Body,
			&sr.PeriodCoveredStart, &sr.PeriodCoveredEnd, &sr.AsOf, &sr.PublishedAt,
			&sr.AssessedStatus, &sr.Impact, &sr.TLP,
			&sr.AuthorID, &sr.OrganizationID, &sr.CreatedAt, &sr.UpdatedAt,
			&total,
		); err != nil {
			return nil, err
		}
		result.TotalCount = total
		result.Items = append(result.Items, &sr)
	}
	return result, rows.Err()
}

func (s *StatusReportStore) Update(ctx context.Context, sr *domain.StatusReport, changedBy uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var revNum int
	err = tx.QueryRow(ctx, `
SELECT COALESCE(MAX(revision_number), 0) FROM fresnel.status_report_revisions WHERE status_report_id = $1`, sr.ID).Scan(&revNum)
	if err != nil {
		return err
	}
	revNum++

	_, err = tx.Exec(ctx, `
INSERT INTO fresnel.status_report_revisions
    (id, status_report_id, revision_number, title, body, assessed_status, impact, tlp, changed_by)
SELECT gen_random_uuid(), id, $2, title, body, assessed_status, impact, tlp, $3
FROM fresnel.status_reports WHERE id = $1`, sr.ID, revNum, changedBy)
	if err != nil {
		return err
	}

	ct, err := tx.Exec(ctx, `
UPDATE fresnel.status_reports SET
    title = $2, body = $3, assessed_status = $4, impact = $5, tlp = $6,
    period_covered_start = $7, period_covered_end = $8, as_of = $9, updated_at = now()
WHERE id = $1`,
		sr.ID, sr.Title, sr.Body, string(sr.AssessedStatus), string(sr.Impact), string(sr.TLP),
		sr.PeriodCoveredStart, sr.PeriodCoveredEnd, sr.AsOf,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return tx.Commit(ctx)
}

func (s *StatusReportStore) Delete(ctx context.Context, id uuid.UUID) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM fresnel.status_reports WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *StatusReportStore) GetRevisions(ctx context.Context, reportID uuid.UUID) ([]*domain.StatusReportRevision, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, status_report_id, revision_number, title, body, assessed_status, impact, tlp, changed_by, changed_at
FROM fresnel.status_report_revisions WHERE status_report_id = $1 ORDER BY revision_number`, reportID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.StatusReportRevision
	for rows.Next() {
		var r domain.StatusReportRevision
		if err := rows.Scan(&r.ID, &r.StatusReportID, &r.RevisionNumber, &r.Title, &r.Body,
			&r.AssessedStatus, &r.Impact, &r.TLP, &r.ChangedBy, &r.ChangedAt); err != nil {
			return nil, err
		}
		out = append(out, &r)
	}
	return out, rows.Err()
}

func (s *StatusReportStore) LinkEvents(ctx context.Context, reportID uuid.UUID, eventIDs []uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, eid := range eventIDs {
		_, err = tx.Exec(ctx, `
INSERT INTO fresnel.status_report_events (status_report_id, event_id)
VALUES ($1, $2)
ON CONFLICT (status_report_id, event_id) DO NOTHING`, reportID, eid)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (s *StatusReportStore) GetLinkedEventIDs(ctx context.Context, reportID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := s.pool.Query(ctx, `
SELECT event_id FROM fresnel.status_report_events WHERE status_report_id = $1`, reportID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []uuid.UUID
	for rows.Next() {
		var eid uuid.UUID
		if err := rows.Scan(&eid); err != nil {
			return nil, err
		}
		out = append(out, eid)
	}
	return out, rows.Err()
}

func (s *StatusReportStore) GetLatestByScope(ctx context.Context, scopeType string, scopeRef uuid.UUID) (*domain.StatusReport, error) {
	var sr domain.StatusReport
	err := s.pool.QueryRow(ctx, `
SELECT id, source_instance, sector_context, scope_type, scope_ref, title, body,
       period_covered_start, period_covered_end, as_of, published_at,
       assessed_status, impact, tlp, author_id, organization_id, created_at, updated_at
FROM fresnel.status_reports
WHERE scope_type = $1 AND scope_ref = $2
ORDER BY published_at DESC
LIMIT 1`, scopeType, scopeRef,
	).Scan(&sr.ID, &sr.SourceInstance, &sr.SectorContext, &sr.ScopeType, &sr.ScopeRef,
		&sr.Title, &sr.Body,
		&sr.PeriodCoveredStart, &sr.PeriodCoveredEnd, &sr.AsOf, &sr.PublishedAt,
		&sr.AssessedStatus, &sr.Impact, &sr.TLP,
		&sr.AuthorID, &sr.OrganizationID, &sr.CreatedAt, &sr.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &sr, nil
}
