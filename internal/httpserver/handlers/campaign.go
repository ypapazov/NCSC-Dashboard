package handlers

import (
	"html/template"
	"net/http"

	"fresnel/internal/domain"
	"fresnel/internal/service"

	"github.com/google/uuid"
)

type CampaignHandler struct {
	campaigns *service.CampaignService
	tmpl      *template.Template
}

func NewCampaignHandler(campaigns *service.CampaignService, tmpl *template.Template) *CampaignHandler {
	return &CampaignHandler{campaigns: campaigns, tmpl: tmpl}
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
	respond(w, r, h.tmpl, "campaign_list", http.StatusOK, CampaignListData{
		User:      auth,
		Campaigns: result.Items,
		Total:     result.TotalCount,
	})
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
	respond(w, r, h.tmpl, "campaign_detail", http.StatusOK, CampaignDetailData{
		User:     auth,
		Campaign: campaign,
	})
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
	respond(w, r, nil, "", http.StatusCreated, &campaign)
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
	respond(w, r, nil, "", http.StatusOK, &campaign)
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
	respond(w, r, nil, "", http.StatusCreated, map[string]string{"status": "linked"})
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
	respond(w, r, nil, "", http.StatusOK, events)
}

func (h *CampaignHandler) Form(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	data := CampaignFormData{User: auth}

	if idStr := r.PathValue("id"); idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			respondError(w, r, service.ErrValidation)
			return
		}
		campaign, err := h.campaigns.GetByID(r.Context(), auth, id)
		if err != nil {
			respondError(w, r, err)
			return
		}
		data.Campaign = campaign
	}
	respond(w, r, h.tmpl, "campaign_form", http.StatusOK, data)
}
