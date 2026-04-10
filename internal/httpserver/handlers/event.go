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

type EventHandler struct {
	events       *service.EventService
	attachments  *service.AttachmentService
	correlations *service.CorrelationService
	lookups      Lookups
}

func NewEventHandler(events *service.EventService, attachments *service.AttachmentService, correlations *service.CorrelationService, lk Lookups) *EventHandler {
	return &EventHandler{events: events, attachments: attachments, correlations: correlations, lookups: lk}
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
	if v := q.Get("sort"); v == "updated_at" {
		filter.SortBy = "updated_at"
	}

	result, err := h.events.List(r.Context(), auth, filter)
	if err != nil {
		respondError(w, r, err)
		return
	}

	if getRenderKind(r) == requestctx.RenderJSON {
		respondJSON(w, http.StatusOK, EventListData{
			User:   auth,
			Events: result.Items,
			Total:  result.TotalCount,
			Filter: filter,
		})
		return
	}

	if q.Get("partial") == "cards" {
		orgID := q.Get("organization_id")
		if len(result.Items) == 0 {
			respondView(w, r, http.StatusOK, views.SwimlaneLaneEmpty())
			return
		}
		respondView(w, r, http.StatusOK, views.EventCardRow(result.Items, orgID, result.TotalCount, filter.Offset, filter.Limit))
		return
	}

	respondView(w, r, http.StatusOK, views.EventList(result.Items, result.TotalCount))
}

func (h *EventHandler) Get(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	ctx := r.Context()
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	event, err := h.events.GetByID(ctx, auth, id)
	if err != nil {
		respondError(w, r, err)
		return
	}

	if getRenderKind(r) == requestctx.RenderJSON {
		respondJSON(w, http.StatusOK, EventDetailData{
			User:  auth,
			Event: event,
		})
		return
	}

	var orgName string
	if org, _ := h.lookups.Orgs.GetByID(ctx, event.OrganizationID); org != nil {
		orgName = org.Name
	}
	var sectorName, ancestry string
	if sec, _ := h.lookups.Sectors.GetByID(ctx, event.SectorContext); sec != nil {
		sectorName = sec.Name
		ancestry = sec.AncestryPath
	}
	var submitterName string
	if user, _ := h.lookups.Users.GetByID(ctx, event.SubmitterID); user != nil {
		submitterName = user.DisplayName
	}

	recipients, _ := h.lookups.TLPRed.GetRecipients(ctx, "event", event.ID)
	res := authz.EventResource(event, ancestry, recipients)
	canEdit := h.lookups.Authz.Authorize(ctx, auth, authz.ActionEdit, res)
	canDelete := h.lookups.Authz.Authorize(ctx, auth, authz.ActionDelete, res)

	attachments, _ := h.attachments.ListByEvent(ctx, auth, event.ID)

	corrs, _ := h.correlations.ListByEvent(ctx, auth, event.ID)
	var corrViews []views.CorrelationView
	for _, c := range corrs {
		relatedID := c.EventBID
		if relatedID == event.ID {
			relatedID = c.EventAID
		}
		var title string
		if related, _ := h.events.GetByID(ctx, auth, relatedID); related != nil {
			title = related.Title
		}
		corrViews = append(corrViews, views.CorrelationView{
			CorrelationType:   c.CorrelationType,
			RelatedEventID:    relatedID,
			RelatedEventTitle: title,
			Label:             c.Label,
		})
	}

	revisions, _ := h.events.GetRevisions(ctx, auth, event.ID)

	respondView(w, r, http.StatusOK, views.EventDetail(views.EventDetailData{
		Event:           event,
		CanEdit:         canEdit,
		CanDelete:       canDelete,
		OrgName:         orgName,
		SectorName:      sectorName,
		SubmitterName:   submitterName,
		DescriptionHTML: string(markdown.Render(event.Description)),
		Attachments:     attachments,
		Correlations:    corrViews,
		Revisions:       revisions,
	}))
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
	respondJSON(w, http.StatusCreated, &req.Event)
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
	respondJSON(w, http.StatusOK, &req.Event)
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
	respondJSON(w, http.StatusCreated, &update)
}

func (h *EventHandler) ListUpdates(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	ctx := r.Context()
	eventID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	updates, err := h.events.ListUpdates(ctx, auth, eventID)
	if err != nil {
		respondError(w, r, err)
		return
	}

	if getRenderKind(r) == requestctx.RenderJSON {
		respondJSON(w, http.StatusOK, map[string]any{
			"User":    auth,
			"Updates": updates,
			"EventID": eventID,
		})
		return
	}

	viewUpdates := make([]views.EventUpdateView, len(updates))
	for i, u := range updates {
		authorName := views.FmtUser(u.AuthorID)
		if user, _ := h.lookups.Users.GetByID(ctx, u.AuthorID); user != nil {
			authorName = user.DisplayName
		}
		viewUpdates[i] = views.EventUpdateView{
			EventUpdate: *u,
			AuthorName:  authorName,
		}
	}

	var canUpdate bool
	var allowedTransitions []domain.EventStatus
	if event, _ := h.events.GetByID(ctx, auth, eventID); event != nil {
		sec, _ := h.lookups.Sectors.GetByID(ctx, event.SectorContext)
		ancestry := ""
		if sec != nil {
			ancestry = sec.AncestryPath
		}
		recipients, _ := h.lookups.TLPRed.GetRecipients(ctx, "event", event.ID)
		res := authz.EventResource(event, ancestry, recipients)
		canUpdate = h.lookups.Authz.Authorize(ctx, auth, authz.ActionEdit, res)

		for _, s := range []domain.EventStatus{
			domain.StatusInvestigating, domain.StatusMitigating,
			domain.StatusResolved, domain.StatusClosed,
		} {
			if event.Status.CanTransitionTo(s) {
				allowedTransitions = append(allowedTransitions, s)
			}
		}
	}

	respondView(w, r, http.StatusOK, views.EventUpdates(views.EventUpdatesData{
		Updates:            viewUpdates,
		CanUpdate:          canUpdate,
		AllowedTransitions: allowedTransitions,
		EventID:            eventID,
	}))
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
	respondJSON(w, http.StatusOK, map[string]any{
		"User":      auth,
		"Revisions": revisions,
		"EventID":   eventID,
	})
}

func (h *EventHandler) Form(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	ctx := r.Context()
	var event *domain.Event

	if idStr := r.PathValue("id"); idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			respondError(w, r, service.ErrValidation)
			return
		}
		event, err = h.events.GetByID(ctx, auth, id)
		if err != nil {
			respondError(w, r, err)
			return
		}
	}

	if getRenderKind(r) == requestctx.RenderJSON {
		respondJSON(w, http.StatusOK, EventFormData{User: auth, Event: event})
		return
	}

	sectors, _ := h.lookups.Sectors.List(ctx)
	sectorOpts := make([]views.SectorOption, len(sectors))
	for i, s := range sectors {
		sectorOpts[i] = views.SectorOption{ID: s.ID, Name: s.Name}
	}

	eventTypes := []domain.EventType{
		domain.EventTypePhishing, domain.EventTypeMalware, domain.EventTypeRansomware,
		domain.EventTypeDDoS, domain.EventTypeDataBreach, domain.EventTypeUnauthorized,
		domain.EventTypeWebDefacement, domain.EventTypeInsiderThreat,
		domain.EventTypeSupplyChain, domain.EventTypeVulnerability, domain.EventTypeOther,
	}

	var recipients []views.RecipientOption
	if event != nil && event.TLP == domain.TLPRed {
		existingRecipients, _ := h.lookups.TLPRed.GetRecipients(ctx, "event", event.ID)
		recipientSet := make(map[uuid.UUID]bool)
		for _, rid := range existingRecipients {
			recipientSet[rid] = true
		}
		if users, _ := h.lookups.Users.List(ctx, nil, domain.Pagination{Limit: 100}); users != nil {
			for _, u := range users.Items {
				recipients = append(recipients, views.RecipientOption{
					ID:          u.ID,
					DisplayName: u.DisplayName,
					Email:       u.Email,
					Selected:    recipientSet[u.ID],
				})
			}
		}
	}

	respondView(w, r, http.StatusOK, views.EventForm(views.EventFormData{
		Event:               event,
		EventTypes:          eventTypes,
		Sectors:             sectorOpts,
		AvailableRecipients: recipients,
	}))
}
