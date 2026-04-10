package handlers

import (
	"html/template"
	"net/http"

	"fresnel/internal/service"
)

type DashboardHandler struct {
	dashboard *service.DashboardService
	tmpl      *template.Template
}

func NewDashboardHandler(dashboard *service.DashboardService, tmpl *template.Template) *DashboardHandler {
	return &DashboardHandler{dashboard: dashboard, tmpl: tmpl}
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
	respond(w, r, h.tmpl, "dashboard", http.StatusOK, DashboardData{
		User:    auth,
		Tree:    tree,
		Sectors: sectors,
	})
}
