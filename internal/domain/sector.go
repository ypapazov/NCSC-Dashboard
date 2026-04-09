package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Sector struct {
	ID             uuid.UUID  `json:"id"`
	ParentSectorID *uuid.UUID `json:"parent_sector_id,omitempty"`
	Name           string     `json:"name"`
	AncestryPath   string     `json:"ancestry_path"`
	Depth          int        `json:"depth"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
}

func (s *Sector) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("sector name is required")
	}
	if len(s.Name) > 255 {
		return fmt.Errorf("sector name too long (max 255)")
	}
	if s.Depth > 5 {
		return fmt.Errorf("sector depth exceeds maximum (5)")
	}
	return nil
}
