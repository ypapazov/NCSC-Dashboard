package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds server and integration settings loaded from the environment.
type Config struct {
	ListenAddr       string
	DatabaseURL      string
	KeycloakIssuer   string
	KeycloakClientID string
	KeycloakClientSecret string
	// KeycloakExternalURL is the realm URL reachable by the browser (e.g. http://localhost:8081/realms/fresnel).
	// Used for authorize and logout redirects. Falls back to KeycloakIssuer if not set
	// (appropriate when the same URL is reachable by both the app and the browser, e.g. production).
	KeycloakExternalURL string
	// KeycloakTokenURL is the full OIDC token endpoint URL for POST (code exchange, refresh).
	// Optional. If empty, derived from KeycloakIssuer. In Docker, set to the same host the browser
	// uses for Keycloak (e.g. http://host.docker.internal:8081/.../token) so refresh matches token iss.
	KeycloakTokenURL string
	AppPublicURL    string // e.g. https://localhost — for redirects and cookie domain hints
	ClamAVSocket    string
	SMTPHost        string
	SMTPPort        int
	HMACSecret      []byte
	AttachmentDir   string
	// DashboardCacheTTL is the default TTL for computed dashboard node status.
	DashboardCacheTTL time.Duration
	// CookieSecure sets the Secure flag on auth cookies (use true behind HTTPS).
	CookieSecure bool
}

func getenv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func mustSecret(key string) ([]byte, error) {
	s := os.Getenv(key)
	if s == "" {
		return nil, fmt.Errorf("%s must be set to a non-empty secret", key)
	}
	if len(s) < 32 {
		return nil, fmt.Errorf("%s must be at least 32 bytes", key)
	}
	return []byte(s), nil
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	hmac, err := mustSecret("HMAC_SECRET")
	if err != nil {
		return nil, err
	}
	smtpPort := 587
	if p := os.Getenv("SMTP_PORT"); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("SMTP_PORT: %w", err)
		}
		smtpPort = n
	}
	cacheTTL := 60 * time.Second
	if s := os.Getenv("DASHBOARD_CACHE_TTL_SEC"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil {
			return nil, fmt.Errorf("DASHBOARD_CACHE_TTL_SEC: %w", err)
		}
		cacheTTL = time.Duration(n) * time.Second
	}
	cookieSecure := strings.HasPrefix(getenv("APP_PUBLIC_URL", "https://localhost"), "https://")
	if v := os.Getenv("COOKIE_SECURE"); v != "" {
		cookieSecure = v == "1" || v == "true"
	}
	return &Config{
		ListenAddr:           getenv("LISTEN_ADDR", ":8080"),
		DatabaseURL:          getenv("DATABASE_URL", ""),
		KeycloakIssuer:       getenv("KEYCLOAK_ISSUER", ""),
		KeycloakClientID:     getenv("KEYCLOAK_CLIENT_ID", ""),
		KeycloakClientSecret: getenv("KEYCLOAK_CLIENT_SECRET", ""),
		KeycloakExternalURL:  getenv("KEYCLOAK_EXTERNAL_URL", ""),
		KeycloakTokenURL:     getenv("KEYCLOAK_TOKEN_URL", ""),
		AppPublicURL:         getenv("APP_PUBLIC_URL", "https://localhost"),
		ClamAVSocket:         getenv("CLAMAV_SOCKET", "/var/run/clamav/clamd.sock"),
		SMTPHost:               getenv("SMTP_HOST", ""),
		SMTPPort:               smtpPort,
		HMACSecret:             hmac,
		AttachmentDir:          getenv("ATTACHMENT_DIR", "/var/lib/fresnel/attachments"),
		DashboardCacheTTL:      cacheTTL,
		CookieSecure:           cookieSecure,
	}, nil
}

// keycloakBrowserBase returns the realm URL reachable by the browser.
// Falls back to KeycloakIssuer when KeycloakExternalURL is not set.
func (c *Config) keycloakBrowserBase() string {
	if c.KeycloakExternalURL != "" {
		return strings.TrimSuffix(c.KeycloakExternalURL, "/")
	}
	return strings.TrimSuffix(c.KeycloakIssuer, "/")
}

// AuthEndpoint returns the OIDC authorization URL (browser-facing).
func (c *Config) AuthEndpoint() string {
	return c.keycloakBrowserBase() + "/protocol/openid-connect/auth"
}

// LogoutEndpoint returns the RP-initiated logout URL (browser-facing).
func (c *Config) LogoutEndpoint() string {
	return c.keycloakBrowserBase() + "/protocol/openid-connect/logout"
}

// TokenEndpoint returns the token POST URL (code exchange, refresh).
func (c *Config) TokenEndpoint() string {
	if c.KeycloakTokenURL != "" {
		return c.KeycloakTokenURL
	}
	return strings.TrimSuffix(c.KeycloakIssuer, "/") + "/protocol/openid-connect/token"
}

// AllowedTokenIssuers lists issuer values Keycloak may put in JWT `iss` (browser vs internal Docker URL).
func (c *Config) AllowedTokenIssuers() []string {
	seen := map[string]bool{}
	var out []string
	add := func(s string) {
		s = strings.TrimSuffix(strings.TrimSpace(s), "/")
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		out = append(out, s)
	}
	add(c.KeycloakIssuer)
	add(c.KeycloakExternalURL)
	return out
}

// JWKSURL returns the JWKS document URL (server-to-server, uses internal issuer URL).
func (c *Config) JWKSURL() string {
	return strings.TrimSuffix(c.KeycloakIssuer, "/") + "/protocol/openid-connect/certs"
}

// RedirectURI is the registered OIDC redirect URI for this app.
func (c *Config) RedirectURI() string {
	return strings.TrimSuffix(c.AppPublicURL, "/") + "/auth/callback"
}

func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.KeycloakIssuer == "" {
		return fmt.Errorf("KEYCLOAK_ISSUER is required")
	}
	if c.KeycloakClientID == "" {
		return fmt.Errorf("KEYCLOAK_CLIENT_ID is required")
	}
	if c.KeycloakClientSecret == "" {
		return fmt.Errorf("KEYCLOAK_CLIENT_SECRET is required")
	}
	return nil
}
