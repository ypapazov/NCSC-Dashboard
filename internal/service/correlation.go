package service

import (
	"context"
	"fmt"

	"fresnel/internal/authz"
	"fresnel/internal/domain"
	"fresnel/internal/storage"

	"github.com/google/uuid"
)

type CorrelationService struct {
	correlations  storage.CorrelationStore
	relationships storage.EventRelationshipStore
	events        storage.EventStore
	sectors       storage.SectorStore
	tlpRed        storage.TLPRedStore
	authz         authz.Authorizer
	audit         *AuditService
}

func NewCorrelationService(
	correlations storage.CorrelationStore,
	relationships storage.EventRelationshipStore,
	events storage.EventStore,
	sectors storage.SectorStore,
	tlpRed storage.TLPRedStore,
	az authz.Authorizer,
	audit *AuditService,
) *CorrelationService {
	return &CorrelationService{
		correlations: correlations, relationships: relationships,
		events: events, sectors: sectors, tlpRed: tlpRed, authz: az, audit: audit,
	}
}

func (s *CorrelationService) eventResource(ctx context.Context, event *domain.Event) *authz.Resource {
	sec, _ := s.sectors.GetByID(ctx, event.SectorContext)
	ancestry := ""
	if sec != nil {
		ancestry = sec.AncestryPath
	}
	recipients, _ := s.tlpRed.GetRecipients(ctx, "event", event.ID)
	return authz.EventResource(event, ancestry, recipients)
}

func (s *CorrelationService) CreateCorrelation(ctx context.Context, auth *domain.AuthContext, corr *domain.Correlation) error {
	if err := corr.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}
	corr.Normalize()

	eventA, err := s.events.GetByID(ctx, corr.EventAID)
	if err != nil || eventA == nil {
		return fmt.Errorf("%w: event_a not found", ErrValidation)
	}
	eventB, err := s.events.GetByID(ctx, corr.EventBID)
	if err != nil || eventB == nil {
		return fmt.Errorf("%w: event_b not found", ErrValidation)
	}

	resA := s.eventResource(ctx, eventA)
	resB := s.eventResource(ctx, eventB)
	if !s.authz.Authorize(ctx, auth, authz.ActionLink, resA) || !s.authz.Authorize(ctx, auth, authz.ActionLink, resB) {
		return ErrForbidden
	}

	corr.ID = uuid.New()
	corr.CreatedByUser = &auth.UserID
	if err := s.correlations.Create(ctx, corr); err != nil {
		return err
	}
	s.audit.Log(ctx, auth, "create", "correlation", &corr.ID, domain.SeverityInfo, map[string]any{
		"event_a": corr.EventAID, "event_b": corr.EventBID, "label": corr.Label,
	})
	return nil
}

func (s *CorrelationService) ListByEvent(ctx context.Context, auth *domain.AuthContext, eventID uuid.UUID) ([]*domain.Correlation, error) {
	event, err := s.events.GetByID(ctx, eventID)
	if err != nil || event == nil {
		return nil, ErrNotFound
	}
	res := s.eventResource(ctx, event)
	if !s.authz.Authorize(ctx, auth, authz.ActionView, res) {
		return nil, ErrForbidden
	}

	all, err := s.correlations.ListByEvent(ctx, eventID)
	if err != nil {
		return nil, err
	}

	// Filter: user must see both events
	var result []*domain.Correlation
	for _, c := range all {
		otherID := c.EventBID
		if otherID == eventID {
			otherID = c.EventAID
		}
		other, err := s.events.GetByID(ctx, otherID)
		if err != nil || other == nil {
			continue
		}
		otherRes := s.eventResource(ctx, other)
		if s.authz.Authorize(ctx, auth, authz.ActionView, otherRes) {
			result = append(result, c)
		}
	}
	return result, nil
}

func (s *CorrelationService) ConfirmCorrelation(ctx context.Context, auth *domain.AuthContext, id uuid.UUID) error {
	corr, err := s.correlations.GetByID(ctx, id)
	if err != nil || corr == nil {
		return ErrNotFound
	}
	eventA, _ := s.events.GetByID(ctx, corr.EventAID)
	if eventA == nil {
		return ErrNotFound
	}
	res := s.eventResource(ctx, eventA)
	if !s.authz.Authorize(ctx, auth, authz.ActionEdit, res) {
		return ErrForbidden
	}
	if err := s.correlations.UpdateType(ctx, id, domain.CorrelationConfirmed); err != nil {
		return err
	}
	s.audit.Log(ctx, auth, "confirm", "correlation", &id, domain.SeverityInfo, nil)
	return nil
}

func (s *CorrelationService) DeleteCorrelation(ctx context.Context, auth *domain.AuthContext, id uuid.UUID) error {
	corr, err := s.correlations.GetByID(ctx, id)
	if err != nil || corr == nil {
		return ErrNotFound
	}
	eventA, _ := s.events.GetByID(ctx, corr.EventAID)
	if eventA == nil {
		return ErrNotFound
	}
	res := s.eventResource(ctx, eventA)
	if !s.authz.Authorize(ctx, auth, authz.ActionDelete, res) {
		return ErrForbidden
	}
	if err := s.correlations.Delete(ctx, id); err != nil {
		return err
	}
	s.audit.Log(ctx, auth, "delete", "correlation", &id, domain.SeverityInfo, nil)
	return nil
}

func (s *CorrelationService) CreateRelationship(ctx context.Context, auth *domain.AuthContext, rel *domain.EventRelationship) error {
	if err := rel.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}
	source, _ := s.events.GetByID(ctx, rel.SourceEventID)
	target, _ := s.events.GetByID(ctx, rel.TargetEventID)
	if source == nil || target == nil {
		return fmt.Errorf("%w: referenced events not found", ErrValidation)
	}
	resS := s.eventResource(ctx, source)
	if !s.authz.Authorize(ctx, auth, authz.ActionLink, resS) {
		return ErrForbidden
	}
	rel.ID = uuid.New()
	rel.CreatedByUser = &auth.UserID
	if err := s.relationships.Create(ctx, rel); err != nil {
		return err
	}
	s.audit.Log(ctx, auth, "create", "event_relationship", &rel.ID, domain.SeverityInfo, map[string]any{
		"source": rel.SourceEventID, "target": rel.TargetEventID, "label": rel.Label,
	})
	return nil
}

func (s *CorrelationService) ListRelationshipsByEvent(ctx context.Context, auth *domain.AuthContext, eventID uuid.UUID) ([]*domain.EventRelationship, error) {
	event, _ := s.events.GetByID(ctx, eventID)
	if event == nil {
		return nil, ErrNotFound
	}
	res := s.eventResource(ctx, event)
	if !s.authz.Authorize(ctx, auth, authz.ActionView, res) {
		return nil, ErrForbidden
	}
	return s.relationships.ListByEvent(ctx, eventID)
}
