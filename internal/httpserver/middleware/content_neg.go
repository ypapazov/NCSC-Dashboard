package middleware

import (
	"net/http"
	"strings"

	"fresnel/internal/httpserver/requestctx"
)

// ContentNegotiation sets requestctx.RenderKind from the Accept header (default HTML).
func ContentNegotiation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		kind := requestctx.RenderHTML
		if prefersJSON(r) {
			kind = requestctx.RenderJSON
		}
		next.ServeHTTP(w, r.WithContext(requestctx.WithRender(r.Context(), kind)))
	})
}

func prefersJSON(r *http.Request) bool {
	a := strings.TrimSpace(r.Header.Get("Accept"))
	if a == "" {
		return false
	}
	first := strings.TrimSpace(strings.Split(a, ",")[0])
	return strings.HasPrefix(first, "application/json")
}
