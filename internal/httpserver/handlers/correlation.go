package handlers

import (
	"html/template"
	"net/http"

	"fresnel/internal/domain"
	"fresnel/internal/service"

	"github.com/google/uuid"
)

type CorrelationHandler struct {
	corrs *service.CorrelationService
	tmpl  *template.Template
}

func NewCorrelationHandler(corrs *service.CorrelationService, tmpl *template.Template) *CorrelationHandler {
	return &CorrelationHandler{corrs: corrs, tmpl: tmpl}
}

func (h *CorrelationHandler) ListByEvent(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	eventID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	correlations, err := h.corrs.ListByEvent(r.Context(), auth, eventID)
	if err != nil {
		respondError(w, r, err)
		return
	}
	respond(w, r, nil, "", http.StatusOK, correlations)
}

func (h *CorrelationHandler) CreateCorrelation(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	eventID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	var corr domain.Correlation
	if err := parseJSON(r, &corr); err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	// Ensure event_a_id is the path event
	if corr.EventAID == uuid.Nil {
		corr.EventAID = eventID
	}
	if err := h.corrs.CreateCorrelation(r.Context(), auth, &corr); err != nil {
		respondError(w, r, err)
		return
	}
	respond(w, r, nil, "", http.StatusCreated, &corr)
}

func (h *CorrelationHandler) ConfirmCorrelation(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	if err := h.corrs.ConfirmCorrelation(r.Context(), auth, id); err != nil {
		respondError(w, r, err)
		return
	}
	respond(w, r, nil, "", http.StatusOK, map[string]string{"status": "confirmed"})
}

func (h *CorrelationHandler) DeleteCorrelation(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	if err := h.corrs.DeleteCorrelation(r.Context(), auth, id); err != nil {
		respondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CorrelationHandler) CreateRelationship(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	eventID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	var rel domain.EventRelationship
	if err := parseJSON(r, &rel); err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	if rel.SourceEventID == uuid.Nil {
		rel.SourceEventID = eventID
	}
	if err := h.corrs.CreateRelationship(r.Context(), auth, &rel); err != nil {
		respondError(w, r, err)
		return
	}
	respond(w, r, nil, "", http.StatusCreated, &rel)
}

func (h *CorrelationHandler) ListRelationships(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	eventID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	rels, err := h.corrs.ListRelationshipsByEvent(r.Context(), auth, eventID)
	if err != nil {
		respondError(w, r, err)
		return
	}
	respond(w, r, nil, "", http.StatusOK, rels)
}
