package handlers

import (
	"fresnel/internal/domain"
	"fresnel/internal/service"
)

// PageData is passed to authenticated content templates (HTML fragments).
type PageData struct {
	User *domain.AuthContext
}

// ShellData is passed to the app shell template (unauthenticated bootstrap page).
type ShellData struct {
	KeycloakURL      string
	KeycloakRealm    string
	KeycloakClientID string
}

// --- Dashboard ---

type DashboardData struct {
	User *domain.AuthContext
	Tree *service.DashboardNode
}

// --- Events ---

type EventListData struct {
	User   *domain.AuthContext
	Events []*domain.Event
	Total  int
	Filter domain.EventFilter
}

type EventDetailData struct {
	User  *domain.AuthContext
	Event *domain.Event
}

type EventFormData struct {
	User  *domain.AuthContext
	Event *domain.Event // nil for new
}

// --- Status Reports ---

type StatusReportListData struct {
	User    *domain.AuthContext
	Reports []*domain.StatusReport
	Total   int
}

type StatusReportDetailData struct {
	User   *domain.AuthContext
	Report *domain.StatusReport
}

type StatusReportFormData struct {
	User   *domain.AuthContext
	Report *domain.StatusReport // nil for new
}

// --- Campaigns ---

type CampaignListData struct {
	User      *domain.AuthContext
	Campaigns []*domain.Campaign
	Total     int
}

type CampaignDetailData struct {
	User     *domain.AuthContext
	Campaign *domain.Campaign
}

type CampaignFormData struct {
	User     *domain.AuthContext
	Campaign *domain.Campaign // nil for new
}

// --- Sectors ---

type SectorListData struct {
	User    *domain.AuthContext
	Sectors []*domain.Sector
}

type SectorDetailData struct {
	User   *domain.AuthContext
	Sector *domain.Sector
}

type SectorFormData struct {
	User   *domain.AuthContext
	Sector *domain.Sector // nil for new
}

// --- Organizations ---

type OrgListData struct {
	User *domain.AuthContext
	Orgs []*domain.Organization
}

type OrgDetailData struct {
	User *domain.AuthContext
	Org  *domain.Organization
}

type OrgFormData struct {
	User *domain.AuthContext
	Org  *domain.Organization // nil for new
}

// --- Users ---

type UserListData struct {
	User  *domain.AuthContext
	Users []*domain.User
	Total int
}

type UserDetailData struct {
	User        *domain.AuthContext
	ProfileUser *domain.User
}

type UserFormData struct {
	User        *domain.AuthContext
	ProfileUser *domain.User // nil for new
}

// --- Audit ---

type AuditListData struct {
	User    *domain.AuthContext
	Entries []*domain.AuditEntry
	Total   int
}
