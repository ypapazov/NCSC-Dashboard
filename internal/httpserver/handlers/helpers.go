package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/x-www-form-urlencoded") || strings.HasPrefix(ct, "multipart/form-data") {
		return r.ParseForm()
	}
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

func isFormSubmission(r *http.Request) bool {
	ct := r.Header.Get("Content-Type")
	return strings.HasPrefix(ct, "application/x-www-form-urlencoded") || strings.HasPrefix(ct, "multipart/form-data")
}

func parseEventFromForm(r *http.Request) (*domain.Event, []uuid.UUID) {
	_ = r.ParseForm()
	e := &domain.Event{
		Title:       r.FormValue("title"),
		Description: r.FormValue("description"),
		EventType:   domain.EventType(r.FormValue("event_type")),
		TLP:         domain.TLP(r.FormValue("tlp")),
		Impact:      domain.Impact(r.FormValue("impact")),
		IntelSource: r.FormValue("intel_source"),
		Target:      r.FormValue("target"),
	}
	if v := r.FormValue("sector_context"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			e.SectorContext = id
		}
	}
	if v := r.FormValue("original_event_date"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			e.OriginalEventDate = &t
		}
	}
	var recipients []uuid.UUID
	for _, v := range r.Form["recipients"] {
		if id, err := uuid.Parse(v); err == nil {
			recipients = append(recipients, id)
		}
	}
	return e, recipients
}

func parseCampaignFromForm(r *http.Request) *domain.Campaign {
	_ = r.ParseForm()
	return &domain.Campaign{
		Title:       r.FormValue("title"),
		Description: r.FormValue("description"),
		TLP:         domain.TLP(r.FormValue("tlp")),
	}
}

func parseLinkEventFromForm(r *http.Request) uuid.UUID {
	_ = r.ParseForm()
	if id, err := uuid.Parse(r.FormValue("event_id")); err == nil {
		return id
	}
	return uuid.Nil
}

func parseSectorFromForm(r *http.Request) *domain.Sector {
	_ = r.ParseForm()
	s := &domain.Sector{
		Name: r.FormValue("name"),
	}
	if v := r.FormValue("parent_sector_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			s.ParentSectorID = &id
		}
	}
	return s
}

func parseOrgFromForm(r *http.Request) *domain.Organization {
	_ = r.ParseForm()
	o := &domain.Organization{
		Name: r.FormValue("name"),
	}
	if v := r.FormValue("sector_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			o.SectorID = id
		}
	}
	return o
}

func parseUserFromForm(r *http.Request) *domain.User {
	_ = r.ParseForm()
	u := &domain.User{
		DisplayName: r.FormValue("display_name"),
		Email:       r.FormValue("email"),
	}
	if v := r.FormValue("primary_org_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			u.PrimaryOrgID = id
		}
	}
	return u
}

func parseEventUpdateFromForm(r *http.Request) *domain.EventUpdate {
	_ = r.ParseForm()
	u := &domain.EventUpdate{
		Body: r.FormValue("body"),
		TLP:  domain.TLP(r.FormValue("tlp")),
	}
	if v := r.FormValue("status_change"); v != "" {
		s := domain.EventStatus(v)
		u.StatusChange = &s
	}
	if v := r.FormValue("impact_change"); v != "" {
		i := domain.Impact(v)
		u.ImpactChange = &i
	}
	return u
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
