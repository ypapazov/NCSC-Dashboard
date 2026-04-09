package service

import (
	"context"
	"fmt"

	"fresnel/internal/authz"
	"fresnel/internal/domain"
	"fresnel/internal/storage"

	"github.com/google/uuid"
)

type SectorService struct {
	sectors storage.SectorStore
	authz   authz.Authorizer
	audit   *AuditService
}

func NewSectorService(sectors storage.SectorStore, az authz.Authorizer, audit *AuditService) *SectorService {
	return &SectorService{sectors: sectors, authz: az, audit: audit}
}

func (s *SectorService) Create(ctx context.Context, auth *domain.AuthContext, sector *domain.Sector) error {
	res := &authz.Resource{Type: "Sector"}
	if !s.authz.Authorize(ctx, auth, authz.ActionCreate, res) {
		return ErrForbidden
	}
	if err := sector.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}

	sector.ID = uuid.New()
	sector.Status = "active"

	if sector.ParentSectorID != nil {
		parent, err := s.sectors.GetByID(ctx, *sector.ParentSectorID)
		if err != nil {
			return fmt.Errorf("parent sector: %w", err)
		}
		sector.Depth = parent.Depth + 1
		if sector.Depth > 5 {
			return fmt.Errorf("%w: maximum sector depth (5) exceeded", ErrValidation)
		}
		sector.AncestryPath = parent.AncestryPath + slug(sector.Name) + "/"
	} else {
		sector.Depth = 1
		sector.AncestryPath = "/" + slug(sector.Name) + "/"
	}

	if err := s.sectors.Create(ctx, sector); err != nil {
		return err
	}

	s.audit.Log(ctx, auth, "create", "sector", &sector.ID, domain.SeverityInfo, map[string]any{
		"name": sector.Name, "ancestry_path": sector.AncestryPath,
	})
	return nil
}

func (s *SectorService) GetByID(ctx context.Context, auth *domain.AuthContext, id uuid.UUID) (*domain.Sector, error) {
	sector, err := s.sectors.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if sector == nil {
		return nil, ErrNotFound
	}
	res := authz.SectorResource(sector)
	if !s.authz.Authorize(ctx, auth, authz.ActionView, res) {
		return nil, ErrForbidden
	}
	return sector, nil
}

func (s *SectorService) List(ctx context.Context, auth *domain.AuthContext) ([]*domain.Sector, error) {
	all, err := s.sectors.List(ctx)
	if err != nil {
		return nil, err
	}
	return authz.FilterAuthorized(ctx, s.authz, auth, authz.ActionView, all, func(sec *domain.Sector) *authz.Resource {
		return authz.SectorResource(sec)
	}), nil
}

func (s *SectorService) GetChildren(ctx context.Context, auth *domain.AuthContext, parentID uuid.UUID) ([]*domain.Sector, error) {
	children, err := s.sectors.GetChildren(ctx, parentID)
	if err != nil {
		return nil, err
	}
	return authz.FilterAuthorized(ctx, s.authz, auth, authz.ActionView, children, func(sec *domain.Sector) *authz.Resource {
		return authz.SectorResource(sec)
	}), nil
}

func (s *SectorService) Update(ctx context.Context, auth *domain.AuthContext, sector *domain.Sector) error {
	existing, err := s.sectors.GetByID(ctx, sector.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrNotFound
	}
	res := authz.SectorResource(existing)
	if !s.authz.Authorize(ctx, auth, authz.ActionEdit, res) {
		return ErrForbidden
	}
	existing.Name = sector.Name
	if err := existing.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}
	if err := s.sectors.Update(ctx, existing); err != nil {
		return err
	}
	s.audit.Log(ctx, auth, "update", "sector", &sector.ID, domain.SeverityInfo, map[string]any{"name": sector.Name})
	return nil
}

func (s *SectorService) Delete(ctx context.Context, auth *domain.AuthContext, id uuid.UUID) error {
	sector, err := s.sectors.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if sector == nil {
		return ErrNotFound
	}
	res := authz.SectorResource(sector)
	if !s.authz.Authorize(ctx, auth, authz.ActionDelete, res) {
		return ErrForbidden
	}
	if err := s.sectors.Delete(ctx, id); err != nil {
		return err
	}
	s.audit.Log(ctx, auth, "delete", "sector", &id, domain.SeverityHigh, map[string]any{"name": sector.Name})
	return nil
}

// GetAncestry returns the ancestry_path for a sector. Used by the authz system.
func (s *SectorService) GetAncestry(ctx context.Context, sectorID uuid.UUID) string {
	sec, err := s.sectors.GetByID(ctx, sectorID)
	if err != nil || sec == nil {
		return ""
	}
	return sec.AncestryPath
}

func slug(name string) string {
	var out []byte
	for _, c := range []byte(name) {
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
			out = append(out, c)
		case c >= 'A' && c <= 'Z':
			out = append(out, c+32)
		case c == ' ' || c == '-' || c == '_':
			if len(out) > 0 && out[len(out)-1] != '_' {
				out = append(out, '_')
			}
		}
	}
	if len(out) == 0 {
		return "unnamed"
	}
	return string(out)
}
