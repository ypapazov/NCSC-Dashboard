package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"fresnel/internal/config"
)

type tokenResponse struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	IDToken          string `json:"id_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	TokenType        string `json:"token_type"`
}

// ExchangeAuthorizationCode trades an auth code for tokens.
func ExchangeAuthorizationCode(ctx context.Context, cfg *config.Config, code string) (access, refresh, id string, accessExpiry time.Time, err error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", cfg.KeycloakClientID)
	form.Set("client_secret", cfg.KeycloakClientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", cfg.RedirectURI())
	return postToken(ctx, cfg, form)
}

// RefreshTokens obtains a new access token using a refresh token.
func RefreshTokens(ctx context.Context, cfg *config.Config, refreshToken string) (access, refresh, id string, accessExpiry time.Time, err error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", cfg.KeycloakClientID)
	form.Set("client_secret", cfg.KeycloakClientSecret)
	form.Set("refresh_token", refreshToken)
	return postToken(ctx, cfg, form)
}

func postToken(ctx context.Context, cfg *config.Config, form url.Values) (access, refresh, id string, accessExpiry time.Time, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.TokenEndpoint(), strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", "", time.Time{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", "", time.Time{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", "", "", time.Time{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", "", time.Time{}, fmt.Errorf("token endpoint: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", "", "", time.Time{}, err
	}
	exp := time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	if tr.ExpiresIn <= 0 {
		exp = time.Now().Add(10 * time.Minute)
	}
	return tr.AccessToken, tr.RefreshToken, tr.IDToken, exp, nil
}
