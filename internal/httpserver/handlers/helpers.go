package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"fresnel/internal/domain"
	"fresnel/internal/httpserver/requestctx"
	"fresnel/internal/service"
	"fresnel/internal/views"

	"github.com/a-h/templ"
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

func respondView(w http.ResponseWriter, r *http.Request, status int, component templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := component.Render(r.Context(), w); err != nil {
		slog.Error("templ render", "err", err)
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

	var component templ.Component
	switch code {
	case http.StatusForbidden:
		component = views.Error403()
	case http.StatusNotFound:
		component = views.Error404()
	default:
		component = views.Error500()
	}
	respondView(w, r, code, component)
}
