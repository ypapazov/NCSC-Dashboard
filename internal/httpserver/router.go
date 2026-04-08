package httpserver

import (
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"fresnel/internal/config"
	httphandlers "fresnel/internal/httpserver/handlers"
	"fresnel/internal/httpserver/middleware"
	apptemplates "fresnel/internal/httpserver/templates"
	"fresnel/internal/oauth"
	"fresnel/static"
)

// NewRouter registers routes and applies the M1 middleware chain:
// logging → OIDC → CSRF → content negotiation → mux.
func NewRouter(log *slog.Logger, cfg *config.Config, pool *pgxpool.Pool) (http.Handler, error) {
	tmpl, err := apptemplates.Parse()
	if err != nil {
		return nil, err
	}

	jwks := &oauth.JWKS{
		URL:    cfg.JWKSURL(),
		Client: http.DefaultClient,
		TTL:    15 * time.Minute,
	}
	oidc := &middleware.OIDC{Cfg: cfg, Pool: pool, JWKS: jwks}

	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/health", httphandlers.Health(log, pool, cfg.KeycloakIssuer))

	st, err := fs.Sub(static.Files, ".")
	if err != nil {
		return nil, err
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(st))))

	mux.Handle("GET /auth/login", httphandlers.Login(cfg))
	mux.Handle("GET /auth/callback", httphandlers.Callback(cfg))
	mux.Handle("GET /auth/logout", httphandlers.Logout(cfg))
	mux.Handle("GET /auth/csrf", httphandlers.CSRFToken(cfg))

	dash := httphandlers.Dashboard(tmpl)
	mux.Handle("GET /", dash)
	mux.Handle("GET /dashboard", dash)

	chain := middleware.RequestLogger(log)(oidc.Handler(middleware.CSRF(cfg)(middleware.ContentNegotiation(mux))))
	return chain, nil
}
