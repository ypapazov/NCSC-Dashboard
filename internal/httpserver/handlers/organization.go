package handlers

import (
	"net/http"

	"fresnel/internal/domain"
	"fresnel/internal/httpserver/requestctx"
	"fresnel/internal/service"
	"fresnel/internal/views"

	"github.com/google/uuid"
)

type OrgHandler struct {
	orgs *service.OrganizationService
}

func NewOrgHandler(orgs *service.OrganizationService) *OrgHandler {
	return &OrgHandler{orgs: orgs}
}

func (h *OrgHandler) List(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	q := r.URL.Query()

	var sectorID *uuid.UUID
	if v := q.Get("sector_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			sectorID = &id
		}
	}

	orgs, err := h.orgs.List(r.Context(), auth, sectorID)
	if err != nil {
		respondError(w, r, err)
		return
	}

	if getRenderKind(r) == requestctx.RenderJSON {
		respondJSON(w, http.StatusOK, OrgListData{
			User: auth,
			Orgs: orgs,
		})
		return
	}
	respondView(w, r, http.StatusOK, views.AdminOrgs(orgs))
}

func (h *OrgHandler) Get(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	org, err := h.orgs.GetByID(r.Context(), auth, id)
	if err != nil {
		respondError(w, r, err)
		return
	}

	if getRenderKind(r) == requestctx.RenderJSON {
		respondJSON(w, http.StatusOK, OrgDetailData{
			User: auth,
			Org:  org,
		})
		return
	}
	respondView(w, r, http.StatusOK, views.OrgDetail(org))
}

func (h *OrgHandler) Create(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	var org domain.Organization
	if err := parseJSON(r, &org); err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	if err := h.orgs.Create(r.Context(), auth, &org); err != nil {
		respondError(w, r, err)
		return
	}
	respondJSON(w, http.StatusCreated, &org)
}

func (h *OrgHandler) Update(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	var org domain.Organization
	if err := parseJSON(r, &org); err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	org.ID = id
	if err := h.orgs.Update(r.Context(), auth, &org); err != nil {
		respondError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, &org)
}

func (h *OrgHandler) Delete(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	if err := h.orgs.Delete(r.Context(), auth, id); err != nil {
		respondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *OrgHandler) Form(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	var org *domain.Organization

	if idStr := r.PathValue("id"); idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			respondError(w, r, service.ErrValidation)
			return
		}
		org, err = h.orgs.GetByID(r.Context(), auth, id)
		if err != nil {
			respondError(w, r, err)
			return
		}
	}

	if getRenderKind(r) == requestctx.RenderJSON {
		respondJSON(w, http.StatusOK, OrgFormData{User: auth, Org: org})
		return
	}
	respondView(w, r, http.StatusOK, views.OrgForm(org))
}
