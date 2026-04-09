package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FormulaStore struct {
	pool *pgxpool.Pool
}

func NewFormulaStore(pool *pgxpool.Pool) *FormulaStore {
	return &FormulaStore{pool: pool}
}

func (s *FormulaStore) Get(ctx context.Context, nodeType string, nodeID *uuid.UUID) (string, error) {
	var source string
	var err error
	if nodeID == nil {
		err = s.pool.QueryRow(ctx, `
SELECT starlark_source FROM fresnel.status_formulas WHERE node_type = $1 AND node_id IS NULL`, nodeType).Scan(&source)
	} else {
		err = s.pool.QueryRow(ctx, `
SELECT starlark_source FROM fresnel.status_formulas WHERE node_type = $1 AND node_id = $2`, nodeType, *nodeID).Scan(&source)
	}
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return source, nil
}

func (s *FormulaStore) Set(ctx context.Context, nodeType string, nodeID *uuid.UUID, source string, setBy uuid.UUID) error {
	pgNodeID := uuidToNullable(nodeID)
	_, err := s.pool.Exec(ctx, `
INSERT INTO fresnel.status_formulas (id, node_type, node_id, starlark_source, set_by)
VALUES (gen_random_uuid(), $1, $2, $3, $4)
ON CONFLICT (node_type, node_id) DO UPDATE SET starlark_source = $3, set_by = $4, set_at = now()`,
		nodeType, pgNodeID, source, setBy,
	)
	return err
}
