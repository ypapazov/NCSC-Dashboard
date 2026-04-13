package handlers

import (
	"net/http"

	"fresnel/internal/authz"
	"fresnel/internal/domain"
	"fresnel/internal/httpserver/requestctx"
	"fresnel/internal/markdown"
	"fresnel/internal/service"
	"fresnel/internal/views"

	"github.com/google/uuid"
)

type CampaignHandler struct {
	campaigns *service.CampaignService
	lookups   Lookups
}

func NewCampaignHandler(campaigns *service.CampaignService, lk Lookups) *CampaignHandler {
	return &CampaignHandler{campaigns: campaigns, lookups: lk}
}

func (h *CampaignHandler) List(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	q := r.URL.Query()

	filter := domain.CampaignFilter{
		Pagination: parsePagination(r),
	}
	if v := q.Get("organization_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			filter.OrganizationID = &id
		}
	}
	if v := q.Get("status"); v != "" {
		s := domain.CampaignStatus(v)
		filter.Status = &s
	}
	if v := q.Get("tlp"); v != "" {
		t := domain.TLP(v)
		filter.TLP = &t
	}
	filter.Search = q.Get("search")

	result, err := h.campaigns.List(r.Context(), auth, filter)
	if err != nil {
		respondError(w, r, err)
		return
	}

	if getRenderKind(r) == requestctx.RenderJSON {
		respondJSON(w, http.StatusOK, CampaignListData{
			User:      auth,
			Campaigns: result.Items,
			Total:     result.TotalCount,
		})
		return
	}

	if q.Get("partial") == "options" {
		respondView(w, r, http.StatusOK, views.CampaignSearchResults(result.Items))
		return
	}

	respondView(w, r, http.StatusOK, views.CampaignList(result.Items, result.TotalCount))
}

func (h *CampaignHandler) Get(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	ctx := r.Context()
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	campaign, err := h.campaigns.GetByID(ctx, auth, id)
	if err != nil {
		respondError(w, r, err)
		return
	}

	if getRenderKind(r) == requestctx.RenderJSON {
		respondJSON(w, http.StatusOK, CampaignDetailData{
			User:     auth,
			Campaign: campaign,
		})
		return
	}

	res := authz.CampaignResource(campaign)
	canEdit := h.lookups.Authz.Authorize(ctx, auth, authz.ActionEdit, res)
	canDelete := h.lookups.Authz.Authorize(ctx, auth, authz.ActionDelete, res)

	eventInfos, _ := h.campaigns.GetLinkedEvents(ctx, auth, campaign.ID)
	var linkedEvents []views.CampaignLinkedEvent
	for _, info := range eventInfos {
		if info.Restricted {
			linkedEvents = append(linkedEvents, views.CampaignLinkedEvent{Restricted: true})
		} else if info.Event != nil {
			linkedEvents = append(linkedEvents, views.CampaignLinkedEvent{
				ID:     info.Event.ID,
				Title:  info.Event.Title,
				Status: info.Event.Status,
				Impact: info.Event.Impact,
				TLP:    info.Event.TLP,
			})
		}
	}

	respondView(w, r, http.StatusOK, views.CampaignDetail(views.CampaignDetailData{
		Campaign:        campaign,
		CanEdit:         canEdit,
		CanDelete:       canDelete,
		DescriptionHTML: string(markdown.Render(campaign.Description)),
		LinkedEvents:    linkedEvents,
	}))
}

func (h *CampaignHandler) Create(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	var campaign *domain.Campaign
	if isFormSubmission(r) {
		campaign = parseCampaignFromForm(r)
	} else {
		var c domain.Campaign
		if err := parseJSON(r, &c); err != nil {
			respondError(w, r, service.ErrValidation)
			return
		}
		campaign = &c
	}
	if err := h.campaigns.Create(r.Context(), auth, campaign); err != nil {
		respondError(w, r, err)
		return
	}
	if getRenderKind(r) == requestctx.RenderJSON {
		respondJSON(w, http.StatusCreated, campaign)
		return
	}
	w.Header().Set("HX-Redirect", "/campaigns/"+campaign.ID.String())
	w.WriteHeader(http.StatusCreated)
}

func (h *CampaignHandler) Update(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	var campaign *domain.Campaign
	if isFormSubmission(r) {
		campaign = parseCampaignFromForm(r)
	} else {
		var c domain.Campaign
		if err := parseJSON(r, &c); err != nil {
			respondError(w, r, service.ErrValidation)
			return
		}
		campaign = &c
	}
	campaign.ID = id
	if err := h.campaigns.Update(r.Context(), auth, campaign); err != nil {
		respondError(w, r, err)
		return
	}
	if getRenderKind(r) == requestctx.RenderJSON {
		respondJSON(w, http.StatusOK, campaign)
		return
	}
	w.Header().Set("HX-Redirect", "/campaigns/"+id.String())
	w.WriteHeader(http.StatusOK)
}

type linkEventRequest struct {
	EventID uuid.UUID `json:"event_id"`
}

func (h *CampaignHandler) LinkEvent(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	campaignID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	var eventID uuid.UUID
	if isFormSubmission(r) {
		eventID = parseLinkEventFromForm(r)
	} else {
		var req linkEventRequest
		if err := parseJSON(r, &req); err != nil {
			respondError(w, r, service.ErrValidation)
			return
		}
		eventID = req.EventID
	}
	if eventID == uuid.Nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	if err := h.campaigns.LinkEvent(r.Context(), auth, campaignID, eventID); err != nil {
		respondError(w, r, err)
		return
	}
	if getRenderKind(r) != requestctx.RenderJSON {
		w.Header().Set("HX-Redirect", "/campaigns/"+campaignID.String())
		w.WriteHeader(http.StatusCreated)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]string{"status": "linked"})
}

func (h *CampaignHandler) UnlinkEvent(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	campaignID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	eventID, err := parseUUID(r, "eventId")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	if err := h.campaigns.UnlinkEvent(r.Context(), auth, campaignID, eventID); err != nil {
		respondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CampaignHandler) GetLinkedEvents(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	campaignID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	infos, err := h.campaigns.GetLinkedEvents(r.Context(), auth, campaignID)
	if err != nil {
		respondError(w, r, err)
		return
	}

	if r.URL.Query().Get("partial") == "cards" {
		var visible []*domain.Event
		for _, info := range infos {
			if !info.Restricted && info.Event != nil {
				visible = append(visible, info.Event)
			}
		}
		if len(visible) == 0 {
			respondView(w, r, http.StatusOK, views.SwimlaneLaneEmpty())
			return
		}
		respondView(w, r, http.StatusOK, views.EventCardRow(visible, "", len(visible), 0, len(visible)))
		return
	}

	respondJSON(w, http.StatusOK, infos)
}

type createFromSelectionRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	TLP         string   `json:"tlp"`
	EventIDs    []string `json:"event_ids"`
}

func (h *CampaignHandler) CreateFromSelection(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)

	var campaign domain.Campaign
	var eventIDs []uuid.UUID

	if isFormSubmission(r) {
		_ = r.ParseForm()
		campaign.Title = r.FormValue("title")
		campaign.Description = r.FormValue("description")
		campaign.TLP = domain.TLP(r.FormValue("tlp"))
		for _, idStr := range r.Form["event_ids"] {
			if id, err := uuid.Parse(idStr); err == nil {
				eventIDs = append(eventIDs, id)
			}
		}
	} else {
		var req createFromSelectionRequest
		if err := parseJSON(r, &req); err != nil {
			respondError(w, r, service.ErrValidation)
			return
		}
		campaign.Title = req.Title
		campaign.Description = req.Description
		campaign.TLP = domain.TLP(req.TLP)
		for _, idStr := range req.EventIDs {
			if id, err := uuid.Parse(idStr); err == nil {
				eventIDs = append(eventIDs, id)
			}
		}
	}

	if err := h.campaigns.CreateFromSelection(r.Context(), auth, &campaign, eventIDs); err != nil {
		respondError(w, r, err)
		return
	}
	if getRenderKind(r) == requestctx.RenderJSON {
		respondJSON(w, http.StatusCreated, campaign)
		return
	}
	w.Header().Set("HX-Redirect", "/campaigns/"+campaign.ID.String())
	w.WriteHeader(http.StatusCreated)
}

func (h *CampaignHandler) Form(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	ctx := r.Context()
	var campaign *domain.Campaign

	if idStr := r.PathValue("id"); idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			respondError(w, r, service.ErrValidation)
			return
		}
		campaign, err = h.campaigns.GetByID(ctx, auth, id)
		if err != nil {
			respondError(w, r, err)
			return
		}
	}

	if getRenderKind(r) == requestctx.RenderJSON {
		respondJSON(w, http.StatusOK, CampaignFormData{User: auth, Campaign: campaign})
		return
	}
	respondView(w, r, http.StatusOK, views.CampaignForm(views.CampaignFormData{
		Campaign: campaign,
	}))
}
