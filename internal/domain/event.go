package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID                uuid.UUID   `json:"id"`
	SourceInstance    string      `json:"source_instance"`
	SectorContext     uuid.UUID   `json:"sector_context"`
	Title             string      `json:"title"`
	Description       string      `json:"description"`
	EventType         EventType   `json:"event_type"`
	SubmitterID       uuid.UUID   `json:"submitter_id"`
	OrganizationID    uuid.UUID   `json:"organization_id"`
	TLP               TLP         `json:"tlp"`
	Impact            Impact      `json:"impact"`
	Status            EventStatus `json:"status"`
	IntelSource       string      `json:"intel_source"`
	Target            string      `json:"target"`
	OriginalEventDate *time.Time  `json:"original_event_date,omitempty"`
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
}

func (e *Event) Validate() error {
	if e.Title == "" {
		return fmt.Errorf("title is required")
	}
	if len(e.Title) > 500 {
		return fmt.Errorf("title too long (max 500)")
	}
	if e.Description == "" {
		return fmt.Errorf("description is required")
	}
	if len(e.Description) > 50000 {
		return fmt.Errorf("description too long (max 50000)")
	}
	if !e.EventType.Valid() {
		return fmt.Errorf("invalid event_type: %q", e.EventType)
	}
	if !e.TLP.Valid() {
		return fmt.Errorf("invalid tlp: %q", e.TLP)
	}
	if !e.Impact.Valid() {
		return fmt.Errorf("invalid impact: %q", e.Impact)
	}
	if !e.Status.Valid() {
		return fmt.Errorf("invalid status: %q", e.Status)
	}
	if e.SectorContext == uuid.Nil {
		return fmt.Errorf("sector_context is required")
	}
	if e.OrganizationID == uuid.Nil {
		return fmt.Errorf("organization_id is required")
	}
	return nil
}

type EventRevision struct {
	ID             uuid.UUID   `json:"id"`
	EventID        uuid.UUID   `json:"event_id"`
	RevisionNumber int         `json:"revision_number"`
	Title          string      `json:"title"`
	Description    string      `json:"description"`
	EventType      EventType   `json:"event_type"`
	TLP            TLP         `json:"tlp"`
	Impact         Impact      `json:"impact"`
	Status         EventStatus `json:"status"`
	ChangedBy      uuid.UUID   `json:"changed_by"`
	ChangedAt      time.Time   `json:"changed_at"`
}

type EventUpdate struct {
	ID           uuid.UUID    `json:"id"`
	EventID      uuid.UUID    `json:"event_id"`
	AuthorID     uuid.UUID    `json:"author_id"`
	Body         string       `json:"body"`
	TLP          TLP          `json:"tlp"`
	ImpactChange *Impact      `json:"impact_change,omitempty"`
	StatusChange *EventStatus `json:"status_change,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
}

func (u *EventUpdate) Validate() error {
	if u.Body == "" {
		return fmt.Errorf("body is required")
	}
	if len(u.Body) > 50000 {
		return fmt.Errorf("body too long (max 50000)")
	}
	if !u.TLP.Valid() {
		return fmt.Errorf("invalid tlp: %q", u.TLP)
	}
	if u.ImpactChange != nil && !u.ImpactChange.Valid() {
		return fmt.Errorf("invalid impact_change: %q", *u.ImpactChange)
	}
	if u.StatusChange != nil && !u.StatusChange.Valid() {
		return fmt.Errorf("invalid status_change: %q", *u.StatusChange)
	}
	return nil
}

type Attachment struct {
	ID          uuid.UUID  `json:"id"`
	EventID     uuid.UUID  `json:"event_id"`
	Filename    string     `json:"filename"`
	ContentType string     `json:"content_type"`
	SizeBytes   int64      `json:"size_bytes"`
	StoragePath string     `json:"-"`
	ScanStatus  ScanStatus `json:"scan_status"`
	UploadedBy  uuid.UUID  `json:"uploaded_by"`
	UploadedAt  time.Time  `json:"uploaded_at"`
}

type TLPRedRecipient struct {
	ID              uuid.UUID `json:"id"`
	ResourceType    string    `json:"resource_type"`
	ResourceID      uuid.UUID `json:"resource_id"`
	RecipientUserID uuid.UUID `json:"recipient_user_id"`
	GrantedBy       uuid.UUID `json:"granted_by"`
	GrantedAt       time.Time `json:"granted_at"`
}

type EventFilter struct {
	SectorContextID *uuid.UUID
	OrganizationID  *uuid.UUID
	Status          *EventStatus
	Impact          *Impact
	EventType       *EventType
	TLP             *TLP
	Search          string
	DateFrom        *time.Time
	DateTo          *time.Time
	SortBy          string // "created_at" (default) or "updated_at"
	Pagination
}
