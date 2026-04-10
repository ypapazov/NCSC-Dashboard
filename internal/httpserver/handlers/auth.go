package handlers

import (
	"html/template"
	"net/http"

	"fresnel/internal/config"
)

// Shell serves the app shell HTML page. This is the unauthenticated entry
// point for all browser navigation. keycloak-js (loaded by the shell page)
// handles the OIDC Authorization Code + PKCE flow client-side, then makes
// authenticated HTMX requests to the API.
func Shell(tmpl *template.Template, cfg *config.Config) http.HandlerFunc {
	data := ShellData{
		KeycloakURL:      cfg.KeycloakBrowserURL(),
		KeycloakRealm:    "fresnel",
		KeycloakClientID: cfg.KeycloakClientID,
	}
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "shell", data); err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
		}
	}
}

// Nav serves the sidebar navigation HTML fragment.
func Nav(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "nav", nil); err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
		}
	}
}
