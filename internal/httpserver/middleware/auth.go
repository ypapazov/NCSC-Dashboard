package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"fresnel/internal/config"
	"fresnel/internal/httpserver/requestctx"
	"fresnel/internal/oauth"
	"fresnel/internal/storage/postgres"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OIDC validates Bearer access tokens and loads AuthContext from the database.
// The browser handles the OIDC flow via keycloak-js; the server is a pure
// resource server that validates JWTs.
type OIDC struct {
	Cfg  *config.Config
	Pool *pgxpool.Pool
	JWKS *oauth.JWKS
}

func (o *OIDC) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		raw := bearerToken(r)
		if raw == "" {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				jsonUnauthorized(w)
				return
			}
			// Non-API path without Bearer token: pass through so the
			// shell handler can serve the keycloak-js bootstrap page.
			next.ServeHTTP(w, r)
			return
		}

		claims, err := oauth.VerifyAccessToken(r.Context(), raw, o.Cfg.AllowedTokenIssuers(), o.JWKS)
		if err != nil {
			jsonUnauthorized(w)
			return
		}

		auth, err := postgres.LoadAuthContext(r.Context(), o.Pool, claims)
		if err != nil {
			if errors.Is(err, postgres.ErrNotRegistered) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error":   "forbidden",
					"message": "user not registered in Fresnel",
				})
				return
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if orgID := activeOrgHeader(r); orgID != uuid.Nil {
			all := append([]uuid.UUID{auth.PrimaryOrgID}, auth.OrgMemberships...)
			for _, m := range all {
				if m == orgID {
					auth.ActiveOrgContext = orgID
					break
				}
			}
		}

		ctx := requestctx.WithAuth(r.Context(), auth)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func isPublicPath(p string) bool {
	switch p {
	case "/api/v1/health", "/favicon.ico":
		return true
	}
	return strings.HasPrefix(p, "/static/")
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) > 7 && strings.EqualFold(h[:7], "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

// activeOrgHeader reads the X-Fresnel-Org header set by the client-side
// org context selector.
func activeOrgHeader(r *http.Request) uuid.UUID {
	v := r.Header.Get("X-Fresnel-Org")
	if v == "" {
		return uuid.Nil
	}
	id, err := uuid.Parse(v)
	if err != nil {
		return uuid.Nil
	}
	return id
}

func jsonUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
}
