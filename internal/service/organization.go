package service

import (
	"context"
	"fmt"

	"fresnel/internal/authz"
	"fresnel/internal/domain"
	"fresnel/internal/storage"

	"github.com/google/uuid"
)

type OrganizationService struct {
	orgs    storage.OrganizationStore
	sectors storage.SectorStore
	authz   authz.Authorizer
	audit   *AuditService
}

func NewOrganizationService(orgs storage.OrganizationStore, sectors storage.SectorStore, az authz.Authorizer, audit *AuditService) *OrganizationService {
	return &OrganizationService{orgs: orgs, sectors: sectors, authz: az, audit: audit}
}

func (s *OrganizationService) Create(ctx context.Context, auth *domain.AuthContext, org *domain.Organization) error {
	sector, err := s.sectors.GetByID(ctx, org.SectorID)
	if err != nil || sector == nil {
		return fmt.Errorf("%w: invalid sector_id", ErrValidation)
	}
	res := authz.OrgResource(&domain.Organization{SectorID: org.SectorID}, sector.AncestryPath)
	if !s.authz.Authorize(ctx, auth, authz.ActionCreate, res) {
		return ErrForbidden
	}
	if err := org.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}
	org.ID = uuid.New()
	org.Status = "active"
	if org.Timezone == "" {
		org.Timezone = "UTC"
	}
	if err := s.orgs.Create(ctx, org); err != nil {
		return err
	}
	s.audit.Log(ctx, auth, "create", "organization", &org.ID, domain.SeverityInfo, map[string]any{
		"name": org.Name, "sector_id": org.SectorID,
	})
	return nil
}

func (s *OrganizationService) GetByID(ctx context.Context, auth *domain.AuthContext, id uuid.UUID) (*domain.Organization, error) {
	org, err := s.orgs.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if org == nil {
		return nil, ErrNotFound
	}
	sector, _ := s.sectors.GetByID(ctx, org.SectorID)
	ancestry := ""
	if sector != nil {
		ancestry = sector.AncestryPath
	}
	res := authz.OrgResource(org, ancestry)
	if !s.authz.Authorize(ctx, auth, authz.ActionView, res) {
		return nil, ErrForbidden
	}
	return org, nil
}

func (s *OrganizationService) List(ctx context.Context, auth *domain.AuthContext, sectorID *uuid.UUID) ([]*domain.Organization, error) {
	all, err := s.orgs.List(ctx, sectorID)
	if err != nil {
		return nil, err
	}
	return authz.FilterAuthorized(ctx, s.authz, auth, authz.ActionView, all, func(o *domain.Organization) *authz.Resource {
		sec, _ := s.sectors.GetByID(ctx, o.SectorID)
		ancestry := ""
		if sec != nil {
			ancestry = sec.AncestryPath
		}
		return authz.OrgResource(o, ancestry)
	}), nil
}

func (s *OrganizationService) Update(ctx context.Context, auth *domain.AuthContext, org *domain.Organization) error {
	existing, err := s.orgs.GetByID(ctx, org.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrNotFound
	}
	sector, _ := s.sectors.GetByID(ctx, existing.SectorID)
	ancestry := ""
	if sector != nil {
		ancestry = sector.AncestryPath
	}
	res := authz.OrgResource(existing, ancestry)
	if !s.authz.Authorize(ctx, auth, authz.ActionEdit, res) {
		return ErrForbidden
	}
	existing.Name = org.Name
	if org.Timezone != "" {
		existing.Timezone = org.Timezone
	}
	if err := existing.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}
	if err := s.orgs.Update(ctx, existing); err != nil {
		return err
	}
	s.audit.Log(ctx, auth, "update", "organization", &org.ID, domain.SeverityInfo, map[string]any{"name": existing.Name})
	return nil
}

func (s *OrganizationService) Delete(ctx context.Context, auth *domain.AuthContext, id uuid.UUID) error {
	existing, err := s.orgs.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrNotFound
	}
	sector, _ := s.sectors.GetByID(ctx, existing.SectorID)
	ancestry := ""
	if sector != nil {
		ancestry = sector.AncestryPath
	}
	res := authz.OrgResource(existing, ancestry)
	if !s.authz.Authorize(ctx, auth, authz.ActionDelete, res) {
		return ErrForbidden
	}
	if err := s.orgs.Delete(ctx, id); err != nil {
		return err
	}
	s.audit.Log(ctx, auth, "delete", "organization", &id, domain.SeverityHigh, map[string]any{"name": existing.Name})
	return nil
}
