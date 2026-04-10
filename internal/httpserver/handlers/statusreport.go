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

type StatusReportHandler struct {
	reports *service.StatusReportService
}

func NewStatusReportHandler(reports *service.StatusReportService) *StatusReportHandler {
	return &StatusReportHandler{reports: reports}
}

func (h *StatusReportHandler) List(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	q := r.URL.Query()

	filter := domain.StatusReportFilter{
		ScopeType:  q.Get("scope_type"),
		Pagination: parsePagination(r),
	}
	if v := q.Get("sector_context_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			filter.SectorContextID = &id
		}
	}
	if v := q.Get("organization_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			filter.OrganizationID = &id
		}
	}
	if v := q.Get("scope_ref"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			filter.ScopeRef = &id
		}
	}
	if v := q.Get("assessed_status"); v != "" {
		s := domain.AssessedStatus(v)
		filter.AssessedStatus = &s
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

	result, err := h.reports.List(r.Context(), auth, filter)
	if err != nil {
		respondError(w, r, err)
		return
	}

	if getRenderKind(r) == requestctx.RenderJSON {
		respondJSON(w, http.StatusOK, StatusReportListData{
			User:    auth,
			Reports: result.Items,
			Total:   result.TotalCount,
		})
		return
	}
	respondView(w, r, http.StatusOK, views.ReportList(result.Items, result.TotalCount))
}

func (h *StatusReportHandler) Get(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	report, err := h.reports.GetByID(r.Context(), auth, id)
	if err != nil {
		respondError(w, r, err)
		return
	}

	if getRenderKind(r) == requestctx.RenderJSON {
		respondJSON(w, http.StatusOK, StatusReportDetailData{
			User:   auth,
			Report: report,
		})
		return
	}
	respondView(w, r, http.StatusOK, views.ReportDetail(views.ReportDetailData{
		Report: report,
	}))
}

type statusReportCreateRequest struct {
	domain.StatusReport
	EventIDs []uuid.UUID `json:"event_ids,omitempty"`
}

func (h *StatusReportHandler) Create(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	var req statusReportCreateRequest
	if err := parseJSON(r, &req); err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	if err := h.reports.Create(r.Context(), auth, &req.StatusReport, req.EventIDs); err != nil {
		respondError(w, r, err)
		return
	}
	respondJSON(w, http.StatusCreated, &req.StatusReport)
}

func (h *StatusReportHandler) Update(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	var sr domain.StatusReport
	if err := parseJSON(r, &sr); err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	sr.ID = id
	if err := h.reports.Update(r.Context(), auth, &sr); err != nil {
		respondError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, &sr)
}

func (h *StatusReportHandler) Delete(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	if err := h.reports.Delete(r.Context(), auth, id); err != nil {
		respondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *StatusReportHandler) Form(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	var report *domain.StatusReport

	if idStr := r.PathValue("id"); idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			respondError(w, r, service.ErrValidation)
			return
		}
		report, err = h.reports.GetByID(r.Context(), auth, id)
		if err != nil {
			respondError(w, r, err)
			return
		}
	}

	if getRenderKind(r) == requestctx.RenderJSON {
		respondJSON(w, http.StatusOK, StatusReportFormData{User: auth, Report: report})
		return
	}
	respondView(w, r, http.StatusOK, views.ReportForm(views.ReportFormData{
		Report: report,
	}))
}
