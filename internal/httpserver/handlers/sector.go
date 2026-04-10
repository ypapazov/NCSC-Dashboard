package handlers

import (
	"html/template"
	"net/http"

	"fresnel/internal/domain"
	"fresnel/internal/service"

	"github.com/google/uuid"
)

type SectorHandler struct {
	sectors *service.SectorService
	tmpl    *template.Template
}

func NewSectorHandler(sectors *service.SectorService, tmpl *template.Template) *SectorHandler {
	return &SectorHandler{sectors: sectors, tmpl: tmpl}
}

func (h *SectorHandler) List(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	sectors, err := h.sectors.List(r.Context(), auth)
	if err != nil {
		respondError(w, r, err)
		return
	}
	respond(w, r, h.tmpl, "admin_sectors", http.StatusOK, SectorListData{
		User:    auth,
		Sectors: sectors,
	})
}

func (h *SectorHandler) Get(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	sector, err := h.sectors.GetByID(r.Context(), auth, id)
	if err != nil {
		respondError(w, r, err)
		return
	}
	respond(w, r, h.tmpl, "sector_detail", http.StatusOK, SectorDetailData{
		User:   auth,
		Sector: sector,
	})
}

func (h *SectorHandler) GetChildren(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	children, err := h.sectors.GetChildren(r.Context(), auth, id)
	if err != nil {
		respondError(w, r, err)
		return
	}
	respond(w, r, nil, "", http.StatusOK, children)
}

func (h *SectorHandler) Create(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	var sector domain.Sector
	if err := parseJSON(r, &sector); err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	if err := h.sectors.Create(r.Context(), auth, &sector); err != nil {
		respondError(w, r, err)
		return
	}
	respond(w, r, nil, "", http.StatusCreated, &sector)
}

func (h *SectorHandler) Update(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	var sector domain.Sector
	if err := parseJSON(r, &sector); err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	sector.ID = id
	if err := h.sectors.Update(r.Context(), auth, &sector); err != nil {
		respondError(w, r, err)
		return
	}
	respond(w, r, nil, "", http.StatusOK, &sector)
}

func (h *SectorHandler) Delete(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	if err := h.sectors.Delete(r.Context(), auth, id); err != nil {
		respondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SectorHandler) Form(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	data := SectorFormData{User: auth}

	if idStr := r.PathValue("id"); idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			respondError(w, r, service.ErrValidation)
			return
		}
		sector, err := h.sectors.GetByID(r.Context(), auth, id)
		if err != nil {
			respondError(w, r, err)
			return
		}
		data.Sector = sector
	}
	respond(w, r, h.tmpl, "sector_form", http.StatusOK, data)
}
