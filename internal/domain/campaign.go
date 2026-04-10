package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Campaign struct {
	ID             uuid.UUID      `json:"id"`
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	TLP            TLP            `json:"tlp"`
	Status         CampaignStatus `json:"status"`
	CreatedBy      uuid.UUID      `json:"created_by"`
	OrganizationID uuid.UUID      `json:"organization_id"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`

	// EventCount is populated by the handler layer, not persisted.
	EventCount int `json:"event_count,omitempty"`
}

func (c *Campaign) Validate() error {
	if c.Title == "" {
		return fmt.Errorf("title is required")
	}
	if len(c.Title) > 500 {
		return fmt.Errorf("title too long (max 500)")
	}
	if len(c.Description) > 50000 {
		return fmt.Errorf("description too long (max 50000)")
	}
	if !c.TLP.Valid() {
		return fmt.Errorf("invalid tlp: %q", c.TLP)
	}
	if c.OrganizationID == uuid.Nil {
		return fmt.Errorf("organization_id is required")
	}
	return nil
}

type CampaignEvent struct {
	CampaignID uuid.UUID `json:"campaign_id"`
	EventID    uuid.UUID `json:"event_id"`
	LinkedBy   uuid.UUID `json:"linked_by"`
	LinkedAt   time.Time `json:"linked_at"`
}

type CampaignFilter struct {
	OrganizationID *uuid.UUID
	Status         *CampaignStatus
	TLP            *TLP
	Pagination
}
