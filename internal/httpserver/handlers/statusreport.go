package handlers

import (
	"net/http"
	"time"

	"fresnel/internal/authz"
	"fresnel/internal/domain"
	"fresnel/internal/httpserver/requestctx"
	"fresnel/internal/markdown"
	"fresnel/internal/service"
	"fresnel/internal/views"

	"github.com/google/uuid"
)

type StatusReportHandler struct {
	reports *service.StatusReportService
	events  *service.EventService
	lookups Lookups
}

func NewStatusReportHandler(reports *service.StatusReportService, events *service.EventService, lk Lookups) *StatusReportHandler {
	return &StatusReportHandler{reports: reports, events: events, lookups: lk}
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
	if v := q.Get("sector_ancestry"); v != "" {
		filter.SectorAncestryPrefix = v
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

	if q.Get("partial") == "timeline" {
		scopeName := q.Get("scope_name")
		if scopeName == "" {
			scopeName = "Status Timeline"
		}
		respondView(w, r, http.StatusOK, views.StatusTimeline(result.Items, scopeName))
		return
	}

	respondView(w, r, http.StatusOK, views.ReportList(result.Items, result.TotalCount))
}

func (h *StatusReportHandler) Get(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	ctx := r.Context()
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	report, err := h.reports.GetByID(ctx, auth, id)
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

	var scopeName string
	switch report.ScopeType {
	case "ORG":
		if org, _ := h.lookups.Orgs.GetByID(ctx, report.ScopeRef); org != nil {
			scopeName = org.Name
		}
	case "SECTOR":
		if sec, _ := h.lookups.Sectors.GetByID(ctx, report.ScopeRef); sec != nil {
			scopeName = sec.Name
		}
	}

	var authorName string
	if user, _ := h.lookups.Users.GetByID(ctx, report.AuthorID); user != nil {
		authorName = user.DisplayName
	}

	sec, _ := h.lookups.Sectors.GetByID(ctx, report.SectorContext)
	ancestry := ""
	if sec != nil {
		ancestry = sec.AncestryPath
	}
	recipients, _ := h.lookups.TLPRed.GetRecipients(ctx, "status_report", report.ID)
	res := authz.StatusReportResource(report, ancestry, recipients)
	canEdit := h.lookups.Authz.Authorize(ctx, auth, authz.ActionEdit, res)

	eventIDs, _ := h.reports.GetLinkedEventIDs(ctx, auth, report.ID)
	var linkedEvents []*domain.Event
	for _, eid := range eventIDs {
		if e, _ := h.events.GetByID(ctx, auth, eid); e != nil {
			linkedEvents = append(linkedEvents, e)
		}
	}

	revisions, _ := h.reports.GetRevisions(ctx, auth, report.ID)

	respondView(w, r, http.StatusOK, views.ReportDetail(views.ReportDetailData{
		Report:       report,
		CanEdit:      canEdit,
		ScopeName:    scopeName,
		AuthorName:   authorName,
		BodyHTML:     string(markdown.Render(report.Body)),
		LinkedEvents: linkedEvents,
		Revisions:    revisions,
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
	ctx := r.Context()
	var report *domain.StatusReport

	if idStr := r.PathValue("id"); idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			respondError(w, r, service.ErrValidation)
			return
		}
		report, err = h.reports.GetByID(ctx, auth, id)
		if err != nil {
			respondError(w, r, err)
			return
		}
	}

	if getRenderKind(r) == requestctx.RenderJSON {
		respondJSON(w, http.StatusOK, StatusReportFormData{User: auth, Report: report})
		return
	}

	var scopeOptions []views.ScopeOption
	if sectors, _ := h.lookups.Sectors.List(ctx); sectors != nil {
		for _, s := range sectors {
			scopeOptions = append(scopeOptions, views.ScopeOption{ID: s.ID, Name: s.Name, Type: "SECTOR"})
		}
	}
	if orgs, _ := h.lookups.Orgs.List(ctx, nil); orgs != nil {
		for _, o := range orgs {
			scopeOptions = append(scopeOptions, views.ScopeOption{ID: o.ID, Name: o.Name, Type: "ORG"})
		}
	}

	respondView(w, r, http.StatusOK, views.ReportForm(views.ReportFormData{
		Report:       report,
		ScopeOptions: scopeOptions,
	}))
}
