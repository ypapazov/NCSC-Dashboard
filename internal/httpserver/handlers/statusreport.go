package handlers

import (
	"html/template"
	"net/http"
	"time"

	"fresnel/internal/domain"
	"fresnel/internal/service"

	"github.com/google/uuid"
)

type StatusReportHandler struct {
	reports *service.StatusReportService
	tmpl    *template.Template
}

func NewStatusReportHandler(reports *service.StatusReportService, tmpl *template.Template) *StatusReportHandler {
	return &StatusReportHandler{reports: reports, tmpl: tmpl}
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
	respond(w, r, h.tmpl, "report_list", http.StatusOK, StatusReportListData{
		User:    auth,
		Reports: result.Items,
		Total:   result.TotalCount,
	})
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
	respond(w, r, h.tmpl, "report_detail", http.StatusOK, StatusReportDetailData{
		User:   auth,
		Report: report,
	})
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
	respond(w, r, nil, "", http.StatusCreated, &req.StatusReport)
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
	respond(w, r, nil, "", http.StatusOK, &sr)
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
	data := StatusReportFormData{User: auth}

	if idStr := r.PathValue("id"); idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			respondError(w, r, service.ErrValidation)
			return
		}
		report, err := h.reports.GetByID(r.Context(), auth, id)
		if err != nil {
			respondError(w, r, err)
			return
		}
		data.Report = report
	}
	respond(w, r, h.tmpl, "report_form", http.StatusOK, data)
}
