package postgres

import (
	"context"

	"fresnel/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AttachmentStore struct {
	pool *pgxpool.Pool
}

func NewAttachmentStore(pool *pgxpool.Pool) *AttachmentStore {
	return &AttachmentStore{pool: pool}
}

func (s *AttachmentStore) Create(ctx context.Context, a *domain.Attachment) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO fresnel.attachments (id, event_id, filename, content_type, size_bytes, storage_path, scan_status, uploaded_by, uploaded_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		a.ID, a.EventID, a.Filename, a.ContentType, a.SizeBytes, a.StoragePath,
		string(a.ScanStatus), a.UploadedBy, a.UploadedAt,
	)
	return err
}

func (s *AttachmentStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.Attachment, error) {
	var a domain.Attachment
	err := s.pool.QueryRow(ctx, `
SELECT id, event_id, filename, content_type, size_bytes, storage_path, scan_status, uploaded_by, uploaded_at
FROM fresnel.attachments WHERE id = $1`, id,
	).Scan(&a.ID, &a.EventID, &a.Filename, &a.ContentType, &a.SizeBytes, &a.StoragePath,
		&a.ScanStatus, &a.UploadedBy, &a.UploadedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *AttachmentStore) ListByEvent(ctx context.Context, eventID uuid.UUID) ([]*domain.Attachment, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, event_id, filename, content_type, size_bytes, storage_path, scan_status, uploaded_by, uploaded_at
FROM fresnel.attachments WHERE event_id = $1 ORDER BY uploaded_at`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Attachment
	for rows.Next() {
		var a domain.Attachment
		if err := rows.Scan(&a.ID, &a.EventID, &a.Filename, &a.ContentType, &a.SizeBytes, &a.StoragePath,
			&a.ScanStatus, &a.UploadedBy, &a.UploadedAt); err != nil {
			return nil, err
		}
		out = append(out, &a)
	}
	return out, rows.Err()
}

func (s *AttachmentStore) UpdateScanStatus(ctx context.Context, id uuid.UUID, status domain.ScanStatus) error {
	ct, err := s.pool.Exec(ctx, `
UPDATE fresnel.attachments SET scan_status = $2 WHERE id = $1`, id, string(status))
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *AttachmentStore) Delete(ctx context.Context, id uuid.UUID) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM fresnel.attachments WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *AttachmentStore) CountByEvent(ctx context.Context, eventID uuid.UUID) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM fresnel.attachments WHERE event_id = $1`, eventID).Scan(&count)
	return count, err
}
