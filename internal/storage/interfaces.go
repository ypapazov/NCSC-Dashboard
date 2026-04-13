package storage

import (
	"context"
	"time"

	"fresnel/internal/domain"

	"github.com/google/uuid"
)

type SectorStore interface {
	Create(ctx context.Context, s *domain.Sector) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Sector, error)
	List(ctx context.Context) ([]*domain.Sector, error)
	GetChildren(ctx context.Context, parentID uuid.UUID) ([]*domain.Sector, error)
	GetDescendants(ctx context.Context, ancestryPrefix string) ([]*domain.Sector, error)
	Update(ctx context.Context, s *domain.Sector) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type OrganizationStore interface {
	Create(ctx context.Context, o *domain.Organization) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error)
	List(ctx context.Context, sectorID *uuid.UUID) ([]*domain.Organization, error)
	Update(ctx context.Context, o *domain.Organization) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListBySectorAncestry(ctx context.Context, ancestryPrefix string) ([]*domain.Organization, error)
}

type UserStore interface {
	Create(ctx context.Context, u *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	List(ctx context.Context, orgID *uuid.UUID, p domain.Pagination) (*domain.ListResult[*domain.User], error)
	Update(ctx context.Context, u *domain.User) error
	Delete(ctx context.Context, id uuid.UUID) error

	AddOrgMembership(ctx context.Context, userID, orgID, assignedBy uuid.UUID) error
	RemoveOrgMembership(ctx context.Context, userID, orgID uuid.UUID) error
	GetOrgMemberships(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}

type RoleStore interface {
	AssignRole(ctx context.Context, userID uuid.UUID, role domain.Role, scopeType domain.ScopeType, scopeID uuid.UUID, assignedBy uuid.UUID) error
	RevokeRole(ctx context.Context, userID uuid.UUID, role domain.Role, scopeType domain.ScopeType, scopeID uuid.UUID) error
	ListRoles(ctx context.Context, userID uuid.UUID) ([]domain.RoleAssignment, error)
	ListRolesByScope(ctx context.Context, scopeType domain.ScopeType, scopeID uuid.UUID) ([]domain.RoleAssignment, error)

	DesignateRoot(ctx context.Context, userID uuid.UUID, scopeType domain.ScopeType, scopeID *uuid.UUID, designatedBy uuid.UUID) error
	RevokeRoot(ctx context.Context, scopeType domain.ScopeType, scopeID *uuid.UUID) error
	GetRoot(ctx context.Context, scopeType domain.ScopeType, scopeID *uuid.UUID) (*uuid.UUID, error)
}

type AuditStore interface {
	Insert(ctx context.Context, entry *domain.AuditEntry) error
	List(ctx context.Context, filter domain.AuditFilter) (*domain.ListResult[*domain.AuditEntry], error)
}

type EventStore interface {
	Create(ctx context.Context, e *domain.Event) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Event, error)
	List(ctx context.Context, filter domain.EventFilter) (*domain.ListResult[*domain.Event], error)
	Update(ctx context.Context, e *domain.Event, changedBy uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetRevisions(ctx context.Context, eventID uuid.UUID) ([]*domain.EventRevision, error)
	CountByOrg(ctx context.Context, orgID uuid.UUID) (int, error)
}

type EventUpdateStore interface {
	Create(ctx context.Context, u *domain.EventUpdate) error
	ListByEvent(ctx context.Context, eventID uuid.UUID) ([]*domain.EventUpdate, error)
	// LatestCreatedAt returns the newest event update timestamp for the event, if any.
	LatestCreatedAt(ctx context.Context, eventID uuid.UUID) (time.Time, bool, error)
}

type AttachmentStore interface {
	Create(ctx context.Context, a *domain.Attachment) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Attachment, error)
	ListByEvent(ctx context.Context, eventID uuid.UUID) ([]*domain.Attachment, error)
	UpdateScanStatus(ctx context.Context, id uuid.UUID, status domain.ScanStatus) error
	Delete(ctx context.Context, id uuid.UUID) error
	CountByEvent(ctx context.Context, eventID uuid.UUID) (int, error)
}

type TLPRedStore interface {
	SetRecipients(ctx context.Context, resourceType string, resourceID uuid.UUID, recipientIDs []uuid.UUID, grantedBy uuid.UUID) error
	GetRecipients(ctx context.Context, resourceType string, resourceID uuid.UUID) ([]uuid.UUID, error)
}

type StatusReportStore interface {
	Create(ctx context.Context, sr *domain.StatusReport) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.StatusReport, error)
	List(ctx context.Context, filter domain.StatusReportFilter) (*domain.ListResult[*domain.StatusReport], error)
	Update(ctx context.Context, sr *domain.StatusReport, changedBy uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetRevisions(ctx context.Context, reportID uuid.UUID) ([]*domain.StatusReportRevision, error)
	LinkEvents(ctx context.Context, reportID uuid.UUID, eventIDs []uuid.UUID) error
	GetLinkedEventIDs(ctx context.Context, reportID uuid.UUID) ([]uuid.UUID, error)
	GetLatestByScope(ctx context.Context, scopeType string, scopeRef uuid.UUID) (*domain.StatusReport, error)
}

type CampaignStore interface {
	Create(ctx context.Context, c *domain.Campaign) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Campaign, error)
	List(ctx context.Context, filter domain.CampaignFilter) (*domain.ListResult[*domain.Campaign], error)
	Update(ctx context.Context, c *domain.Campaign) error
	Delete(ctx context.Context, id uuid.UUID) error
	LinkEvent(ctx context.Context, campaignID, eventID, linkedBy uuid.UUID) error
	UnlinkEvent(ctx context.Context, campaignID, eventID uuid.UUID) error
	GetLinkedEventIDs(ctx context.Context, campaignID uuid.UUID) ([]uuid.UUID, error)
}

type CorrelationStore interface {
	Create(ctx context.Context, c *domain.Correlation) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Correlation, error)
	ListByEvent(ctx context.Context, eventID uuid.UUID) ([]*domain.Correlation, error)
	ListByEventIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.Correlation, error)
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateType(ctx context.Context, id uuid.UUID, corrType domain.CorrelationType) error
}

type EventRelationshipStore interface {
	Create(ctx context.Context, r *domain.EventRelationship) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.EventRelationship, error)
	ListByEvent(ctx context.Context, eventID uuid.UUID) ([]*domain.EventRelationship, error)
	ListByEventIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.EventRelationship, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type NudgeStore interface {
	LogNudge(ctx context.Context, eventID, recipientID uuid.UUID, nudgeType string, level int) error
	HasNudgeToday(ctx context.Context, eventID, recipientID uuid.UUID) (bool, error)
	// LastNudgeSentAt returns the most recent nudge to the recipient for the event (any type).
	LastNudgeSentAt(ctx context.Context, eventID, recipientID uuid.UUID) (time.Time, bool, error)
	// LastEscalationNudgeTime returns the most recent ESCALATION nudge for the event, if any.
	LastEscalationNudgeTime(ctx context.Context, eventID uuid.UUID) (time.Time, bool, error)
	GetEscalationState(ctx context.Context, eventID uuid.UUID) (level int, lastResponse *domain.AuditEntry, err error)
	SetEscalationLevel(ctx context.Context, eventID uuid.UUID, level int) error
	ResetEscalation(ctx context.Context, eventID uuid.UUID) error
}

type FormulaStore interface {
	Get(ctx context.Context, nodeType string, nodeID *uuid.UUID) (string, error)
	Set(ctx context.Context, nodeType string, nodeID *uuid.UUID, source string, setBy uuid.UUID) error
}
