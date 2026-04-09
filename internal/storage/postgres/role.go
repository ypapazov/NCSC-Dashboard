package postgres

import (
	"context"

	"fresnel/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RoleStore struct {
	pool *pgxpool.Pool
}

func NewRoleStore(pool *pgxpool.Pool) *RoleStore {
	return &RoleStore{pool: pool}
}

func (s *RoleStore) AssignRole(ctx context.Context, userID uuid.UUID, role domain.Role, scopeType domain.ScopeType, scopeID uuid.UUID, assignedBy uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO fresnel_iam.role_assignments (id, user_id, role, scope_type, scope_id, assigned_by)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5)
ON CONFLICT (user_id, role, scope_type, scope_id) DO NOTHING`,
		userID, string(role), string(scopeType), scopeID, assignedBy,
	)
	return err
}

func (s *RoleStore) RevokeRole(ctx context.Context, userID uuid.UUID, role domain.Role, scopeType domain.ScopeType, scopeID uuid.UUID) error {
	ct, err := s.pool.Exec(ctx, `
DELETE FROM fresnel_iam.role_assignments
WHERE user_id = $1 AND role = $2 AND scope_type = $3 AND scope_id = $4`,
		userID, string(role), string(scopeType), scopeID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *RoleStore) ListRoles(ctx context.Context, userID uuid.UUID) ([]domain.RoleAssignment, error) {
	rows, err := s.pool.Query(ctx, `
SELECT role, scope_type, scope_id
FROM fresnel_iam.role_assignments WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.RoleAssignment
	for rows.Next() {
		var ra domain.RoleAssignment
		if err := rows.Scan(&ra.Role, &ra.ScopeType, &ra.ScopeID); err != nil {
			return nil, err
		}
		out = append(out, ra)
	}
	return out, rows.Err()
}

func (s *RoleStore) ListRolesByScope(ctx context.Context, scopeType domain.ScopeType, scopeID uuid.UUID) ([]domain.RoleAssignment, error) {
	rows, err := s.pool.Query(ctx, `
SELECT role, scope_type, scope_id
FROM fresnel_iam.role_assignments WHERE scope_type = $1 AND scope_id = $2`,
		string(scopeType), scopeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.RoleAssignment
	for rows.Next() {
		var ra domain.RoleAssignment
		if err := rows.Scan(&ra.Role, &ra.ScopeType, &ra.ScopeID); err != nil {
			return nil, err
		}
		out = append(out, ra)
	}
	return out, rows.Err()
}

func (s *RoleStore) DesignateRoot(ctx context.Context, userID uuid.UUID, scopeType domain.ScopeType, scopeID *uuid.UUID, designatedBy uuid.UUID) error {
	pgScopeID := uuidToNullable(scopeID)
	_, err := s.pool.Exec(ctx, `
INSERT INTO fresnel_iam.root_designations (id, user_id, scope_type, scope_id, designated_by)
VALUES (gen_random_uuid(), $1, $2, $3, $4)
ON CONFLICT (scope_type, scope_id) DO UPDATE SET user_id = $1, designated_by = $4, designated_at = now()`,
		userID, string(scopeType), pgScopeID, designatedBy,
	)
	return err
}

func (s *RoleStore) RevokeRoot(ctx context.Context, scopeType domain.ScopeType, scopeID *uuid.UUID) error {
	var err error
	if scopeID == nil {
		_, err = s.pool.Exec(ctx, `
DELETE FROM fresnel_iam.root_designations WHERE scope_type = $1 AND scope_id IS NULL`, string(scopeType))
	} else {
		_, err = s.pool.Exec(ctx, `
DELETE FROM fresnel_iam.root_designations WHERE scope_type = $1 AND scope_id = $2`, string(scopeType), *scopeID)
	}
	return err
}

func (s *RoleStore) GetRoot(ctx context.Context, scopeType domain.ScopeType, scopeID *uuid.UUID) (*uuid.UUID, error) {
	var userID uuid.UUID
	var err error
	if scopeID == nil {
		err = s.pool.QueryRow(ctx, `
SELECT user_id FROM fresnel_iam.root_designations WHERE scope_type = $1 AND scope_id IS NULL`,
			string(scopeType)).Scan(&userID)
	} else {
		err = s.pool.QueryRow(ctx, `
SELECT user_id FROM fresnel_iam.root_designations WHERE scope_type = $1 AND scope_id = $2`,
			string(scopeType), *scopeID).Scan(&userID)
	}
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &userID, nil
}
