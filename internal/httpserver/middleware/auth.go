package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"fresnel/internal/config"
	"fresnel/internal/httpserver/requestctx"
	"fresnel/internal/oauth"
	"fresnel/internal/storage/postgres"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Cookie names for OIDC tokens (httpOnly).
const (
	CookieAccess    = "access_token"
	CookieRefresh   = "refresh_token"
	CookieIDToken   = "id_token"
	CookieActiveOrg = "active_org"
)

// OIDC validates access tokens (cookie or Authorization Bearer), refreshes when expired, and loads AuthContext.
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

		raw := bearerOrCookie(r)
		if raw == "" {
			o.unauthenticated(w, r)
			return
		}

		claims, err := oauth.VerifyAccessToken(r.Context(), raw, o.Cfg.AllowedTokenIssuers(), o.JWKS)
		if err != nil {
			rt, err2 := r.Cookie(CookieRefresh)
			if err2 != nil || rt.Value == "" {
				o.unauthenticated(w, r)
				return
			}
			naccess, nrefresh, nid, _, err3 := oauth.RefreshTokens(r.Context(), o.Cfg, rt.Value)
			if err3 != nil {
				o.unauthenticated(w, r)
				return
			}
			SetAuthCookies(w, o.Cfg, naccess, nrefresh, nid)
			raw = naccess
			claims, err = oauth.VerifyAccessToken(r.Context(), raw, o.Cfg.AllowedTokenIssuers(), o.JWKS)
			if err != nil {
				o.unauthenticated(w, r)
				return
			}
		}

		auth, err := postgres.LoadAuthContext(r.Context(), o.Pool, claims)
		if err != nil {
			if errors.Is(err, postgres.ErrNotRegistered) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden", "message": "user not registered in Fresnel"})
				return
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if c, err := r.Cookie(CookieActiveOrg); err == nil && c.Value != "" {
			if id, err := uuid.Parse(c.Value); err == nil {
				all := append([]uuid.UUID{auth.PrimaryOrgID}, auth.OrgMemberships...)
				for _, m := range all {
					if m == id {
						auth.ActiveOrgContext = id
						break
					}
				}
			}
		}

		ctx := requestctx.WithRawAccessToken(requestctx.WithAuth(r.Context(), auth), raw)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func isPublicPath(p string) bool {
	switch p {
	case "/api/v1/health", "/auth/callback", "/auth/login", "/auth/logout", "/favicon.ico":
		return true
	}
	return strings.HasPrefix(p, "/static/")
}

func bearerOrCookie(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) > 7 && strings.EqualFold(h[:7], "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	c, err := r.Cookie(CookieAccess)
	if err != nil || c.Value == "" {
		return ""
	}
	return c.Value
}

// SetAuthCookies sets OIDC token cookies after login or refresh.
func SetAuthCookies(w http.ResponseWriter, cfg *config.Config, access, refresh, idtok string) {
	same := http.SameSiteStrictMode
	http.SetCookie(w, &http.Cookie{
		Name:     CookieAccess,
		Value:    access,
		Path:     "/",
		HttpOnly: true,
		Secure:   cfg.CookieSecure,
		SameSite: same,
		MaxAge:   600,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     CookieRefresh,
		Value:    refresh,
		Path:     "/",
		HttpOnly: true,
		Secure:   cfg.CookieSecure,
		SameSite: same,
		MaxAge:   8 * 3600,
	})
	if idtok != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     CookieIDToken,
			Value:    idtok,
			Path:     "/",
			HttpOnly: true,
			Secure:   cfg.CookieSecure,
			SameSite: same,
			MaxAge:   8 * 3600,
		})
	}
}

func (o *OIDC) unauthenticated(w http.ResponseWriter, r *http.Request) {
	if wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}
	next := "/"
	if r.URL.RequestURI() != "" {
		next = r.URL.RequestURI()
	}
	u := "/auth/login?next=" + url.QueryEscape(next)
	http.Redirect(w, r, u, http.StatusFound)
}

func wantsJSON(r *http.Request) bool {
	a := strings.ToLower(r.Header.Get("Accept"))
	return strings.Contains(a, "application/json") && !strings.Contains(a, "text/html")
}

// ClearAuthCookies clears token cookies (logout).
func ClearAuthCookies(w http.ResponseWriter, cfg *config.Config) {
	same := http.SameSiteStrictMode
	clear := func(name string) {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   cfg.CookieSecure,
			SameSite: same,
			MaxAge:   -1,
		})
	}
	clear(CookieAccess)
	clear(CookieRefresh)
	clear(CookieIDToken)
}
