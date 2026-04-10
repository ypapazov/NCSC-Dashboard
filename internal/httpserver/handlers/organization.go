package handlers

import (
	"html/template"
	"net/http"

	"fresnel/internal/domain"
	"fresnel/internal/service"

	"github.com/google/uuid"
)

type OrgHandler struct {
	orgs *service.OrganizationService
	tmpl *template.Template
}

func NewOrgHandler(orgs *service.OrganizationService, tmpl *template.Template) *OrgHandler {
	return &OrgHandler{orgs: orgs, tmpl: tmpl}
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
	respond(w, r, h.tmpl, "admin_orgs", http.StatusOK, OrgListData{
		User: auth,
		Orgs: orgs,
	})
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
	respond(w, r, h.tmpl, "org_detail", http.StatusOK, OrgDetailData{
		User: auth,
		Org:  org,
	})
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
	respond(w, r, nil, "", http.StatusCreated, &org)
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
	respond(w, r, nil, "", http.StatusOK, &org)
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
	data := OrgFormData{User: auth}

	if idStr := r.PathValue("id"); idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			respondError(w, r, service.ErrValidation)
			return
		}
		org, err := h.orgs.GetByID(r.Context(), auth, id)
		if err != nil {
			respondError(w, r, err)
			return
		}
		data.Org = org
	}
	respond(w, r, h.tmpl, "org_form", http.StatusOK, data)
}
