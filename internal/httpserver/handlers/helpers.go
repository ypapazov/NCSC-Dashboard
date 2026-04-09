package handlers

import (
	"encoding/json"
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"

	"fresnel/internal/domain"
	"fresnel/internal/httpserver/requestctx"
	"fresnel/internal/service"

	"github.com/google/uuid"
)

func getAuth(r *http.Request) *domain.AuthContext {
	return requestctx.AuthFrom(r.Context())
}

func getRenderKind(r *http.Request) requestctx.RenderKind {
	return requestctx.RenderFrom(r.Context())
}

func parseUUID(r *http.Request, param string) (uuid.UUID, error) {
	return uuid.Parse(r.PathValue(param))
}

func parseJSON(r *http.Request, dest any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dest)
}

func parsePagination(r *http.Request) domain.Pagination {
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	p := domain.Pagination{Offset: offset, Limit: limit}
	p.Normalize()
	return p
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

func respondHTML(w http.ResponseWriter, tmpl *template.Template, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		slog.Error("template render", "template", name, "err", err)
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func respondError(w http.ResponseWriter, r *http.Request, err error) {
	code := http.StatusInternalServerError
	msg := "internal error"

	switch {
	case errors.Is(err, service.ErrForbidden):
		code = http.StatusForbidden
		msg = "forbidden"
	case errors.Is(err, service.ErrNotFound):
		code = http.StatusNotFound
		msg = "not found"
	case errors.Is(err, service.ErrValidation):
		code = http.StatusBadRequest
		msg = err.Error()
	case errors.Is(err, service.ErrConflict):
		code = http.StatusConflict
		msg = "conflict"
	case errors.Is(err, service.ErrQuarantined):
		code = http.StatusUnprocessableEntity
		msg = err.Error()
	}

	if getRenderKind(r) == requestctx.RenderJSON {
		respondJSON(w, code, map[string]string{"error": msg})
		return
	}
	http.Error(w, msg, code)
}

// respond writes either JSON or HTML based on content negotiation.
func respond(w http.ResponseWriter, r *http.Request, tmpl *template.Template, templateName string, status int, data any) {
	if getRenderKind(r) == requestctx.RenderJSON {
		respondJSON(w, status, data)
		return
	}
	if tmpl != nil && templateName != "" {
		w.WriteHeader(status)
		respondHTML(w, tmpl, templateName, data)
		return
	}
	respondJSON(w, status, data)
}
