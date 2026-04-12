package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type StatusReport struct {
	ID                 uuid.UUID      `json:"id"`
	SourceInstance     string         `json:"source_instance"`
	SectorContext      uuid.UUID      `json:"sector_context"`
	ScopeType          string         `json:"scope_type"`
	ScopeRef           uuid.UUID      `json:"scope_ref"`
	Title              string         `json:"title"`
	Body               string         `json:"body"`
	PeriodCoveredStart time.Time      `json:"period_covered_start"`
	PeriodCoveredEnd   time.Time      `json:"period_covered_end"`
	AsOf               time.Time      `json:"as_of"`
	PublishedAt        time.Time      `json:"published_at"`
	AssessedStatus     AssessedStatus `json:"assessed_status"`
	Impact             Impact         `json:"impact"`
	TLP                TLP            `json:"tlp"`
	AuthorID           uuid.UUID      `json:"author_id"`
	OrganizationID     uuid.UUID      `json:"organization_id"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

func (sr *StatusReport) Validate() error {
	if sr.Title == "" {
		return fmt.Errorf("title is required")
	}
	if len(sr.Title) > 500 {
		return fmt.Errorf("title too long (max 500)")
	}
	if sr.Body == "" {
		return fmt.Errorf("body is required")
	}
	if len(sr.Body) > 50000 {
		return fmt.Errorf("body too long (max 50000)")
	}
	if !sr.AssessedStatus.Valid() {
		return fmt.Errorf("invalid assessed_status: %q", sr.AssessedStatus)
	}
	if !sr.Impact.Valid() {
		return fmt.Errorf("invalid impact: %q", sr.Impact)
	}
	if !sr.TLP.Valid() {
		return fmt.Errorf("invalid tlp: %q", sr.TLP)
	}
	if sr.ScopeType != "ORG" && sr.ScopeType != "SECTOR" {
		return fmt.Errorf("scope_type must be ORG or SECTOR")
	}
	if sr.PeriodCoveredEnd.Before(sr.PeriodCoveredStart) {
		return fmt.Errorf("period_covered_end must be after period_covered_start")
	}
	return nil
}

type StatusReportRevision struct {
	ID             uuid.UUID      `json:"id"`
	StatusReportID uuid.UUID      `json:"status_report_id"`
	RevisionNumber int            `json:"revision_number"`
	Title          string         `json:"title"`
	Body           string         `json:"body"`
	AssessedStatus AssessedStatus `json:"assessed_status"`
	Impact         Impact         `json:"impact"`
	TLP            TLP            `json:"tlp"`
	ChangedBy      uuid.UUID      `json:"changed_by"`
	ChangedAt      time.Time      `json:"changed_at"`
}

type StatusReportFilter struct {
	SectorContextID     *uuid.UUID
	SectorAncestryPrefix string // ancestry path prefix for recursive sector queries
	OrganizationID      *uuid.UUID
	ScopeType           string
	ScopeRef            *uuid.UUID
	AssessedStatus      *AssessedStatus
	DateFrom            *time.Time
	DateTo              *time.Time
	Pagination
}
