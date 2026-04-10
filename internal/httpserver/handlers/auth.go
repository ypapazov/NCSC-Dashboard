package handlers

import (
	"net/http"

	"fresnel/internal/config"
	"fresnel/internal/views"
)

func Shell(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		respondView(w, r, http.StatusOK, views.Shell(
			cfg.KeycloakBrowserURL(),
			"fresnel",
			cfg.KeycloakClientID,
		))
	}
}

func Nav() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		respondView(w, r, http.StatusOK, views.Nav())
	}
}
