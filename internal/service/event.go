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

type EventService struct {
	events          storage.EventStore
	updates         storage.EventUpdateStore
	sectors         storage.SectorStore
	tlpRed          storage.TLPRedStore
	authz           authz.Authorizer
	audit           *AuditService
	escalationReset EscalationResetter
}

func NewEventService(events storage.EventStore, updates storage.EventUpdateStore, sectors storage.SectorStore, tlpRed storage.TLPRedStore, az authz.Authorizer, audit *AuditService, escalationReset EscalationResetter) *EventService {
	if escalationReset == nil {
		escalationReset = noopEscalationReset{}
	}
	return &EventService{events: events, updates: updates, sectors: sectors, tlpRed: tlpRed, authz: az, audit: audit, escalationReset: escalationReset}
}

func (s *EventService) Create(ctx context.Context, auth *domain.AuthContext, event *domain.Event, tlpRedRecipients []uuid.UUID) error {
	sector, err := s.sectors.GetByID(ctx, event.SectorContext)
	if err != nil || sector == nil {
		return fmt.Errorf("%w: invalid sector_context", ErrValidation)
	}

	event.SubmitterID = auth.UserID
	if event.OrganizationID == uuid.Nil {
		event.OrganizationID = auth.ActiveOrgContext
	}
	event.Status = domain.StatusOpen
	event.SourceInstance = "local"
	if event.IntelSource == "" {
		event.IntelSource = "Manual"
	}

	if err := event.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}

	res := authz.EventResource(event, sector.AncestryPath, nil)
	if !s.authz.Authorize(ctx, auth, authz.ActionCreate, res) {
		return ErrForbidden
	}

	event.ID = uuid.New()
	event.CreatedAt = time.Now().UTC()
	event.UpdatedAt = event.CreatedAt

	if err := s.events.Create(ctx, event); err != nil {
		return err
	}

	if event.TLP == domain.TLPRed && len(tlpRedRecipients) > 0 {
		if err := s.tlpRed.SetRecipients(ctx, "event", event.ID, tlpRedRecipients, auth.UserID); err != nil {
			return err
		}
	}

	s.audit.Log(ctx, auth, "create", "event", &event.ID, domain.SeverityInfo, map[string]any{
		"title": event.Title, "tlp": event.TLP, "impact": event.Impact,
	})
	return nil
}

func (s *EventService) GetByID(ctx context.Context, auth *domain.AuthContext, id uuid.UUID) (*domain.Event, error) {
	event, err := s.events.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if event == nil {
		return nil, ErrNotFound
	}
	sector, _ := s.sectors.GetByID(ctx, event.SectorContext)
	ancestry := ""
	if sector != nil {
		ancestry = sector.AncestryPath
	}
	recipients, _ := s.tlpRed.GetRecipients(ctx, "event", event.ID)
	res := authz.EventResource(event, ancestry, recipients)
	if !s.authz.Authorize(ctx, auth, authz.ActionView, res) {
		return nil, ErrForbidden
	}
	return event, nil
}

func (s *EventService) List(ctx context.Context, auth *domain.AuthContext, filter domain.EventFilter) (*domain.ListResult[*domain.Event], error) {
	filter.Normalize()
	result, err := s.events.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	filtered := authz.FilterAuthorized(ctx, s.authz, auth, authz.ActionView, result.Items, func(e *domain.Event) *authz.Resource {
		sec, _ := s.sectors.GetByID(ctx, e.SectorContext)
		ancestry := ""
		if sec != nil {
			ancestry = sec.AncestryPath
		}
		recipients, _ := s.tlpRed.GetRecipients(ctx, "event", e.ID)
		return authz.EventResource(e, ancestry, recipients)
	})
	return &domain.ListResult[*domain.Event]{Items: filtered, TotalCount: len(filtered)}, nil
}

func (s *EventService) Update(ctx context.Context, auth *domain.AuthContext, event *domain.Event, tlpRedRecipients []uuid.UUID) error {
	existing, err := s.events.GetByID(ctx, event.ID)
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
	recipients, _ := s.tlpRed.GetRecipients(ctx, "event", existing.ID)
	res := authz.EventResource(existing, ancestry, recipients)
	if !s.authz.Authorize(ctx, auth, authz.ActionEdit, res) {
		return ErrForbidden
	}

	existing.Title = event.Title
	existing.Description = event.Description
	existing.EventType = event.EventType
	existing.Impact = event.Impact
	existing.IntelSource = event.IntelSource
	existing.Target = event.Target
	existing.OriginalEventDate = event.OriginalEventDate

	// TLP can only become more restrictive
	if event.TLP.Restrictiveness() < existing.TLP.Restrictiveness() {
		return fmt.Errorf("%w: TLP cannot become less restrictive", ErrValidation)
	}
	existing.TLP = event.TLP

	// Sector context is immutable
	if event.SectorContext != uuid.Nil && event.SectorContext != existing.SectorContext {
		return fmt.Errorf("%w: sector_context is immutable after creation", ErrValidation)
	}

	if err := existing.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}

	existing.UpdatedAt = time.Now().UTC()
	if err := s.events.Update(ctx, existing, auth.UserID); err != nil {
		return err
	}

	if existing.TLP == domain.TLPRed && len(tlpRedRecipients) > 0 {
		if err := s.tlpRed.SetRecipients(ctx, "event", existing.ID, tlpRedRecipients, auth.UserID); err != nil {
			return err
		}
	}

	s.audit.Log(ctx, auth, "update", "event", &existing.ID, domain.SeverityInfo, map[string]any{
		"title": existing.Title, "tlp": existing.TLP, "impact": existing.Impact,
	})
	return nil
}

func (s *EventService) Delete(ctx context.Context, auth *domain.AuthContext, id uuid.UUID) error {
	existing, err := s.events.GetByID(ctx, id)
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
	res := authz.EventResource(existing, ancestry, nil)
	if !s.authz.Authorize(ctx, auth, authz.ActionDelete, res) {
		return ErrForbidden
	}
	if err := s.events.Delete(ctx, id); err != nil {
		return err
	}
	s.audit.Log(ctx, auth, "delete", "event", &id, domain.SeverityHigh, map[string]any{"title": existing.Title})
	return nil
}

func (s *EventService) CreateUpdate(ctx context.Context, auth *domain.AuthContext, update *domain.EventUpdate) error {
	event, err := s.events.GetByID(ctx, update.EventID)
	if err != nil {
		return err
	}
	if event == nil {
		return ErrNotFound
	}

	sector, _ := s.sectors.GetByID(ctx, event.SectorContext)
	ancestry := ""
	if sector != nil {
		ancestry = sector.AncestryPath
	}
	recipients, _ := s.tlpRed.GetRecipients(ctx, "event", event.ID)
	res := authz.EventResource(event, ancestry, recipients)
	if !s.authz.Authorize(ctx, auth, authz.ActionEdit, res) {
		return ErrForbidden
	}

	update.AuthorID = auth.UserID
	if err := update.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}

	// Update TLP on the update cannot be less restrictive than parent event
	if update.TLP.Restrictiveness() < event.TLP.Restrictiveness() {
		return fmt.Errorf("%w: update TLP cannot be less restrictive than event TLP", ErrValidation)
	}

	// Validate status transition
	if update.StatusChange != nil && !event.Status.CanTransitionTo(*update.StatusChange) {
		return fmt.Errorf("%w: invalid status transition from %s to %s", ErrValidation, event.Status, *update.StatusChange)
	}

	update.ID = uuid.New()
	update.CreatedAt = time.Now().UTC()

	if err := s.updates.Create(ctx, update); err != nil {
		return err
	}

	_ = s.escalationReset.ResetEscalation(ctx, update.EventID)

	s.audit.Log(ctx, auth, "create_update", "event", &event.ID, domain.SeverityInfo, map[string]any{
		"update_id": update.ID, "impact_change": update.ImpactChange, "status_change": update.StatusChange,
	})
	return nil
}

func (s *EventService) ListUpdates(ctx context.Context, auth *domain.AuthContext, eventID uuid.UUID) ([]*domain.EventUpdate, error) {
	_, err := s.GetByID(ctx, auth, eventID)
	if err != nil {
		return nil, err
	}
	return s.updates.ListByEvent(ctx, eventID)
}

func (s *EventService) GetRevisions(ctx context.Context, auth *domain.AuthContext, eventID uuid.UUID) ([]*domain.EventRevision, error) {
	_, err := s.GetByID(ctx, auth, eventID)
	if err != nil {
		return nil, err
	}
	return s.events.GetRevisions(ctx, eventID)
}
