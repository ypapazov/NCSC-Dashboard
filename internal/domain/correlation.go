package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Correlation struct {
	ID              uuid.UUID       `json:"id"`
	EventAID        uuid.UUID       `json:"event_a_id"`
	EventBID        uuid.UUID       `json:"event_b_id"`
	Label           string          `json:"label"`
	CorrelationType CorrelationType `json:"correlation_type"`
	CreatedByUser   *uuid.UUID      `json:"created_by_user,omitempty"`
	CreatedByAgent  string          `json:"created_by_agent,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

func (c *Correlation) Validate() error {
	if c.EventAID == uuid.Nil || c.EventBID == uuid.Nil {
		return fmt.Errorf("both event IDs are required")
	}
	if c.EventAID == c.EventBID {
		return fmt.Errorf("cannot correlate an event with itself")
	}
	if c.Label == "" {
		return fmt.Errorf("label is required")
	}
	if len(c.Label) > 255 {
		return fmt.Errorf("label too long (max 255)")
	}
	return nil
}

// Normalize ensures canonical ordering (event_a_id < event_b_id).
func (c *Correlation) Normalize() {
	if c.EventAID.String() > c.EventBID.String() {
		c.EventAID, c.EventBID = c.EventBID, c.EventAID
	}
}

type EventRelationship struct {
	ID             uuid.UUID  `json:"id"`
	SourceEventID  uuid.UUID  `json:"source_event_id"`
	TargetEventID  uuid.UUID  `json:"target_event_id"`
	Label          string     `json:"label"`
	CreatedByUser  *uuid.UUID `json:"created_by_user,omitempty"`
	CreatedByAgent string     `json:"created_by_agent,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

func (r *EventRelationship) Validate() error {
	if r.SourceEventID == uuid.Nil || r.TargetEventID == uuid.Nil {
		return fmt.Errorf("both event IDs are required")
	}
	if r.SourceEventID == r.TargetEventID {
		return fmt.Errorf("cannot relate an event to itself")
	}
	if r.Label == "" {
		return fmt.Errorf("label is required")
	}
	return nil
}
