package service

import (
	"context"
	"fmt"
	"time"

	"fresnel/internal/authz"
	"fresnel/internal/domain"
	"fresnel/internal/storage"

	"github.com/google/uuid"
)

type StatusReportService struct {
	reports storage.StatusReportStore
	sectors storage.SectorStore
	tlpRed  storage.TLPRedStore
	authz   authz.Authorizer
	audit   *AuditService
}

func NewStatusReportService(reports storage.StatusReportStore, sectors storage.SectorStore, tlpRed storage.TLPRedStore, az authz.Authorizer, audit *AuditService) *StatusReportService {
	return &StatusReportService{reports: reports, sectors: sectors, tlpRed: tlpRed, authz: az, audit: audit}
}

func (s *StatusReportService) Create(ctx context.Context, auth *domain.AuthContext, sr *domain.StatusReport, eventIDs []uuid.UUID) error {
	sector, err := s.sectors.GetByID(ctx, sr.SectorContext)
	if err != nil || sector == nil {
		return fmt.Errorf("%w: invalid sector_context", ErrValidation)
	}

	sr.AuthorID = auth.UserID
	if sr.OrganizationID == uuid.Nil {
		sr.OrganizationID = auth.ActiveOrgContext
	}
	sr.SourceInstance = "local"

	if err := sr.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}

	res := authz.StatusReportResource(sr, sector.AncestryPath, nil)
	if !s.authz.Authorize(ctx, auth, authz.ActionCreate, res) {
		return ErrForbidden
	}

	sr.ID = uuid.New()
	now := time.Now().UTC()
	sr.CreatedAt = now
	sr.UpdatedAt = now
	sr.PublishedAt = now

	if err := s.reports.Create(ctx, sr); err != nil {
		return err
	}
	if len(eventIDs) > 0 {
		if err := s.reports.LinkEvents(ctx, sr.ID, eventIDs); err != nil {
			return err
		}
	}

	s.audit.Log(ctx, auth, "create", "status_report", &sr.ID, domain.SeverityInfo, map[string]any{
		"title": sr.Title, "assessed_status": sr.AssessedStatus, "scope_type": sr.ScopeType,
	})
	return nil
}

func (s *StatusReportService) GetByID(ctx context.Context, auth *domain.AuthContext, id uuid.UUID) (*domain.StatusReport, error) {
	sr, err := s.reports.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if sr == nil {
		return nil, ErrNotFound
	}
	sector, _ := s.sectors.GetByID(ctx, sr.SectorContext)
	ancestry := ""
	if sector != nil {
		ancestry = sector.AncestryPath
	}
	recipients, _ := s.tlpRed.GetRecipients(ctx, "status_report", sr.ID)
	res := authz.StatusReportResource(sr, ancestry, recipients)
	if !s.authz.Authorize(ctx, auth, authz.ActionView, res) {
		return nil, ErrForbidden
	}
	return sr, nil
}

func (s *StatusReportService) List(ctx context.Context, auth *domain.AuthContext, filter domain.StatusReportFilter) (*domain.ListResult[*domain.StatusReport], error) {
	filter.Normalize()
	result, err := s.reports.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	filtered := authz.FilterAuthorized(ctx, s.authz, auth, authz.ActionView, result.Items, func(sr *domain.StatusReport) *authz.Resource {
		sec, _ := s.sectors.GetByID(ctx, sr.SectorContext)
		ancestry := ""
		if sec != nil {
			ancestry = sec.AncestryPath
		}
		recipients, _ := s.tlpRed.GetRecipients(ctx, "status_report", sr.ID)
		return authz.StatusReportResource(sr, ancestry, recipients)
	})
	return &domain.ListResult[*domain.StatusReport]{Items: filtered, TotalCount: result.TotalCount}, nil
}

func (s *StatusReportService) Update(ctx context.Context, auth *domain.AuthContext, sr *domain.StatusReport) error {
	existing, err := s.reports.GetByID(ctx, sr.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrNotFound
	}
	sector, _ := s.sectors.GetByID(ctx, existing.SectorContext)
	ancestry := ""
	if sector != nil {
		ancestry = sector.AncestryPath
	}
	recipients, _ := s.tlpRed.GetRecipients(ctx, "status_report", existing.ID)
	res := authz.StatusReportResource(existing, ancestry, recipients)
	if !s.authz.Authorize(ctx, auth, authz.ActionEdit, res) {
		return ErrForbidden
	}

	existing.Title = sr.Title
	existing.Body = sr.Body
	existing.AssessedStatus = sr.AssessedStatus
	existing.Impact = sr.Impact
	if sr.TLP.Restrictiveness() < existing.TLP.Restrictiveness() {
		return fmt.Errorf("%w: TLP cannot become less restrictive", ErrValidation)
	}
	existing.TLP = sr.TLP
	existing.UpdatedAt = time.Now().UTC()

	if err := existing.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}
	if err := s.reports.Update(ctx, existing, auth.UserID); err != nil {
		return err
	}
	s.audit.Log(ctx, auth, "update", "status_report", &existing.ID, domain.SeverityInfo, map[string]any{
		"title": existing.Title, "assessed_status": existing.AssessedStatus,
	})
	return nil
}

func (s *StatusReportService) Delete(ctx context.Context, auth *domain.AuthContext, id uuid.UUID) error {
	existing, err := s.reports.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrNotFound
	}
	sector, _ := s.sectors.GetByID(ctx, existing.SectorContext)
	ancestry := ""
	if sector != nil {
		ancestry = sector.AncestryPath
	}
	res := authz.StatusReportResource(existing, ancestry, nil)
	if !s.authz.Authorize(ctx, auth, authz.ActionDelete, res) {
		return ErrForbidden
	}
	if err := s.reports.Delete(ctx, id); err != nil {
		return err
	}
	s.audit.Log(ctx, auth, "delete", "status_report", &id, domain.SeverityHigh, map[string]any{"title": existing.Title})
	return nil
}

func (s *StatusReportService) GetRevisions(ctx context.Context, auth *domain.AuthContext, reportID uuid.UUID) ([]*domain.StatusReportRevision, error) {
	if _, err := s.GetByID(ctx, auth, reportID); err != nil {
		return nil, err
	}
	return s.reports.GetRevisions(ctx, reportID)
}

func (s *StatusReportService) GetLinkedEventIDs(ctx context.Context, auth *domain.AuthContext, reportID uuid.UUID) ([]uuid.UUID, error) {
	if _, err := s.GetByID(ctx, auth, reportID); err != nil {
		return nil, err
	}
	return s.reports.GetLinkedEventIDs(ctx, reportID)
}
