package handlers

import (
	"fresnel/internal/authz"
	"fresnel/internal/storage"
)

// Lookups provides direct store access for view-layer data enrichment.
type Lookups struct {
	Orgs    storage.OrganizationStore
	Sectors storage.SectorStore
	Users   storage.UserStore
	TLPRed  storage.TLPRedStore
	Authz   authz.Authorizer
}
