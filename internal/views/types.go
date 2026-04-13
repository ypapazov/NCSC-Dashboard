package views

import (
	"fresnel/internal/domain"
	"fresnel/internal/service"

	"github.com/google/uuid"
)

// --- Dashboard ---

type DashboardData struct {
	User            *domain.AuthContext
	RootStatus      domain.AssessedStatus
	Sectors         []*service.DashboardNode
	RecentEvents    []*domain.Event
	ActiveCampaigns []*domain.Campaign
	ViewMode        string // "tree" or "lanes"
}

type SwimlaneData struct {
	User            *domain.AuthContext
	Sectors         []*service.DashboardNode
	ActiveCampaigns []*domain.Campaign
	FilterQuery     string // pre-built filter params for lane hx-get URLs
}

type DashboardGraphData struct {
	Events []*domain.Event
	Edges  *service.GraphEdges
}

type SyncTimelineData struct {
	Events  []*domain.Event
	Sectors []*service.DashboardNode
}

// --- Events ---

type CorrelationView struct {
	CorrelationType domain.CorrelationType
	RelatedEventID  uuid.UUID
	RelatedEventTitle string
	Label           string
}

type EventDetailData struct {
	Event           *domain.Event
	CanEdit         bool
	CanDelete       bool
	OrgName         string
	SectorName      string
	SubmitterName   string
	DescriptionHTML string
	Attachments     []*domain.Attachment
	Correlations    []CorrelationView
	Revisions       []*domain.EventRevision
}

type SectorOption struct {
	ID   uuid.UUID
	Name string
}

type RecipientOption struct {
	ID          uuid.UUID
	DisplayName string
	Email       string
	Selected    bool
}

type EventFormData struct {
	Event               *domain.Event
	EventTypes          []domain.EventType
	Sectors             []SectorOption
	AvailableRecipients []RecipientOption
}

type EventUpdateView struct {
	domain.EventUpdate
	AuthorName string
}

type EventUpdatesData struct {
	Updates            []EventUpdateView
	CanUpdate          bool
	AllowedTransitions []domain.EventStatus
	EventID            uuid.UUID
}

// --- Status Reports ---

type ReportDetailData struct {
	Report       *domain.StatusReport
	CanEdit      bool
	ScopeName    string
	AuthorName   string
	BodyHTML     string
	LinkedEvents []*domain.Event
	Revisions    []*domain.StatusReportRevision
}

type ScopeOption struct {
	ID   uuid.UUID
	Name string
	Type string
}

type ReportFormData struct {
	Report       *domain.StatusReport
	ScopeOptions []ScopeOption
}

// --- Campaigns ---

type CampaignLinkedEvent struct {
	ID         uuid.UUID
	Title      string
	Status     domain.EventStatus
	Impact     domain.Impact
	TLP        domain.TLP
	Restricted bool
}

type CampaignDetailData struct {
	Campaign        *domain.Campaign
	CanEdit         bool
	CanDelete       bool
	DescriptionHTML string
	LinkedEvents    []CampaignLinkedEvent
}

type CampaignFormData struct {
	Campaign *domain.Campaign
}

// --- Admin ---

type RoleView struct {
	Role      string
	ScopeType string
	ScopeName string
	ScopeID   uuid.UUID
}

type AdminRolesData struct {
	UserID       uuid.UUID
	UserName     string
	Roles        []RoleView
	ScopeOptions []ScopeOption
}

// --- Pagination ---

type PaginationData struct {
	From       int
	To         int
	Total      int
	HasPrev    bool
	HasNext    bool
	PrevOffset int
	NextOffset int
	Limit      int
	BaseURL    string
	Target     string
}
