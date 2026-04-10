package handlers

import (
	"net/http"

	"fresnel/internal/domain"
	"fresnel/internal/httpserver/requestctx"
	"fresnel/internal/service"
	"fresnel/internal/views"

	"github.com/google/uuid"
)

type CampaignHandler struct {
	campaigns *service.CampaignService
}

func NewCampaignHandler(campaigns *service.CampaignService) *CampaignHandler {
	return &CampaignHandler{campaigns: campaigns}
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
	respondView(w, r, http.StatusOK, views.CampaignList(result.Items, result.TotalCount))
}

func (h *CampaignHandler) Get(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	campaign, err := h.campaigns.GetByID(r.Context(), auth, id)
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
	respondView(w, r, http.StatusOK, views.CampaignDetail(views.CampaignDetailData{
		Campaign: campaign,
	}))
}

func (h *CampaignHandler) Create(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	var campaign domain.Campaign
	if err := parseJSON(r, &campaign); err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	if err := h.campaigns.Create(r.Context(), auth, &campaign); err != nil {
		respondError(w, r, err)
		return
	}
	respondJSON(w, http.StatusCreated, &campaign)
}

func (h *CampaignHandler) Update(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	var campaign domain.Campaign
	if err := parseJSON(r, &campaign); err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	campaign.ID = id
	if err := h.campaigns.Update(r.Context(), auth, &campaign); err != nil {
		respondError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, &campaign)
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
	var req linkEventRequest
	if err := parseJSON(r, &req); err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	if err := h.campaigns.LinkEvent(r.Context(), auth, campaignID, req.EventID); err != nil {
		respondError(w, r, err)
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
	events, err := h.campaigns.GetLinkedEvents(r.Context(), auth, campaignID)
	if err != nil {
		respondError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, events)
}

func (h *CampaignHandler) Form(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	var campaign *domain.Campaign

	if idStr := r.PathValue("id"); idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			respondError(w, r, service.ErrValidation)
			return
		}
		campaign, err = h.campaigns.GetByID(r.Context(), auth, id)
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
