package middleware

import (
	"net/http"

	"fresnel/internal/i18n"
)

// Locale resolves the user's preferred language from the fresnel_lang cookie
// or the Accept-Language header and stores it in the request context.
func Locale(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		locale := i18n.ResolveLocale(r)
		next.ServeHTTP(w, r.WithContext(i18n.WithLocale(r.Context(), locale)))
	})
}
