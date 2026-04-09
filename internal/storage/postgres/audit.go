package postgres

import (
	"context"
	"fmt"
	"strings"

	"fresnel/internal/domain"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditStore struct {
	pool *pgxpool.Pool
}

func NewAuditStore(pool *pgxpool.Pool) *AuditStore {
	return &AuditStore{pool: pool}
}

func (s *AuditStore) Insert(ctx context.Context, e *domain.AuditEntry) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO fresnel_audit.audit_entries
    (id, timestamp, actor_id, actor_type, action, resource_type, resource_id, scope_type, scope_id, detail, severity, ip_address, user_agent)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12::inet, $13)`,
		e.ID, e.Timestamp, e.ActorID, e.ActorType, e.Action,
		e.ResourceType, uuidToNullable(e.ResourceID),
		e.ScopeType, uuidToNullable(e.ScopeID),
		e.Detail, string(e.Severity),
		nilIfEmpty(e.IPAddress), e.UserAgent,
	)
	return err
}

func (s *AuditStore) List(ctx context.Context, f domain.AuditFilter) (*domain.ListResult[*domain.AuditEntry], error) {
	f.Normalize()

	var conditions []string
	var args []any
	idx := 1

	if f.ActorID != nil {
		conditions = append(conditions, fmt.Sprintf("actor_id = $%d", idx))
		args = append(args, *f.ActorID)
		idx++
	}
	if f.ResourceType != "" {
		conditions = append(conditions, fmt.Sprintf("resource_type = $%d", idx))
		args = append(args, f.ResourceType)
		idx++
	}
	if f.ResourceID != nil {
		conditions = append(conditions, fmt.Sprintf("resource_id = $%d", idx))
		args = append(args, *f.ResourceID)
		idx++
	}
	if f.ScopeType != "" {
		conditions = append(conditions, fmt.Sprintf("scope_type = $%d", idx))
		args = append(args, f.ScopeType)
		idx++
	}
	if f.ScopeID != nil {
		conditions = append(conditions, fmt.Sprintf("scope_id = $%d", idx))
		args = append(args, *f.ScopeID)
		idx++
	}
	if f.Severity != nil {
		conditions = append(conditions, fmt.Sprintf("severity = $%d", idx))
		args = append(args, string(*f.Severity))
		idx++
	}
	if f.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", idx))
		args = append(args, *f.DateFrom)
		idx++
	}
	if f.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", idx))
		args = append(args, *f.DateTo)
		idx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	q := fmt.Sprintf(`
SELECT id, timestamp, actor_id, actor_type, action, resource_type, resource_id, scope_type, scope_id, detail, severity, ip_address, user_agent,
       COUNT(*) OVER() AS total
FROM fresnel_audit.audit_entries
%s
ORDER BY timestamp DESC
LIMIT $%d OFFSET $%d`, where, idx, idx+1)
	args = append(args, f.Limit, f.Offset)

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := &domain.ListResult[*domain.AuditEntry]{}
	for rows.Next() {
		var e domain.AuditEntry
		var resourceID, scopeID pgtype.UUID
		var ipAddr, scopeType *string
		var total int
		if err := rows.Scan(
			&e.ID, &e.Timestamp, &e.ActorID, &e.ActorType, &e.Action,
			&e.ResourceType, &resourceID, &scopeType, &scopeID,
			&e.Detail, &e.Severity, &ipAddr, &e.UserAgent,
			&total,
		); err != nil {
			return nil, err
		}
		e.ResourceID = nullableToUUID(resourceID)
		e.ScopeID = nullableToUUID(scopeID)
		if scopeType != nil {
			e.ScopeType = *scopeType
		}
		if ipAddr != nil {
			e.IPAddress = *ipAddr
		}
		result.TotalCount = total
		result.Items = append(result.Items, &e)
	}
	return result, rows.Err()
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
