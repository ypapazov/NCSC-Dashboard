package postgres

import (
	"context"

	"fresnel/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrganizationStore struct {
	pool *pgxpool.Pool
}

func NewOrganizationStore(pool *pgxpool.Pool) *OrganizationStore {
	return &OrganizationStore{pool: pool}
}

func (s *OrganizationStore) Create(ctx context.Context, o *domain.Organization) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO fresnel.organizations (id, sector_id, name, timezone, status, created_at)
VALUES ($1, $2, $3, $4, $5, $6)`,
		o.ID, o.SectorID, o.Name, o.Timezone, o.Status, o.CreatedAt,
	)
	return err
}

func (s *OrganizationStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	var o domain.Organization
	err := s.pool.QueryRow(ctx, `
SELECT id, sector_id, name, timezone, status, created_at
FROM fresnel.organizations WHERE id = $1`, id,
	).Scan(&o.ID, &o.SectorID, &o.Name, &o.Timezone, &o.Status, &o.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (s *OrganizationStore) List(ctx context.Context, sectorID *uuid.UUID) ([]*domain.Organization, error) {
	var (
		rows pgx.Rows
		err  error
	)
	if sectorID != nil {
		rows, err = s.pool.Query(ctx, `
SELECT id, sector_id, name, timezone, status, created_at
FROM fresnel.organizations WHERE sector_id = $1 ORDER BY name`, *sectorID)
	} else {
		rows, err = s.pool.Query(ctx, `
SELECT id, sector_id, name, timezone, status, created_at
FROM fresnel.organizations ORDER BY name`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Organization
	for rows.Next() {
		var o domain.Organization
		if err := rows.Scan(&o.ID, &o.SectorID, &o.Name, &o.Timezone, &o.Status, &o.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &o)
	}
	return out, rows.Err()
}

func (s *OrganizationStore) Update(ctx context.Context, o *domain.Organization) error {
	ct, err := s.pool.Exec(ctx, `
UPDATE fresnel.organizations SET sector_id = $2, name = $3, timezone = $4, status = $5
WHERE id = $1`,
		o.ID, o.SectorID, o.Name, o.Timezone, o.Status,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *OrganizationStore) Delete(ctx context.Context, id uuid.UUID) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM fresnel.organizations WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *OrganizationStore) ListBySectorAncestry(ctx context.Context, ancestryPrefix string) ([]*domain.Organization, error) {
	rows, err := s.pool.Query(ctx, `
SELECT o.id, o.sector_id, o.name, o.timezone, o.status, o.created_at
FROM fresnel.organizations o
JOIN fresnel.sectors s ON s.id = o.sector_id
WHERE s.ancestry_path LIKE $1 || '%'
ORDER BY o.name`, ancestryPrefix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Organization
	for rows.Next() {
		var o domain.Organization
		if err := rows.Scan(&o.ID, &o.SectorID, &o.Name, &o.Timezone, &o.Status, &o.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &o)
	}
	return out, rows.Err()
}

