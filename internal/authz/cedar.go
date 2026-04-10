package authz

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"fresnel/internal/domain"
)

// SectorAncestryFunc resolves a sector ID to its ancestry_path.
// It is injected at construction so the authorizer can evaluate sector
// hierarchy checks without importing a repository directly.
type SectorAncestryFunc func(sectorID uuid.UUID) string

// CedarAuthorizer implements Authorizer with Go-native logic that mirrors the
// intended Cedar policy set. The role × action × TLP matrix is encoded
// directly so it is easy to test and audit.
type CedarAuthorizer struct {
	sectorAncestry SectorAncestryFunc
}

// NewCedarAuthorizer returns an Authorizer backed by the Go-native role matrix.
func NewCedarAuthorizer(saf SectorAncestryFunc) *CedarAuthorizer {
	return &CedarAuthorizer{sectorAncestry: saf}
}

func (a *CedarAuthorizer) Authorize(_ context.Context, auth *domain.AuthContext, action Action, res *Resource) bool {
	if auth == nil || res == nil {
		return false
	}
	if auth.IsRoot && auth.RootScope != nil && auth.RootScope.Type == string(domain.ScopePlatform) {
		return true
	}
	for _, role := range auth.Roles {
		if a.rolePermits(auth, role, action, res) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Role dispatch
// ---------------------------------------------------------------------------

func (a *CedarAuthorizer) rolePermits(auth *domain.AuthContext, role domain.RoleAssignment, action Action, res *Resource) bool {
	switch domain.Role(role.Role) {

	// --- Superuser roles (bypass TLP) ---

	case domain.RolePlatformRoot:
		return true

	case domain.RoleSectorRoot:
		return a.sectorCovers(role.ScopeID, res)

	case domain.RoleOrgRoot:
		return res.OrganizationID == role.ScopeID

	// --- Org management (bypass TLP within own org) ---

	case domain.RoleOrgAdmin:
		if res.OrganizationID != role.ScopeID {
			return false
		}
		return action != ActionManageRoles

	// --- Cross-org content role (bypass TLP for content resources) ---

	case domain.RoleContentAdmin:
		if res.Type != "Event" && res.Type != "StatusReport" {
			return false
		}
		return action == ActionView || action == ActionEdit

	// --- Standard data-plane roles (subject to TLP) ---

	case domain.RoleContributor:
		return a.contributorPermits(auth, role, action, res)

	case domain.RoleViewer:
		if action != ActionView {
			return false
		}
		if !a.scopeCovers(role, res) {
			return false
		}
		return a.tlpAllows(auth, res)

	case domain.RoleLiaison:
		if action != ActionView {
			return false
		}
		if !a.scopeCovers(role, res) {
			return false
		}
		return a.tlpAllows(auth, res)
	}

	return false
}

// ---------------------------------------------------------------------------
// Contributor logic
// ---------------------------------------------------------------------------

func (a *CedarAuthorizer) contributorPermits(auth *domain.AuthContext, role domain.RoleAssignment, action Action, res *Resource) bool {
	if !a.scopeCovers(role, res) {
		return false
	}

	switch action {
	case ActionView:
		return a.tlpAllows(auth, res)
	case ActionCreate:
		return res.Type == "Event"
	case ActionEdit:
		return res.Type == "Event" && res.SubmitterID == auth.UserID
	case ActionLink:
		return a.tlpAllows(auth, res)
	default:
		return false
	}
}

// ---------------------------------------------------------------------------
// Scope helpers
// ---------------------------------------------------------------------------

// sectorCovers returns true when the sector identified by sectorID is an
// ancestor of (or is) the resource's sector context.
func (a *CedarAuthorizer) sectorCovers(sectorID uuid.UUID, res *Resource) bool {
	if res.SectorAncestry == "" {
		return false
	}
	rolePath := a.sectorAncestry(sectorID)
	if rolePath == "" {
		return false
	}
	return strings.HasPrefix(res.SectorAncestry, rolePath)
}

// scopeCovers checks whether a role assignment's scope encompasses the
// resource. PLATFORM scope covers everything; SECTOR scope uses ancestry
// prefix matching; ORG scope requires an exact org match.
func (a *CedarAuthorizer) scopeCovers(role domain.RoleAssignment, res *Resource) bool {
	switch domain.ScopeType(role.ScopeType) {
	case domain.ScopePlatform:
		return true
	case domain.ScopeSector:
		return a.sectorCovers(role.ScopeID, res)
	case domain.ScopeOrg:
		return res.OrganizationID == role.ScopeID
	}
	return false
}

// ---------------------------------------------------------------------------
// TLP visibility
// ---------------------------------------------------------------------------

func (a *CedarAuthorizer) tlpAllows(auth *domain.AuthContext, res *Resource) bool {
	if res.TLP == "" {
		return true
	}

	switch res.TLP {
	case domain.TLPClear, domain.TLPGreen:
		return true

	case domain.TLPAmber:
		if isMemberOf(auth, res.OrganizationID) {
			return true
		}
		return a.hasSectorRootOver(auth, res)

	case domain.TLPAmberStrict:
		return isMemberOf(auth, res.OrganizationID)

	case domain.TLPRed:
		return isRecipient(auth.UserID, res.TLPRedRecipients)
	}

	return false
}

// hasSectorRootOver returns true if any of the user's SECTOR_ROOT assignments
// cover the resource's sector.
func (a *CedarAuthorizer) hasSectorRootOver(auth *domain.AuthContext, res *Resource) bool {
	for _, role := range auth.Roles {
		if domain.Role(role.Role) == domain.RoleSectorRoot && a.sectorCovers(role.ScopeID, res) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Tiny helpers
// ---------------------------------------------------------------------------

func isMemberOf(auth *domain.AuthContext, orgID uuid.UUID) bool {
	for _, m := range auth.OrgMemberships {
		if m == orgID {
			return true
		}
	}
	return false
}

func isRecipient(userID uuid.UUID, recipients []uuid.UUID) bool {
	for _, r := range recipients {
		if r == userID {
			return true
		}
	}
	return false
}
