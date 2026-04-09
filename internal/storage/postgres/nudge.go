package postgres

import (
	"context"
	"errors"
	"time"

	"fresnel/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NudgeStore struct {
	pool *pgxpool.Pool
}

func NewNudgeStore(pool *pgxpool.Pool) *NudgeStore {
	return &NudgeStore{pool: pool}
}

func (s *NudgeStore) LogNudge(ctx context.Context, eventID, recipientID uuid.UUID, nudgeType string, level int) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO fresnel.nudge_log (id, event_id, recipient_id, nudge_type, escalation_level)
VALUES (gen_random_uuid(), $1, $2, $3, $4)`,
		eventID, recipientID, nudgeType, level,
	)
	return err
}

func (s *NudgeStore) HasNudgeToday(ctx context.Context, eventID, recipientID uuid.UUID) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1 FROM fresnel.nudge_log
    WHERE event_id = $1 AND recipient_id = $2
      AND sent_at >= CURRENT_DATE
)`, eventID, recipientID).Scan(&exists)
	return exists, err
}

func (s *NudgeStore) LastNudgeSentAt(ctx context.Context, eventID, recipientID uuid.UUID) (time.Time, bool, error) {
	var t time.Time
	err := s.pool.QueryRow(ctx, `
SELECT sent_at FROM fresnel.nudge_log
WHERE event_id = $1 AND recipient_id = $2
ORDER BY sent_at DESC
LIMIT 1`, eventID, recipientID).Scan(&t)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, false, nil
		}
		return time.Time{}, false, err
	}
	return t, true, nil
}

func (s *NudgeStore) LastEscalationNudgeTime(ctx context.Context, eventID uuid.UUID) (time.Time, bool, error) {
	var t time.Time
	err := s.pool.QueryRow(ctx, `
SELECT sent_at FROM fresnel.nudge_log
WHERE event_id = $1 AND nudge_type = 'ESCALATION'
ORDER BY sent_at DESC
LIMIT 1`, eventID).Scan(&t)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, false, nil
		}
		return time.Time{}, false, err
	}
	return t, true, nil
}

func (s *NudgeStore) GetEscalationState(ctx context.Context, eventID uuid.UUID) (int, *domain.AuditEntry, error) {
	var level int
	var lastResponseAt pgtype.Timestamptz
	err := s.pool.QueryRow(ctx, `
SELECT current_level, last_response_at FROM fresnel.escalation_state WHERE event_id = $1`, eventID,
	).Scan(&level, &lastResponseAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil, nil
		}
		return 0, nil, err
	}

	if !lastResponseAt.Valid {
		return level, nil, nil
	}

	var entry domain.AuditEntry
	var resourceID, scopeID pgtype.UUID
	var scopeType, ipAddr *string
	err = s.pool.QueryRow(ctx, `
SELECT id, timestamp, actor_id, actor_type, action, resource_type, resource_id,
       scope_type, scope_id, detail, severity, ip_address, user_agent
FROM fresnel_audit.audit_entries
WHERE resource_type = 'event' AND resource_id = $1
  AND timestamp <= $2
ORDER BY timestamp DESC
LIMIT 1`, eventID, lastResponseAt.Time,
	).Scan(&entry.ID, &entry.Timestamp, &entry.ActorID, &entry.ActorType, &entry.Action,
		&entry.ResourceType, &resourceID, &scopeType, &scopeID,
		&entry.Detail, &entry.Severity, &ipAddr, &entry.UserAgent)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return level, nil, nil
		}
		return 0, nil, err
	}
	entry.ResourceID = nullableToUUID(resourceID)
	entry.ScopeID = nullableToUUID(scopeID)
	if scopeType != nil {
		entry.ScopeType = *scopeType
	}
	if ipAddr != nil {
		entry.IPAddress = *ipAddr
	}

	return level, &entry, nil
}

func (s *NudgeStore) SetEscalationLevel(ctx context.Context, eventID uuid.UUID, level int) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO fresnel.escalation_state (event_id, current_level, escalated_at)
VALUES ($1, $2, $3)
ON CONFLICT (event_id) DO UPDATE SET current_level = $2, escalated_at = $3`,
		eventID, level, time.Now(),
	)
	return err
}

func (s *NudgeStore) ResetEscalation(ctx context.Context, eventID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM fresnel.escalation_state WHERE event_id = $1`, eventID)
	return err
}
