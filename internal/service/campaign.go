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

type CampaignService struct {
	campaigns storage.CampaignStore
	events    storage.EventStore
	sectors   storage.SectorStore
	tlpRed    storage.TLPRedStore
	authz     authz.Authorizer
	audit     *AuditService
}

func NewCampaignService(campaigns storage.CampaignStore, events storage.EventStore, sectors storage.SectorStore, tlpRed storage.TLPRedStore, az authz.Authorizer, audit *AuditService) *CampaignService {
	return &CampaignService{campaigns: campaigns, events: events, sectors: sectors, tlpRed: tlpRed, authz: az, audit: audit}
}

func (s *CampaignService) Create(ctx context.Context, auth *domain.AuthContext, campaign *domain.Campaign) error {
	if campaign.OrganizationID == uuid.Nil {
		campaign.OrganizationID = auth.ActiveOrgContext
	}
	campaign.CreatedBy = auth.UserID
	campaign.Status = domain.CampaignActive

	if err := campaign.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}

	res := authz.CampaignResource(campaign)
	if !s.authz.Authorize(ctx, auth, authz.ActionCreate, res) {
		return ErrForbidden
	}

	campaign.ID = uuid.New()
	now := time.Now().UTC()
	campaign.CreatedAt = now
	campaign.UpdatedAt = now

	if err := s.campaigns.Create(ctx, campaign); err != nil {
		return err
	}
	s.audit.Log(ctx, auth, "create", "campaign", &campaign.ID, domain.SeverityInfo, map[string]any{
		"title": campaign.Title, "tlp": campaign.TLP,
	})
	return nil
}

func (s *CampaignService) GetByID(ctx context.Context, auth *domain.AuthContext, id uuid.UUID) (*domain.Campaign, error) {
	campaign, err := s.campaigns.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if campaign == nil {
		return nil, ErrNotFound
	}
	res := authz.CampaignResource(campaign)
	if !s.authz.Authorize(ctx, auth, authz.ActionView, res) {
		return nil, ErrForbidden
	}
	return campaign, nil
}

func (s *CampaignService) List(ctx context.Context, auth *domain.AuthContext, filter domain.CampaignFilter) (*domain.ListResult[*domain.Campaign], error) {
	filter.Normalize()
	result, err := s.campaigns.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	filtered := authz.FilterAuthorized(ctx, s.authz, auth, authz.ActionView, result.Items, func(c *domain.Campaign) *authz.Resource {
		return authz.CampaignResource(c)
	})
	return &domain.ListResult[*domain.Campaign]{Items: filtered, TotalCount: result.TotalCount}, nil
}

func (s *CampaignService) Update(ctx context.Context, auth *domain.AuthContext, campaign *domain.Campaign) error {
	existing, err := s.campaigns.GetByID(ctx, campaign.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrNotFound
	}
	res := authz.CampaignResource(existing)
	if !s.authz.Authorize(ctx, auth, authz.ActionEdit, res) {
		return ErrForbidden
	}
	existing.Title = campaign.Title
	existing.Description = campaign.Description
	if campaign.TLP.Restrictiveness() < existing.TLP.Restrictiveness() {
		return fmt.Errorf("%w: TLP cannot become less restrictive", ErrValidation)
	}
	existing.TLP = campaign.TLP
	if campaign.Status != "" {
		existing.Status = campaign.Status
	}
	existing.UpdatedAt = time.Now().UTC()

	if err := s.campaigns.Update(ctx, existing); err != nil {
		return err
	}
	s.audit.Log(ctx, auth, "update", "campaign", &existing.ID, domain.SeverityInfo, map[string]any{"title": existing.Title})
	return nil
}

func (s *CampaignService) LinkEvent(ctx context.Context, auth *domain.AuthContext, campaignID, eventID uuid.UUID) error {
	campaign, err := s.campaigns.GetByID(ctx, campaignID)
	if err != nil || campaign == nil {
		return ErrNotFound
	}
	res := authz.CampaignResource(campaign)
	if !s.authz.Authorize(ctx, auth, authz.ActionLink, res) {
		return ErrForbidden
	}
	if err := s.campaigns.LinkEvent(ctx, campaignID, eventID, auth.UserID); err != nil {
		return err
	}
	s.audit.Log(ctx, auth, "link_event", "campaign", &campaignID, domain.SeverityInfo, map[string]any{"event_id": eventID})
	return nil
}

func (s *CampaignService) UnlinkEvent(ctx context.Context, auth *domain.AuthContext, campaignID, eventID uuid.UUID) error {
	campaign, err := s.campaigns.GetByID(ctx, campaignID)
	if err != nil || campaign == nil {
		return ErrNotFound
	}
	res := authz.CampaignResource(campaign)
	if !s.authz.Authorize(ctx, auth, authz.ActionEdit, res) {
		return ErrForbidden
	}
	if err := s.campaigns.UnlinkEvent(ctx, campaignID, eventID); err != nil {
		return err
	}
	s.audit.Log(ctx, auth, "unlink_event", "campaign", &campaignID, domain.SeverityInfo, map[string]any{"event_id": eventID})
	return nil
}

type CampaignEventInfo struct {
	Event      *domain.Event
	Restricted bool
}

func (s *CampaignService) GetLinkedEvents(ctx context.Context, auth *domain.AuthContext, campaignID uuid.UUID) ([]CampaignEventInfo, error) {
	if _, err := s.GetByID(ctx, auth, campaignID); err != nil {
		return nil, err
	}
	eventIDs, err := s.campaigns.GetLinkedEventIDs(ctx, campaignID)
	if err != nil {
		return nil, err
	}

	var result []CampaignEventInfo
	for _, eid := range eventIDs {
		event, err := s.events.GetByID(ctx, eid)
		if err != nil || event == nil {
			continue
		}
		sec, _ := s.sectors.GetByID(ctx, event.SectorContext)
		ancestry := ""
		if sec != nil {
			ancestry = sec.AncestryPath
		}
		recipients, _ := s.tlpRed.GetRecipients(ctx, "event", event.ID)
		res := authz.EventResource(event, ancestry, recipients)
		if s.authz.Authorize(ctx, auth, authz.ActionView, res) {
			result = append(result, CampaignEventInfo{Event: event})
		} else {
			result = append(result, CampaignEventInfo{Restricted: true})
		}
	}
	return result, nil
}
