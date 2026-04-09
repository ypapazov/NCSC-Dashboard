package authz

import (
	"context"

	"github.com/google/uuid"

	"fresnel/internal/domain"
)

// Action represents a permission-guarded operation.
type Action string

const (
	ActionView          Action = "view"
	ActionCreate        Action = "create"
	ActionEdit          Action = "edit"
	ActionDelete        Action = "delete"
	ActionManageMembers Action = "manage_members"
	ActionManageRoles   Action = "manage_roles"
	ActionLink          Action = "link"
	ActionViewAudit     Action = "view_audit"
)

// Resource is a normalised representation of any domain object being authorised
// against. Callers populate only the fields relevant to the object type; unused
// fields keep their zero value.
type Resource struct {
	Type             string // "Event", "StatusReport", "Campaign", "Sector", "Organization", "User"
	ID               uuid.UUID
	OrganizationID   uuid.UUID
	SectorContextID  uuid.UUID
	SectorAncestry   string // ancestry_path of the resource's sector context
	TLP              domain.TLP
	SubmitterID      uuid.UUID // who created the resource (events: submitter, status reports: author)
	TLPRedRecipients []uuid.UUID
}

// Authorizer decides whether a principal may perform an action on a resource.
type Authorizer interface {
	Authorize(ctx context.Context, auth *domain.AuthContext, action Action, res *Resource) bool
}

// FilterAuthorized returns the subset of items for which the given action is
// permitted. The toResource function converts each item to a Resource.
func FilterAuthorized[T any](
	ctx context.Context,
	az Authorizer,
	auth *domain.AuthContext,
	action Action,
	items []T,
	toResource func(T) *Resource,
) []T {
	out := make([]T, 0, len(items))
	for _, item := range items {
		if az.Authorize(ctx, auth, action, toResource(item)) {
			out = append(out, item)
		}
	}
	return out
}
