package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Health returns database and Keycloak connectivity.
func Health(log *slog.Logger, pool *pgxpool.Pool, keycloakIssuer string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		dbOK := true
		if err := pool.Ping(ctx); err != nil {
			log.Error("health db ping", "err", err)
			dbOK = false
		}

		kcOK := checkKeycloak(ctx, keycloakIssuer)

		w.Header().Set("Content-Type", "application/json")
		if !dbOK || !kcOK {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":    "unhealthy",
				"database": map[bool]string{true: "ok", false: "unreachable"}[dbOK],
				"keycloak": map[bool]string{true: "ok", false: "unreachable"}[kcOK],
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
}

func checkKeycloak(ctx context.Context, issuer string) bool {
	if issuer == "" {
		return false
	}
	u := issuer
	if u[len(u)-1] != '/' {
		u += "/"
	}
	u += ".well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
