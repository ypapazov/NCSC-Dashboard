package middleware

import (
	"net/http"
	"strings"

	"fresnel/internal/config"
	"fresnel/internal/httpserver/csrf"
	"fresnel/internal/httpserver/requestctx"
)

// CSRF validates X-CSRF-Token for unsafe methods when the client prefers HTML responses.
func CSRF(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isSafeMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}
			if !strings.Contains(r.Header.Get("Accept"), "text/html") {
				next.ServeHTTP(w, r)
				return
			}
			raw := requestctx.RawAccessTokenFrom(r.Context())
			if raw == "" {
				next.ServeHTTP(w, r)
				return
			}
			client := r.Header.Get("X-CSRF-Token")
			if !csrf.Valid(cfg.HMACSecret, raw, client) {
				http.Error(w, "invalid CSRF token", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isSafeMethod(m string) bool {
	switch m {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}
