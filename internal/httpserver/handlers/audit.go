package handlers

import (
	"net/http"
	"time"

	"fresnel/internal/domain"
	"fresnel/internal/httpserver/requestctx"
	"fresnel/internal/service"
	"fresnel/internal/views"

	"github.com/google/uuid"
)

type AuditHandler struct {
	audit *service.AuditService
}

func NewAuditHandler(audit *service.AuditService) *AuditHandler {
	return &AuditHandler{audit: audit}
}

func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	q := r.URL.Query()

	filter := domain.AuditFilter{
		ResourceType: q.Get("resource_type"),
		ScopeType:    q.Get("scope_type"),
		Pagination:   parsePagination(r),
	}
	if v := q.Get("actor_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			filter.ActorID = &id
		}
	}
	if v := q.Get("resource_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			filter.ResourceID = &id
		}
	}
	if v := q.Get("scope_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			filter.ScopeID = &id
		}
	}
	if v := q.Get("severity"); v != "" {
		s := domain.AuditSeverity(v)
		filter.Severity = &s
	}
	if v := q.Get("date_from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.DateFrom = &t
		}
	}
	if v := q.Get("date_to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.DateTo = &t
		}
	}

	result, err := h.audit.List(r.Context(), filter)
	if err != nil {
		respondError(w, r, err)
		return
	}

	if getRenderKind(r) == requestctx.RenderJSON {
		respondJSON(w, http.StatusOK, AuditListData{
			User:    auth,
			Entries: result.Items,
			Total:   result.TotalCount,
		})
		return
	}
	respondView(w, r, http.StatusOK, views.AuditLog(result.Items, result.TotalCount))
}
