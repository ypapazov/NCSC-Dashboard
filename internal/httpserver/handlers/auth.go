package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"fresnel/internal/config"
	"fresnel/internal/httpserver/csrf"
	"fresnel/internal/httpserver/middleware"
	"fresnel/internal/httpserver/requestctx"
	"fresnel/internal/oauth"
)

const (
	cookieOAuthState  = "oidc_state"
	cookieOAuthReturn = "oidc_return"
)

// Login redirects the browser to Keycloak (OIDC authorization code flow).
func Login(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		next := safeInternalPath(r.URL.Query().Get("next"))
		st, err := randomState()
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		same := http.SameSiteLaxMode
		http.SetCookie(w, &http.Cookie{
			Name:     cookieOAuthState,
			Value:    st,
			Path:     "/",
			HttpOnly: true,
			Secure:   cfg.CookieSecure,
			SameSite: same,
			MaxAge:   600,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     cookieOAuthReturn,
			Value:    next,
			Path:     "/",
			HttpOnly: true,
			Secure:   cfg.CookieSecure,
			SameSite: same,
			MaxAge:   600,
		})
		q := url.Values{}
		q.Set("client_id", cfg.KeycloakClientID)
		q.Set("redirect_uri", cfg.RedirectURI())
		q.Set("response_type", "code")
		q.Set("scope", "openid profile email")
		q.Set("state", st)
		authURL := cfg.AuthEndpoint() + "?" + q.Encode()
		http.Redirect(w, r, authURL, http.StatusFound)
	}
}

// Callback completes OIDC login: exchanges the code and sets token cookies.
func Callback(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		q := r.URL.Query()
		if errMsg := q.Get("error"); errMsg != "" {
			http.Error(w, "oidc error: "+errMsg, http.StatusBadRequest)
			return
		}
		code := q.Get("code")
		state := q.Get("state")
		if code == "" || state == "" {
			http.Error(w, "missing code or state", http.StatusBadRequest)
			return
		}
		sc, err := r.Cookie(cookieOAuthState)
		if err != nil || sc.Value == "" || sc.Value != state {
			http.Error(w, "invalid state", http.StatusBadRequest)
			return
		}
		ret, err := r.Cookie(cookieOAuthReturn)
		next := "/"
		if err == nil && ret.Value != "" {
			next = safeInternalPath(ret.Value)
		}

		access, refresh, idtok, _, err := oauth.ExchangeAuthorizationCode(r.Context(), cfg, code)
		if err != nil {
			http.Error(w, "token exchange failed", http.StatusBadGateway)
			return
		}

		http.SetCookie(w, &http.Cookie{Name: cookieOAuthState, MaxAge: -1, Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: cookieOAuthReturn, MaxAge: -1, Path: "/"})

		middleware.SetAuthCookies(w, cfg, access, refresh, idtok)
		http.Redirect(w, r, next, http.StatusFound)
	}
}

// Logout clears cookies and redirects to Keycloak logout.
func Logout(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idt, _ := r.Cookie(middleware.CookieIDToken)
		middleware.ClearAuthCookies(w, cfg)
		u, err := url.Parse(cfg.LogoutEndpoint())
		if err != nil {
			http.Redirect(w, r, cfg.AppPublicURL, http.StatusFound)
			return
		}
		q := u.Query()
		if idt != nil && idt.Value != "" {
			q.Set("id_token_hint", idt.Value)
		}
		q.Set("post_logout_redirect_uri", strings.TrimSuffix(cfg.AppPublicURL, "/")+"/")
		u.RawQuery = q.Encode()
		http.Redirect(w, r, u.String(), http.StatusFound)
	}
}

// CSRFToken returns the HMAC CSRF token for HTMX / forms (requires auth middleware).
func CSRFToken(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw := requestctx.RawAccessTokenFrom(r.Context())
		if raw == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		tok := csrf.Token(cfg.HMACSecret, raw)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"csrf_token": tok})
	}
}

func safeInternalPath(p string) string {
	if p == "" || !strings.HasPrefix(p, "/") || strings.HasPrefix(p, "//") || strings.Contains(p, "\r") || strings.Contains(p, "\n") {
		return "/"
	}
	return p
}

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
