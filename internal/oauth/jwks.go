package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-jose/go-jose/v4"
)

// JWKS holds a cached Keycloak JWKS.
type JWKS struct {
	URL    string
	Client *http.Client
	TTL    time.Duration

	mu  sync.RWMutex
	set jose.JSONWebKeySet
	at  time.Time
}

// KeySet returns a cached JSONWebKeySet, refreshing when stale.
func (j *JWKS) KeySet(ctx context.Context) (*jose.JSONWebKeySet, error) {
	j.mu.RLock()
	if time.Since(j.at) < j.TTL && len(j.set.Keys) > 0 {
		defer j.mu.RUnlock()
		return &j.set, nil
	}
	j.mu.RUnlock()

	j.mu.Lock()
	defer j.mu.Unlock()
	if time.Since(j.at) < j.TTL && len(j.set.Keys) > 0 {
		return &j.set, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, j.URL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := j.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jwks: status %d", resp.StatusCode)
	}
	var set jose.JSONWebKeySet
	if err := json.NewDecoder(resp.Body).Decode(&set); err != nil {
		return nil, err
	}
	j.set = set
	j.at = time.Now()
	return &j.set, nil
}
