package handlers

import (
	"net/http"

	"fresnel/internal/domain"
	"fresnel/internal/httpserver/requestctx"
	"fresnel/internal/service"
	"fresnel/internal/views"

	"github.com/google/uuid"
)

type SectorHandler struct {
	sectors *service.SectorService
}

func NewSectorHandler(sectors *service.SectorService) *SectorHandler {
	return &SectorHandler{sectors: sectors}
}

func (h *SectorHandler) List(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	sectors, err := h.sectors.List(r.Context(), auth)
	if err != nil {
		respondError(w, r, err)
		return
	}

	if getRenderKind(r) == requestctx.RenderJSON {
		respondJSON(w, http.StatusOK, SectorListData{
			User:    auth,
			Sectors: sectors,
		})
		return
	}
	respondView(w, r, http.StatusOK, views.AdminSectors(sectors))
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

	if getRenderKind(r) == requestctx.RenderJSON {
		respondJSON(w, http.StatusOK, SectorDetailData{
			User:   auth,
			Sector: sector,
		})
		return
	}
	respondView(w, r, http.StatusOK, views.SectorDetail(sector))
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
	respondJSON(w, http.StatusOK, children)
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
	respondJSON(w, http.StatusCreated, &sector)
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
	respondJSON(w, http.StatusOK, &sector)
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
	var sector *domain.Sector

	if idStr := r.PathValue("id"); idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			respondError(w, r, service.ErrValidation)
			return
		}
		sector, err = h.sectors.GetByID(r.Context(), auth, id)
		if err != nil {
			respondError(w, r, err)
			return
		}
	}

	if getRenderKind(r) == requestctx.RenderJSON {
		respondJSON(w, http.StatusOK, SectorFormData{User: auth, Sector: sector})
		return
	}
	respondView(w, r, http.StatusOK, views.SectorForm(sector))
}
