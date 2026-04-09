package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Organization struct {
	ID        uuid.UUID `json:"id"`
	SectorID  uuid.UUID `json:"sector_id"`
	Name      string    `json:"name"`
	Timezone  string    `json:"timezone"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func (o *Organization) Validate() error {
	if o.Name == "" {
		return fmt.Errorf("organization name is required")
	}
	if len(o.Name) > 255 {
		return fmt.Errorf("organization name too long (max 255)")
	}
	if o.SectorID == uuid.Nil {
		return fmt.Errorf("sector_id is required")
	}
	return nil
}

type OrgSectorMembership struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	SectorID       uuid.UUID  `json:"sector_id"`
	RootUserID     *uuid.UUID `json:"root_user_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}
