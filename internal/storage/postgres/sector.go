package postgres

import (
	"context"
	"errors"

	"fresnel/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SectorStore struct {
	pool *pgxpool.Pool
}

func NewSectorStore(pool *pgxpool.Pool) *SectorStore {
	return &SectorStore{pool: pool}
}

func (s *SectorStore) Create(ctx context.Context, sec *domain.Sector) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO fresnel.sectors (id, parent_sector_id, name, ancestry_path, depth, status, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		sec.ID, uuidToNullable(sec.ParentSectorID), sec.Name, sec.AncestryPath, sec.Depth, sec.Status, sec.CreatedAt,
	)
	return err
}

func (s *SectorStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.Sector, error) {
	row := s.pool.QueryRow(ctx, `
SELECT id, parent_sector_id, name, ancestry_path, depth, status, created_at
FROM fresnel.sectors WHERE id = $1`, id)
	return scanSector(row)
}

func (s *SectorStore) List(ctx context.Context) ([]*domain.Sector, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, parent_sector_id, name, ancestry_path, depth, status, created_at
FROM fresnel.sectors ORDER BY ancestry_path`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectSectors(rows)
}

func (s *SectorStore) GetChildren(ctx context.Context, parentID uuid.UUID) ([]*domain.Sector, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, parent_sector_id, name, ancestry_path, depth, status, created_at
FROM fresnel.sectors WHERE parent_sector_id = $1 ORDER BY name`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectSectors(rows)
}

func (s *SectorStore) GetDescendants(ctx context.Context, ancestryPrefix string) ([]*domain.Sector, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, parent_sector_id, name, ancestry_path, depth, status, created_at
FROM fresnel.sectors WHERE ancestry_path LIKE $1 || '%' ORDER BY ancestry_path`, ancestryPrefix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectSectors(rows)
}

func (s *SectorStore) Update(ctx context.Context, sec *domain.Sector) error {
	ct, err := s.pool.Exec(ctx, `
UPDATE fresnel.sectors SET parent_sector_id = $2, name = $3, ancestry_path = $4, depth = $5, status = $6
WHERE id = $1`,
		sec.ID, uuidToNullable(sec.ParentSectorID), sec.Name, sec.AncestryPath, sec.Depth, sec.Status,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *SectorStore) Delete(ctx context.Context, id uuid.UUID) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM fresnel.sectors WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func scanSector(row pgx.Row) (*domain.Sector, error) {
	var sec domain.Sector
	var parentID pgtype.UUID
	err := row.Scan(&sec.ID, &parentID, &sec.Name, &sec.AncestryPath, &sec.Depth, &sec.Status, &sec.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}
	sec.ParentSectorID = nullableToUUID(parentID)
	return &sec, nil
}

func collectSectors(rows pgx.Rows) ([]*domain.Sector, error) {
	var out []*domain.Sector
	for rows.Next() {
		var sec domain.Sector
		var parentID pgtype.UUID
		if err := rows.Scan(&sec.ID, &parentID, &sec.Name, &sec.AncestryPath, &sec.Depth, &sec.Status, &sec.CreatedAt); err != nil {
			return nil, err
		}
		sec.ParentSectorID = nullableToUUID(parentID)
		out = append(out, &sec)
	}
	return out, rows.Err()
}

func uuidToNullable(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

func nullableToUUID(pg pgtype.UUID) *uuid.UUID {
	if !pg.Valid {
		return nil
	}
	id := uuid.UUID(pg.Bytes)
	return &id
}
