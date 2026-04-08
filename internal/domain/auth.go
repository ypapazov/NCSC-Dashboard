package domain

import "github.com/google/uuid"

// ScopeEntry identifies a scope in the hierarchy (platform, sector, or org).
type ScopeEntry struct {
	Type string    // PLATFORM, SECTOR, ORG
	ID   uuid.UUID // Meaningful for SECTOR and ORG
}

// RoleAssignment is an assigned role within a scope.
type RoleAssignment struct {
	Role      string
	ScopeType string
	ScopeID   uuid.UUID
}

// AuthContext is the request-scoped identity and authorization view for a user.
type AuthContext struct {
	UserID               uuid.UUID
	KeycloakSub          string
	DisplayName          string
	Email                string
	PrimaryOrgID         uuid.UUID
	OrgMemberships       []uuid.UUID
	ActiveOrgContext     uuid.UUID
	AdministrativeScope  []ScopeEntry
	IsRoot               bool
	RootScope            *ScopeEntry
	Roles                []RoleAssignment
}
