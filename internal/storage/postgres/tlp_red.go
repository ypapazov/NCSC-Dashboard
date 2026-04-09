package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TLPRedStore struct {
	pool *pgxpool.Pool
}

func NewTLPRedStore(pool *pgxpool.Pool) *TLPRedStore {
	return &TLPRedStore{pool: pool}
}

func (s *TLPRedStore) SetRecipients(ctx context.Context, resourceType string, resourceID uuid.UUID, recipientIDs []uuid.UUID, grantedBy uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
DELETE FROM fresnel.tlp_red_recipients WHERE resource_type = $1 AND resource_id = $2`,
		resourceType, resourceID)
	if err != nil {
		return err
	}

	for _, rid := range recipientIDs {
		_, err = tx.Exec(ctx, `
INSERT INTO fresnel.tlp_red_recipients (id, resource_type, resource_id, recipient_user_id, granted_by)
VALUES (gen_random_uuid(), $1, $2, $3, $4)`,
			resourceType, resourceID, rid, grantedBy)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (s *TLPRedStore) GetRecipients(ctx context.Context, resourceType string, resourceID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := s.pool.Query(ctx, `
SELECT recipient_user_id FROM fresnel.tlp_red_recipients
WHERE resource_type = $1 AND resource_id = $2`, resourceType, resourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []uuid.UUID
	for rows.Next() {
		var uid uuid.UUID
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		out = append(out, uid)
	}
	return out, rows.Err()
}
