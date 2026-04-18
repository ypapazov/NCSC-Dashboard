package httpserver

import (
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"fresnel/internal/authz"
	"fresnel/internal/config"
	httphandlers "fresnel/internal/httpserver/handlers"
	"fresnel/internal/httpserver/middleware"
	"fresnel/internal/oauth"
	"fresnel/internal/service"
	"fresnel/internal/storage"
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

// Lookups provides direct store access for view-layer data enrichment
// (name resolution, authz checks, TLP recipient lookups).
type Lookups struct {
	Orgs    storage.OrganizationStore
	Sectors storage.SectorStore
	Users   storage.UserStore
	Roles   storage.RoleStore
	TLPRed  storage.TLPRedStore
	Authz   authz.Authorizer
}

// NewRouter registers routes and applies the middleware chain:
// logging → audit context → OIDC → cedar gate → content negotiation → mux.
func NewRouter(log *slog.Logger, cfg *config.Config, pool *pgxpool.Pool, svc Services, lk Lookups) (http.Handler, error) {
	jwks := &oauth.JWKS{
		URL:    cfg.JWKSURL(),
		Client: http.DefaultClient,
		TTL:    15 * time.Minute,
	}
	oidc := &middleware.OIDC{Cfg: cfg, Pool: pool, JWKS: jwks}

	hlk := httphandlers.Lookups{
		Orgs:    lk.Orgs,
		Sectors: lk.Sectors,
		Users:   lk.Users,
		Roles:   lk.Roles,
		TLPRed:  lk.TLPRed,
		Authz:   lk.Authz,
	}

	// --- Handlers ---
	dashboardH := httphandlers.NewDashboardHandler(svc.Dashboard, svc.Events, svc.Campaigns, svc.Correlations)
	eventH := httphandlers.NewEventHandler(svc.Events, svc.Attachments, svc.Correlations, hlk)
	statusReportH := httphandlers.NewStatusReportHandler(svc.StatusReports, svc.Events, hlk)
	campaignH := httphandlers.NewCampaignHandler(svc.Campaigns, hlk)
	sectorH := httphandlers.NewSectorHandler(svc.Sectors)
	orgH := httphandlers.NewOrgHandler(svc.Orgs, svc.Sectors)
	userH := httphandlers.NewUserHandler(svc.Users, svc.Orgs, hlk)
	corrH := httphandlers.NewCorrelationHandler(svc.Correlations, svc.Events, hlk)
	attachH := httphandlers.NewAttachmentHandler(svc.Attachments)
	auditH := httphandlers.NewAuditHandler(svc.Audit, svc.Users)

	mux := http.NewServeMux()

	// --- Public routes ---
	mux.Handle("GET /api/v1/health", httphandlers.Health(log, pool, cfg.KeycloakIssuer))

	st, err := fs.Sub(static.Files, ".")
	if err != nil {
		return nil, err
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(st))))

	// --- Nav ---
	mux.Handle("GET /api/v1/nav", httphandlers.Nav())

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
	mux.HandleFunc("GET /api/v1/events/{id}/attachments/{attachmentId}", attachH.Download)
	mux.HandleFunc("DELETE /api/v1/events/{id}/attachments/{attachmentId}", attachH.Delete)

	// Event correlations
	mux.HandleFunc("GET /api/v1/events/{id}/correlations", corrH.ListByEvent)
	mux.HandleFunc("POST /api/v1/events/{id}/correlations", corrH.CreateCorrelation)

	// Event relationships
	mux.HandleFunc("GET /api/v1/events/{id}/relationships", corrH.ListRelationships)
	mux.HandleFunc("POST /api/v1/events/{id}/relationships", corrH.CreateRelationship)

	// Graph view
	mux.HandleFunc("GET /api/v1/events/{id}/graph", corrH.GraphPage)

	// --- Status Reports ---
	mux.HandleFunc("GET /api/v1/status-reports", statusReportH.List)
	mux.HandleFunc("GET /api/v1/status-reports/new", statusReportH.Form)
	mux.HandleFunc("POST /api/v1/status-reports", statusReportH.Create)
	mux.HandleFunc("GET /api/v1/status-reports/{id}", statusReportH.Get)
	mux.HandleFunc("GET /api/v1/status-reports/{id}/edit", statusReportH.Form)
	mux.HandleFunc("PUT /api/v1/status-reports/{id}", statusReportH.Update)
	mux.HandleFunc("DELETE /api/v1/status-reports/{id}", statusReportH.Delete)

	// --- Campaigns ---
	mux.HandleFunc("GET /api/v1/campaigns", campaignH.List)
	mux.HandleFunc("GET /api/v1/campaigns/new", campaignH.Form)
	mux.HandleFunc("POST /api/v1/campaigns", campaignH.Create)
	mux.HandleFunc("POST /api/v1/campaigns/from-selection", campaignH.CreateFromSelection)
	mux.HandleFunc("GET /api/v1/campaigns/{id}", campaignH.Get)
	mux.HandleFunc("GET /api/v1/campaigns/{id}/edit", campaignH.Form)
	mux.HandleFunc("PUT /api/v1/campaigns/{id}", campaignH.Update)
	mux.HandleFunc("GET /api/v1/campaigns/{id}/events", campaignH.GetLinkedEvents)
	mux.HandleFunc("POST /api/v1/campaigns/{id}/events", campaignH.LinkEvent)
	mux.HandleFunc("DELETE /api/v1/campaigns/{id}/events/{eventId}", campaignH.UnlinkEvent)

	// --- Sectors ---
	mux.HandleFunc("GET /api/v1/sectors", sectorH.List)
	mux.HandleFunc("GET /api/v1/sectors/new", sectorH.Form)
	mux.HandleFunc("POST /api/v1/sectors", sectorH.Create)
	mux.HandleFunc("GET /api/v1/sectors/{id}", sectorH.Get)
	mux.HandleFunc("GET /api/v1/sectors/{id}/edit", sectorH.Form)
	mux.HandleFunc("PUT /api/v1/sectors/{id}", sectorH.Update)
	mux.HandleFunc("DELETE /api/v1/sectors/{id}", sectorH.Delete)
	mux.HandleFunc("GET /api/v1/sectors/{id}/children", sectorH.GetChildren)

	// --- Organizations ---
	mux.HandleFunc("GET /api/v1/orgs", orgH.List)
	mux.HandleFunc("GET /api/v1/orgs/new", orgH.Form)
	mux.HandleFunc("POST /api/v1/orgs", orgH.Create)
	mux.HandleFunc("GET /api/v1/orgs/{id}", orgH.Get)
	mux.HandleFunc("GET /api/v1/orgs/{id}/edit", orgH.Form)
	mux.HandleFunc("PUT /api/v1/orgs/{id}", orgH.Update)
	mux.HandleFunc("DELETE /api/v1/orgs/{id}", orgH.Delete)

	// --- Users ---
	mux.HandleFunc("GET /api/v1/users", userH.List)
	mux.HandleFunc("GET /api/v1/users/new", userH.Form)
	mux.HandleFunc("POST /api/v1/users", userH.Create)
	mux.HandleFunc("GET /api/v1/users/me", userH.GetMe)
	mux.HandleFunc("GET /api/v1/users/{id}", userH.Get)
	mux.HandleFunc("GET /api/v1/users/{id}/edit", userH.Form)
	mux.HandleFunc("PUT /api/v1/users/{id}", userH.Update)
	mux.HandleFunc("DELETE /api/v1/users/{id}", userH.Delete)
	mux.HandleFunc("GET /api/v1/users/{id}/roles", userH.GetRoles)
	mux.HandleFunc("POST /api/v1/users/{id}/roles", userH.AssignRole)
	mux.HandleFunc("DELETE /api/v1/users/{id}/roles", userH.RevokeRole)

	// --- Audit ---
	mux.HandleFunc("GET /api/v1/audit", auditH.List)

	// --- Federation (stub) ---
	mux.HandleFunc("GET /api/v1/federation/", httphandlers.FederationStub)
	mux.HandleFunc("POST /api/v1/federation/", httphandlers.FederationStub)

	// --- Catch-all: app shell ---
	mux.Handle("GET /{path...}", httphandlers.Shell(cfg))

	// Middleware chain: outermost wraps first
	auditCtx := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(service.ContextWithRequest(r.Context(), r)))
		})
	}

	chain := middleware.RequestLogger(log)(
		middleware.Locale(
			auditCtx(
				oidc.Handler(
					middleware.CedarGate(
						middleware.ContentNegotiation(mux),
					),
				),
			),
		),
	)
	return chain, nil
}
