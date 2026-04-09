package authz

import (
	"github.com/google/uuid"

	"fresnel/internal/domain"
)

// EventResource converts a domain Event into an authorisation Resource.
func EventResource(e *domain.Event, sectorAncestry string, tlpRedRecipients []uuid.UUID) *Resource {
	return &Resource{
		Type:             "Event",
		ID:               e.ID,
		OrganizationID:   e.OrganizationID,
		SectorContextID:  e.SectorContext,
		SectorAncestry:   sectorAncestry,
		TLP:              e.TLP,
		SubmitterID:      e.SubmitterID,
		TLPRedRecipients: tlpRedRecipients,
	}
}

// StatusReportResource converts a domain StatusReport into an authorisation Resource.
func StatusReportResource(sr *domain.StatusReport, sectorAncestry string, tlpRedRecipients []uuid.UUID) *Resource {
	return &Resource{
		Type:             "StatusReport",
		ID:               sr.ID,
		OrganizationID:   sr.OrganizationID,
		SectorContextID:  sr.SectorContext,
		SectorAncestry:   sectorAncestry,
		TLP:              sr.TLP,
		SubmitterID:      sr.AuthorID,
		TLPRedRecipients: tlpRedRecipients,
	}
}

// CampaignResource converts a domain Campaign into an authorisation Resource.
// Campaigns carry TLP but no sector context of their own; sector-scoped
// checks will not match unless the caller provides ancestry separately.
func CampaignResource(c *domain.Campaign) *Resource {
	return &Resource{
		Type:           "Campaign",
		ID:             c.ID,
		OrganizationID: c.OrganizationID,
		TLP:            c.TLP,
		SubmitterID:    c.CreatedBy,
	}
}

// SectorResource converts a domain Sector into an authorisation Resource.
func SectorResource(s *domain.Sector) *Resource {
	return &Resource{
		Type:            "Sector",
		ID:              s.ID,
		SectorContextID: s.ID,
		SectorAncestry:  s.AncestryPath,
	}
}

// OrgResource converts a domain Organization into an authorisation Resource.
func OrgResource(o *domain.Organization, sectorAncestry string) *Resource {
	return &Resource{
		Type:            "Organization",
		ID:              o.ID,
		OrganizationID:  o.ID,
		SectorContextID: o.SectorID,
		SectorAncestry:  sectorAncestry,
	}
}

// UserResource converts a domain User into an authorisation Resource.
func UserResource(u *domain.User) *Resource {
	return &Resource{
		Type:           "User",
		ID:             u.ID,
		OrganizationID: u.PrimaryOrgID,
	}
}
