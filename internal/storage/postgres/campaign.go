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

type CampaignStore struct {
	pool *pgxpool.Pool
}

func NewCampaignStore(pool *pgxpool.Pool) *CampaignStore {
	return &CampaignStore{pool: pool}
}

func (s *CampaignStore) Create(ctx context.Context, c *domain.Campaign) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO fresnel.campaigns (id, title, description, tlp, status, created_by, organization_id, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		c.ID, c.Title, c.Description, string(c.TLP), string(c.Status),
		c.CreatedBy, c.OrganizationID, c.CreatedAt, c.UpdatedAt,
	)
	return err
}

func (s *CampaignStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.Campaign, error) {
	var c domain.Campaign
	err := s.pool.QueryRow(ctx, `
SELECT id, title, description, tlp, status, created_by, organization_id, created_at, updated_at
FROM fresnel.campaigns WHERE id = $1`, id,
	).Scan(&c.ID, &c.Title, &c.Description, &c.TLP, &c.Status,
		&c.CreatedBy, &c.OrganizationID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *CampaignStore) List(ctx context.Context, f domain.CampaignFilter) (*domain.ListResult[*domain.Campaign], error) {
	f.Normalize()

	var conditions []string
	var args []any
	idx := 1

	if f.OrganizationID != nil {
		conditions = append(conditions, fmt.Sprintf("organization_id = $%d", idx))
		args = append(args, *f.OrganizationID)
		idx++
	}
	if f.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", idx))
		args = append(args, string(*f.Status))
		idx++
	}
	if f.TLP != nil {
		conditions = append(conditions, fmt.Sprintf("tlp = $%d", idx))
		args = append(args, string(*f.TLP))
		idx++
	}
	if f.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(title ILIKE '%%' || $%d || '%%' OR description ILIKE '%%' || $%d || '%%')", idx, idx))
		args = append(args, f.Search)
		idx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	q := fmt.Sprintf(`
SELECT id, title, description, tlp, status, created_by, organization_id, created_at, updated_at,
       COUNT(*) OVER() AS total
FROM fresnel.campaigns
%s
ORDER BY created_at DESC
LIMIT $%d OFFSET $%d`, where, idx, idx+1)
	args = append(args, f.Limit, f.Offset)

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := &domain.ListResult[*domain.Campaign]{}
	for rows.Next() {
		var c domain.Campaign
		var total int
		if err := rows.Scan(&c.ID, &c.Title, &c.Description, &c.TLP, &c.Status,
			&c.CreatedBy, &c.OrganizationID, &c.CreatedAt, &c.UpdatedAt, &total); err != nil {
			return nil, err
		}
		result.TotalCount = total
		result.Items = append(result.Items, &c)
	}
	return result, rows.Err()
}

func (s *CampaignStore) Update(ctx context.Context, c *domain.Campaign) error {
	ct, err := s.pool.Exec(ctx, `
UPDATE fresnel.campaigns SET title = $2, description = $3, tlp = $4, status = $5, updated_at = now()
WHERE id = $1`,
		c.ID, c.Title, c.Description, string(c.TLP), string(c.Status),
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *CampaignStore) Delete(ctx context.Context, id uuid.UUID) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM fresnel.campaigns WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *CampaignStore) LinkEvent(ctx context.Context, campaignID, eventID, linkedBy uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO fresnel.campaign_events (campaign_id, event_id, linked_by)
VALUES ($1, $2, $3)
ON CONFLICT (campaign_id, event_id) DO NOTHING`, campaignID, eventID, linkedBy)
	return err
}

func (s *CampaignStore) UnlinkEvent(ctx context.Context, campaignID, eventID uuid.UUID) error {
	ct, err := s.pool.Exec(ctx, `
DELETE FROM fresnel.campaign_events WHERE campaign_id = $1 AND event_id = $2`, campaignID, eventID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *CampaignStore) GetLinkedEventIDs(ctx context.Context, campaignID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := s.pool.Query(ctx, `
SELECT event_id FROM fresnel.campaign_events WHERE campaign_id = $1`, campaignID)
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
