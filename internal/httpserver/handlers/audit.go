package handlers

import (
	"html/template"
	"net/http"
	"time"

	"fresnel/internal/domain"
	"fresnel/internal/service"

	"github.com/google/uuid"
)

type AuditHandler struct {
	audit *service.AuditService
	tmpl  *template.Template
}

func NewAuditHandler(audit *service.AuditService, tmpl *template.Template) *AuditHandler {
	return &AuditHandler{audit: audit, tmpl: tmpl}
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
	respond(w, r, h.tmpl, "audit_list", http.StatusOK, AuditListData{
		User:    auth,
		Entries: result.Items,
		Total:   result.TotalCount,
	})
}
