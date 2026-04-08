package httpserver

import (
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"fresnel/internal/config"
	httphandlers "fresnel/internal/httpserver/handlers"
)

// NewRouter registers application routes (expanded in later milestones).
func NewRouter(log *slog.Logger, cfg *config.Config, pool *pgxpool.Pool) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/health", httphandlers.Health(log, pool, cfg.KeycloakIssuer))
	return mux
}
