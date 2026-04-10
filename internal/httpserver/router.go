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
	"fresnel/internal/service"
	"fresnel/static"
)

// Services bundles all application services for the HTTP layer.
type Services struct {
	Events        *service.EventService
	StatusReports *service.StatusReportService
	Campaigns     *service.CampaignService
	Sectors       *service.SectorService
	Orgs          *service.OrganizationService
	Users         *service.UserService
	Correlations  *service.CorrelationService
	Attachments   *service.AttachmentService
	Audit         *service.AuditService
	Dashboard     *service.DashboardService
}

// NewRouter registers routes and applies the middleware chain:
// logging → audit context → OIDC → cedar gate → content negotiation → mux.
func NewRouter(log *slog.Logger, cfg *config.Config, pool *pgxpool.Pool, svc Services) (http.Handler, error) {
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

	// --- Handlers ---
	dashboardH := httphandlers.NewDashboardHandler(svc.Dashboard, tmpl)
	eventH := httphandlers.NewEventHandler(svc.Events, tmpl)
	statusReportH := httphandlers.NewStatusReportHandler(svc.StatusReports, tmpl)
	campaignH := httphandlers.NewCampaignHandler(svc.Campaigns, tmpl)
	sectorH := httphandlers.NewSectorHandler(svc.Sectors, tmpl)
	orgH := httphandlers.NewOrgHandler(svc.Orgs, tmpl)
	userH := httphandlers.NewUserHandler(svc.Users, tmpl)
	corrH := httphandlers.NewCorrelationHandler(svc.Correlations, tmpl)
	attachH := httphandlers.NewAttachmentHandler(svc.Attachments)
	auditH := httphandlers.NewAuditHandler(svc.Audit, tmpl)

	mux := http.NewServeMux()

	// --- Public routes ---
	mux.Handle("GET /api/v1/health", httphandlers.Health(log, pool, cfg.KeycloakIssuer))

	st, err := fs.Sub(static.Files, ".")
	if err != nil {
		return nil, err
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(st))))

	// --- Nav ---
	mux.Handle("GET /api/v1/nav", httphandlers.Nav(tmpl))

	// --- Dashboard ---
	mux.HandleFunc("GET /api/v1/dashboard", dashboardH.Get)

	// --- Events ---
	mux.HandleFunc("GET /api/v1/events", eventH.List)
	mux.HandleFunc("GET /api/v1/events/new", eventH.Form)
	mux.HandleFunc("POST /api/v1/events", eventH.Create)
	mux.HandleFunc("GET /api/v1/events/{id}", eventH.Get)
	mux.HandleFunc("GET /api/v1/events/{id}/edit", eventH.Form)
	mux.HandleFunc("PUT /api/v1/events/{id}", eventH.Update)
	mux.HandleFunc("DELETE /api/v1/events/{id}", eventH.Delete)
	mux.HandleFunc("GET /api/v1/events/{id}/updates", eventH.ListUpdates)
	mux.HandleFunc("POST /api/v1/events/{id}/updates", eventH.CreateUpdate)
	mux.HandleFunc("GET /api/v1/events/{id}/revisions", eventH.ListRevisions)

	// Event attachments
	mux.HandleFunc("GET /api/v1/events/{id}/attachments", attachH.ListByEvent)
	mux.HandleFunc("POST /api/v1/events/{id}/attachments", attachH.Upload)

	// Event correlations
	mux.HandleFunc("GET /api/v1/events/{id}/correlations", corrH.ListByEvent)
	mux.HandleFunc("POST /api/v1/events/{id}/correlations", corrH.CreateCorrelation)

	// Event relationships
	mux.HandleFunc("GET /api/v1/events/{id}/relationships", corrH.ListRelationships)
	mux.HandleFunc("POST /api/v1/events/{id}/relationships", corrH.CreateRelationship)

	// --- Status Reports ---
	mux.HandleFunc("GET /api/v1/status-reports", statusReportH.List)
	mux.HandleFunc("POST /api/v1/status-reports", statusReportH.Create)
	mux.HandleFunc("GET /api/v1/status-reports/{id}", statusReportH.Get)
	mux.HandleFunc("PUT /api/v1/status-reports/{id}", statusReportH.Update)
	mux.HandleFunc("DELETE /api/v1/status-reports/{id}", statusReportH.Delete)

	// --- Campaigns ---
	mux.HandleFunc("GET /api/v1/campaigns", campaignH.List)
	mux.HandleFunc("POST /api/v1/campaigns", campaignH.Create)
	mux.HandleFunc("GET /api/v1/campaigns/{id}", campaignH.Get)
	mux.HandleFunc("PUT /api/v1/campaigns/{id}", campaignH.Update)
	mux.HandleFunc("GET /api/v1/campaigns/{id}/events", campaignH.GetLinkedEvents)
	mux.HandleFunc("POST /api/v1/campaigns/{id}/events", campaignH.LinkEvent)
	mux.HandleFunc("DELETE /api/v1/campaigns/{id}/events/{eventId}", campaignH.UnlinkEvent)

	// --- Sectors ---
	mux.HandleFunc("GET /api/v1/sectors", sectorH.List)
	mux.HandleFunc("POST /api/v1/sectors", sectorH.Create)
	mux.HandleFunc("GET /api/v1/sectors/{id}", sectorH.Get)
	mux.HandleFunc("PUT /api/v1/sectors/{id}", sectorH.Update)
	mux.HandleFunc("DELETE /api/v1/sectors/{id}", sectorH.Delete)
	mux.HandleFunc("GET /api/v1/sectors/{id}/children", sectorH.GetChildren)

	// --- Organizations ---
	mux.HandleFunc("GET /api/v1/orgs", orgH.List)
	mux.HandleFunc("POST /api/v1/orgs", orgH.Create)
	mux.HandleFunc("GET /api/v1/orgs/{id}", orgH.Get)
	mux.HandleFunc("PUT /api/v1/orgs/{id}", orgH.Update)
	mux.HandleFunc("DELETE /api/v1/orgs/{id}", orgH.Delete)

	// --- Users ---
	mux.HandleFunc("GET /api/v1/users", userH.List)
	mux.HandleFunc("POST /api/v1/users", userH.Create)
	mux.HandleFunc("GET /api/v1/users/me", userH.GetMe)
	mux.HandleFunc("GET /api/v1/users/{id}", userH.Get)
	mux.HandleFunc("PUT /api/v1/users/{id}", userH.Update)
	mux.HandleFunc("POST /api/v1/users/{id}/roles", userH.AssignRole)
	mux.HandleFunc("DELETE /api/v1/users/{id}/roles", userH.RevokeRole)

	// --- Audit ---
	mux.HandleFunc("GET /api/v1/audit", auditH.List)

	// --- Federation (stub) ---
	mux.HandleFunc("GET /api/v1/federation/", httphandlers.FederationStub)
	mux.HandleFunc("POST /api/v1/federation/", httphandlers.FederationStub)

	// --- Catch-all: app shell ---
	mux.Handle("GET /{path...}", httphandlers.Shell(tmpl, cfg))

	// Middleware chain: outermost wraps first
	auditCtx := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(service.ContextWithRequest(r.Context(), r)))
		})
	}

	chain := middleware.RequestLogger(log)(
		auditCtx(
			oidc.Handler(
				middleware.CedarGate(
					middleware.ContentNegotiation(mux),
				),
			),
		),
	)
	return chain, nil
}
