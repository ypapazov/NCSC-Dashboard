package handlers

import (
	"net/http"

	"fresnel/internal/domain"
	"fresnel/internal/httpserver/requestctx"
	"fresnel/internal/service"
	"fresnel/internal/views"

	"github.com/google/uuid"
)

type DashboardHandler struct {
	dashboard    *service.DashboardService
	events       *service.EventService
	campaigns    *service.CampaignService
	correlations *service.CorrelationService
}

func NewDashboardHandler(dashboard *service.DashboardService, events *service.EventService, campaigns *service.CampaignService, correlations *service.CorrelationService) *DashboardHandler {
	return &DashboardHandler{dashboard: dashboard, events: events, campaigns: campaigns, correlations: correlations}
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
			c.EventCount, _ = h.campaigns.CountLinkedEvents(ctx, auth, c.ID)
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

	if viewMode == "lanes" || viewMode == "timeline" {
		recentResult, _ := h.events.List(ctx, auth, domain.EventFilter{
			Pagination: domain.Pagination{Limit: 200},
		})
		var allEvents []*domain.Event
		if recentResult != nil {
			allEvents = recentResult.Items
		}
		respondView(w, r, http.StatusOK, views.SyncTimelineDashboard(views.SyncTimelineData{
			Events:  allEvents,
			Sectors: sectors,
		}))
		return
	}

	if viewMode == "graph" {
		recentResult, _ := h.events.List(ctx, auth, domain.EventFilter{
			Pagination: domain.Pagination{Limit: 200},
		})
		var allEvents []*domain.Event
		if recentResult != nil {
			allEvents = recentResult.Items
		}
		var ids []uuid.UUID
		for _, e := range allEvents {
			ids = append(ids, e.ID)
		}
		edges, _ := h.correlations.ListEdgesForEvents(ctx, ids)
		campaignLinks := make(map[uuid.UUID][]uuid.UUID)
		for _, c := range activeCampaigns {
			if linked, err := h.campaigns.GetLinkedEventIDs(ctx, auth, c.ID); err == nil {
				campaignLinks[c.ID] = linked
			}
		}
		respondView(w, r, http.StatusOK, views.DashboardGraph(views.DashboardGraphData{
			Events:        allEvents,
			Edges:         edges,
			Campaigns:     activeCampaigns,
			CampaignLinks: campaignLinks,
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
