package postgres

import (
	"context"
	"fmt"

	"fresnel/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserStore struct {
	pool *pgxpool.Pool
}

func NewUserStore(pool *pgxpool.Pool) *UserStore {
	return &UserStore{pool: pool}
}

func (s *UserStore) Create(ctx context.Context, u *domain.User) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO fresnel.users (id, keycloak_sub, display_name, email, primary_org_id, timezone, status, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		u.ID, u.KeycloakSub, u.DisplayName, u.Email, u.PrimaryOrgID, u.Timezone, u.Status, u.CreatedAt,
	)
	return err
}

func (s *UserStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var u domain.User
	err := s.pool.QueryRow(ctx, `
SELECT id, keycloak_sub, display_name, email, primary_org_id, timezone, status, created_at
FROM fresnel.users WHERE id = $1`, id,
	).Scan(&u.ID, &u.KeycloakSub, &u.DisplayName, &u.Email, &u.PrimaryOrgID, &u.Timezone, &u.Status, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *UserStore) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var u domain.User
	err := s.pool.QueryRow(ctx, `
SELECT id, keycloak_sub, display_name, email, primary_org_id, timezone, status, created_at
FROM fresnel.users WHERE lower(email) = lower($1)`, email,
	).Scan(&u.ID, &u.KeycloakSub, &u.DisplayName, &u.Email, &u.PrimaryOrgID, &u.Timezone, &u.Status, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *UserStore) List(ctx context.Context, orgID *uuid.UUID, p domain.Pagination) (*domain.ListResult[*domain.User], error) {
	p.Normalize()
	var (
		rows pgx.Rows
		err  error
	)
	if orgID != nil {
		rows, err = s.pool.Query(ctx, `
SELECT u.id, u.keycloak_sub, u.display_name, u.email, u.primary_org_id, u.timezone, u.status, u.created_at,
       COUNT(*) OVER() AS total
FROM fresnel.users u
JOIN fresnel.user_org_memberships m ON m.user_id = u.id
WHERE m.organization_id = $1
ORDER BY u.display_name
LIMIT $2 OFFSET $3`, *orgID, p.Limit, p.Offset)
	} else {
		rows, err = s.pool.Query(ctx, `
SELECT id, keycloak_sub, display_name, email, primary_org_id, timezone, status, created_at,
       COUNT(*) OVER() AS total
FROM fresnel.users
ORDER BY display_name
LIMIT $1 OFFSET $2`, p.Limit, p.Offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := &domain.ListResult[*domain.User]{}
	for rows.Next() {
		var u domain.User
		var total int
		if err := rows.Scan(&u.ID, &u.KeycloakSub, &u.DisplayName, &u.Email, &u.PrimaryOrgID, &u.Timezone, &u.Status, &u.CreatedAt, &total); err != nil {
			return nil, err
		}
		result.TotalCount = total
		result.Items = append(result.Items, &u)
	}
	return result, rows.Err()
}

func (s *UserStore) Update(ctx context.Context, u *domain.User) error {
	ct, err := s.pool.Exec(ctx, `
UPDATE fresnel.users SET keycloak_sub = $2, display_name = $3, email = $4, primary_org_id = $5, timezone = $6, status = $7
WHERE id = $1`,
		u.ID, u.KeycloakSub, u.DisplayName, u.Email, u.PrimaryOrgID, u.Timezone, u.Status,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *UserStore) Delete(ctx context.Context, id uuid.UUID) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM fresnel.users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *UserStore) AddOrgMembership(ctx context.Context, userID, orgID, assignedBy uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO fresnel.user_org_memberships (user_id, organization_id, assigned_by)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, organization_id) DO NOTHING`, userID, orgID, assignedBy)
	return err
}

func (s *UserStore) RemoveOrgMembership(ctx context.Context, userID, orgID uuid.UUID) error {
	ct, err := s.pool.Exec(ctx, `
DELETE FROM fresnel.user_org_memberships WHERE user_id = $1 AND organization_id = $2`, userID, orgID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("membership not found")
	}
	return nil
}

func (s *UserStore) GetOrgMemberships(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := s.pool.Query(ctx, `
SELECT organization_id FROM fresnel.user_org_memberships WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []uuid.UUID
	for rows.Next() {
		var oid uuid.UUID
		if err := rows.Scan(&oid); err != nil {
			return nil, err
		}
		orgs = append(orgs, oid)
	}
	return orgs, rows.Err()
}
