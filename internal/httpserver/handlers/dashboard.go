package handlers

import (
	"net/http"

	"fresnel/internal/domain"
	"fresnel/internal/httpserver/requestctx"
	"fresnel/internal/service"
	"fresnel/internal/views"
)

type DashboardHandler struct {
	dashboard *service.DashboardService
	events    *service.EventService
	campaigns *service.CampaignService
}

func NewDashboardHandler(dashboard *service.DashboardService, events *service.EventService, campaigns *service.CampaignService) *DashboardHandler {
	return &DashboardHandler{dashboard: dashboard, events: events, campaigns: campaigns}
}

func (h *DashboardHandler) Get(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	if auth == nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	ctx := r.Context()
	viewMode := r.URL.Query().Get("view")
	if viewMode == "" {
		viewMode = "tree"
	}

	tree, err := h.dashboard.GetTree(ctx, auth)
	if err != nil {
		respondError(w, r, err)
		return
	}
	var sectors []*service.DashboardNode
	var rootStatus domain.AssessedStatus
	if tree != nil {
		sectors = tree.Children
		rootStatus = tree.AssessedStatus
	}

	activeStatus := domain.CampaignActive
	campaignResult, _ := h.campaigns.List(ctx, auth, domain.CampaignFilter{
		Status:     &activeStatus,
		Pagination: domain.Pagination{Limit: 10},
	})
	var activeCampaigns []*domain.Campaign
	if campaignResult != nil {
		activeCampaigns = campaignResult.Items
		for _, c := range activeCampaigns {
			c.EventCount, _ = h.campaigns.CountLinkedEvents(ctx, c.ID)
		}
	}

	if getRenderKind(r) == requestctx.RenderJSON {
		recentResult, _ := h.events.List(ctx, auth, domain.EventFilter{
			Pagination: domain.Pagination{Limit: 10},
		})
		var recentEvents []*domain.Event
		if recentResult != nil {
			recentEvents = recentResult.Items
		}
		respondJSON(w, http.StatusOK, DashboardData{
			User:            auth,
			Tree:            tree,
			Sectors:         sectors,
			RecentEvents:    recentEvents,
			ActiveCampaigns: activeCampaigns,
		})
		return
	}

	if viewMode == "lanes" {
		q := r.URL.Query()
		filterQuery := views.SwimlaneBuildFilterQuery(q.Get("impact"), q.Get("tlp"), q.Get("status"), q.Get("event_type"))
		respondView(w, r, http.StatusOK, views.SwimlaneDashboard(views.SwimlaneData{
			User:            auth,
			Sectors:         sectors,
			ActiveCampaigns: activeCampaigns,
			FilterQuery:     filterQuery,
		}))
		return
	}

	recentResult, _ := h.events.List(ctx, auth, domain.EventFilter{
		Pagination: domain.Pagination{Limit: 10},
	})
	var recentEvents []*domain.Event
	if recentResult != nil {
		recentEvents = recentResult.Items
	}

	respondView(w, r, http.StatusOK, views.Dashboard(views.DashboardData{
		User:            auth,
		RootStatus:      rootStatus,
		Sectors:         sectors,
		RecentEvents:    recentEvents,
		ActiveCampaigns: activeCampaigns,
		ViewMode:        viewMode,
	}))
}
