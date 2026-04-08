package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"

	"fresnel/internal/httpserver/requestctx"
)

// Dashboard is a placeholder home page (HTML or JSON).
func Dashboard(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && r.URL.Path != "/dashboard" {
			http.NotFound(w, r)
			return
		}
		auth := requestctx.AuthFrom(r.Context())
		if requestctx.RenderFrom(r.Context()) == requestctx.RenderJSON {
			w.Header().Set("Content-Type", "application/json")
			if auth == nil {
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"page":       "dashboard",
				"user_id":    auth.UserID.String(),
				"email":      auth.Email,
				"display_name": auth.DisplayName,
			})
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := PageData{User: auth}
		if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
		}
	}
}
