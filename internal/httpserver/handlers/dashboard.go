package handlers

import (
	"net/http"

	"fresnel/internal/httpserver/requestctx"
	"fresnel/internal/service"
	"fresnel/internal/views"
)

type DashboardHandler struct {
	dashboard *service.DashboardService
}

func NewDashboardHandler(dashboard *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{dashboard: dashboard}
}

func (h *DashboardHandler) Get(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	if auth == nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	tree, err := h.dashboard.GetTree(r.Context(), auth)
	if err != nil {
		respondError(w, r, err)
		return
	}
	var sectors []*service.DashboardNode
	if tree != nil {
		sectors = tree.Children
	}

	data := views.DashboardData{
		User:    auth,
		Sectors: sectors,
	}

	if getRenderKind(r) == requestctx.RenderJSON {
		respondJSON(w, http.StatusOK, DashboardData{
			User:    auth,
			Tree:    tree,
			Sectors: sectors,
		})
		return
	}
	respondView(w, r, http.StatusOK, views.Dashboard(data))
}
