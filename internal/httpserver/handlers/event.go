package handlers

import (
	"html/template"
	"net/http"
	"time"

	"fresnel/internal/domain"
	"fresnel/internal/service"

	"github.com/google/uuid"
)

type EventHandler struct {
	events *service.EventService
	tmpl   *template.Template
}

func NewEventHandler(events *service.EventService, tmpl *template.Template) *EventHandler {
	return &EventHandler{events: events, tmpl: tmpl}
}

func (h *EventHandler) List(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	q := r.URL.Query()

	filter := domain.EventFilter{
		Search:     q.Get("search"),
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
	if v := q.Get("status"); v != "" {
		s := domain.EventStatus(v)
		filter.Status = &s
	}
	if v := q.Get("impact"); v != "" {
		i := domain.Impact(v)
		filter.Impact = &i
	}
	if v := q.Get("event_type"); v != "" {
		et := domain.EventType(v)
		filter.EventType = &et
	}
	if v := q.Get("tlp"); v != "" {
		t := domain.TLP(v)
		filter.TLP = &t
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

	result, err := h.events.List(r.Context(), auth, filter)
	if err != nil {
		respondError(w, r, err)
		return
	}
	respond(w, r, h.tmpl, "event_list", http.StatusOK, EventListData{
		User:   auth,
		Events: result.Items,
		Total:  result.TotalCount,
		Filter: filter,
	})
}

func (h *EventHandler) Get(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	event, err := h.events.GetByID(r.Context(), auth, id)
	if err != nil {
		respondError(w, r, err)
		return
	}
	respond(w, r, h.tmpl, "event_detail", http.StatusOK, EventDetailData{
		User:  auth,
		Event: event,
	})
}

type eventCreateRequest struct {
	domain.Event
	TLPRedRecipients []uuid.UUID `json:"tlp_red_recipients,omitempty"`
}

func (h *EventHandler) Create(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	var req eventCreateRequest
	if err := parseJSON(r, &req); err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	if err := h.events.Create(r.Context(), auth, &req.Event, req.TLPRedRecipients); err != nil {
		respondError(w, r, err)
		return
	}
	respond(w, r, nil, "", http.StatusCreated, &req.Event)
}

type eventUpdateRequest struct {
	domain.Event
	TLPRedRecipients []uuid.UUID `json:"tlp_red_recipients,omitempty"`
}

func (h *EventHandler) Update(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	var req eventUpdateRequest
	if err := parseJSON(r, &req); err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	req.Event.ID = id
	if err := h.events.Update(r.Context(), auth, &req.Event, req.TLPRedRecipients); err != nil {
		respondError(w, r, err)
		return
	}
	respond(w, r, nil, "", http.StatusOK, &req.Event)
}

func (h *EventHandler) Delete(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	if err := h.events.Delete(r.Context(), auth, id); err != nil {
		respondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *EventHandler) CreateUpdate(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	eventID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	var update domain.EventUpdate
	if err := parseJSON(r, &update); err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	update.EventID = eventID
	if err := h.events.CreateUpdate(r.Context(), auth, &update); err != nil {
		respondError(w, r, err)
		return
	}
	respond(w, r, nil, "", http.StatusCreated, &update)
}

func (h *EventHandler) ListUpdates(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	eventID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	updates, err := h.events.ListUpdates(r.Context(), auth, eventID)
	if err != nil {
		respondError(w, r, err)
		return
	}
	respond(w, r, h.tmpl, "event_updates", http.StatusOK, map[string]any{
		"User":    auth,
		"Updates": updates,
		"EventID": eventID,
	})
}

func (h *EventHandler) ListRevisions(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	eventID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	revisions, err := h.events.GetRevisions(r.Context(), auth, eventID)
	if err != nil {
		respondError(w, r, err)
		return
	}
	respond(w, r, nil, "", http.StatusOK, map[string]any{
		"User":      auth,
		"Revisions": revisions,
		"EventID":   eventID,
	})
}

func (h *EventHandler) Form(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	data := EventFormData{User: auth}

	if idStr := r.PathValue("id"); idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			respondError(w, r, service.ErrValidation)
			return
		}
		event, err := h.events.GetByID(r.Context(), auth, id)
		if err != nil {
			respondError(w, r, err)
			return
		}
		data.Event = event
	}
	respond(w, r, h.tmpl, "event_form", http.StatusOK, data)
}
