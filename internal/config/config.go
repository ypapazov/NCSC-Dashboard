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
	ListenAddr  string
	DatabaseURL string

	// KeycloakIssuer is the internal issuer URL for server-to-server
	// communication (JWKS fetch, token validation).
	KeycloakIssuer   string
	KeycloakClientID string

	// KeycloakExternalURL is the realm URL reachable by the browser
	// (e.g. http://localhost:8081/realms/fresnel). Used to configure
	// keycloak-js on the client. Falls back to KeycloakIssuer when not set.
	KeycloakExternalURL string

	AppPublicURL   string // e.g. https://localhost
	ClamAVAddress  string // TCP address (host:port) for clamd; empty disables scanning
	SMTPHost       string
	SMTPPort      int
	AttachmentDir string
	// DashboardCacheTTL is the default TTL for computed dashboard node status.
	DashboardCacheTTL time.Duration
}

func getenv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
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
	return &Config{
		ListenAddr:          getenv("LISTEN_ADDR", ":8080"),
		DatabaseURL:         getenv("DATABASE_URL", ""),
		KeycloakIssuer:      getenv("KEYCLOAK_ISSUER", ""),
		KeycloakClientID:    getenv("KEYCLOAK_CLIENT_ID", ""),
		KeycloakExternalURL: getenv("KEYCLOAK_EXTERNAL_URL", ""),
		AppPublicURL:        getenv("APP_PUBLIC_URL", "https://localhost"),
		ClamAVAddress:       getenv("CLAMAV_ADDRESS", ""),
		SMTPHost:            getenv("SMTP_HOST", ""),
		SMTPPort:            smtpPort,
		AttachmentDir:       getenv("ATTACHMENT_DIR", "/var/lib/fresnel/attachments"),
		DashboardCacheTTL:   cacheTTL,
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

// KeycloakBrowserURL returns the Keycloak base URL (without /realms/...)
// for configuring keycloak-js on the client side.
func (c *Config) KeycloakBrowserURL() string {
	base := c.keycloakBrowserBase()
	if idx := strings.Index(base, "/realms/"); idx != -1 {
		return base[:idx]
	}
	return base
}

// AllowedTokenIssuers lists issuer values Keycloak may put in JWT `iss`
// (browser vs internal Docker URL).
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
	return nil
}
