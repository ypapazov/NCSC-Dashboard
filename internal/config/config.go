package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds server and integration settings loaded from the environment.
type Config struct {
	ListenAddr       string
	DatabaseURL      string
	KeycloakIssuer   string
	KeycloakClientID string
	KeycloakClientSecret string
	// KeycloakBaseURL is the origin only (scheme + host + optional port), used for token endpoint.
	KeycloakBaseURL string
	AppPublicURL    string // e.g. https://localhost — for redirects and cookie domain hints
	ClamAVSocket    string
	SMTPHost        string
	SMTPPort        int
	HMACSecret      []byte
	AttachmentDir   string
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
	return &Config{
		ListenAddr:           getenv("LISTEN_ADDR", ":8080"),
		DatabaseURL:          getenv("DATABASE_URL", ""),
		KeycloakIssuer:       getenv("KEYCLOAK_ISSUER", ""),
		KeycloakClientID:     getenv("KEYCLOAK_CLIENT_ID", ""),
		KeycloakClientSecret: getenv("KEYCLOAK_CLIENT_SECRET", ""),
		KeycloakBaseURL:      getenv("KEYCLOAK_BASE_URL", ""),
		AppPublicURL:         getenv("APP_PUBLIC_URL", "https://localhost"),
		ClamAVSocket:         getenv("CLAMAV_SOCKET", "/var/run/clamav/clamd.sock"),
		SMTPHost:               getenv("SMTP_HOST", ""),
		SMTPPort:               smtpPort,
		HMACSecret:             hmac,
		AttachmentDir:          getenv("ATTACHMENT_DIR", "/var/lib/fresnel/attachments"),
		DashboardCacheTTL:      cacheTTL,
	}, nil
}

func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.KeycloakIssuer == "" {
		return fmt.Errorf("KEYCLOAK_ISSUER is required")
	}
	return nil
}
