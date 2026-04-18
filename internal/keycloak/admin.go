package keycloak

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// AdminClient manages users in a Keycloak realm via the Admin REST API.
type AdminClient struct {
	baseURL  string // e.g. http://keycloak:8080
	realm    string
	user     string
	password string

	mu    sync.Mutex
	token string
	exp   time.Time

	hc *http.Client
}

// NewAdminClient creates a client. issuerURL is the full realm issuer
// (e.g. http://keycloak:8080/realms/fresnel); admin credentials are the
// bootstrap admin from KC_BOOTSTRAP_ADMIN_USERNAME/PASSWORD.
func NewAdminClient(issuerURL, adminUser, adminPassword string) *AdminClient {
	base := issuerURL
	if idx := strings.Index(base, "/realms/"); idx != -1 {
		base = base[:idx]
	}
	realm := "fresnel"
	if idx := strings.LastIndex(issuerURL, "/realms/"); idx != -1 {
		realm = issuerURL[idx+len("/realms/"):]
		realm = strings.TrimRight(realm, "/")
	}
	return &AdminClient{
		baseURL:  strings.TrimRight(base, "/"),
		realm:    realm,
		user:     adminUser,
		password: adminPassword,
		hc:       &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *AdminClient) accessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token != "" && time.Now().Before(c.exp) {
		return c.token, nil
	}

	data := url.Values{
		"grant_type": {"password"},
		"client_id":  {"admin-cli"},
		"username":   {c.user},
		"password":   {c.password},
	}
	tokenURL := c.baseURL + "/realms/master/protocol/openid-connect/token"
	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("keycloak token: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("keycloak token %d: %s", resp.StatusCode, body)
	}
	var tok struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return "", err
	}
	c.token = tok.AccessToken
	c.exp = time.Now().Add(time.Duration(tok.ExpiresIn-30) * time.Second)
	return c.token, nil
}

// CreateUser provisions a user in Keycloak and sets their password.
// Returns the Keycloak user ID (sub).
func (c *AdminClient) CreateUser(ctx context.Context, email, firstName, lastName, password string) (string, error) {
	token, err := c.accessToken(ctx)
	if err != nil {
		return "", err
	}

	payload := map[string]any{
		"username":      email,
		"email":         email,
		"emailVerified": true,
		"enabled":       true,
		"firstName":     firstName,
		"lastName":      lastName,
		"credentials": []map[string]any{{
			"type":      "password",
			"value":     password,
			"temporary": false,
		}},
	}
	body, _ := json.Marshal(payload)

	usersURL := fmt.Sprintf("%s/admin/realms/%s/users", c.baseURL, c.realm)
	req, err := http.NewRequestWithContext(ctx, "POST", usersURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("keycloak create user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		rb, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("keycloak create user %d: %s", resp.StatusCode, rb)
	}

	loc := resp.Header.Get("Location")
	if loc == "" {
		return "", fmt.Errorf("keycloak: no Location header in response")
	}
	parts := strings.Split(strings.TrimRight(loc, "/"), "/")
	return parts[len(parts)-1], nil
}
