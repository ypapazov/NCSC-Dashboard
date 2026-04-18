package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fresnel/internal/authz"
	"fresnel/internal/clamav"
	"fresnel/internal/config"
	apphttp "fresnel/internal/httpserver"
	"fresnel/internal/mail"
	"fresnel/internal/service"
	"fresnel/internal/storage/postgres"

	"github.com/google/uuid"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		os.Exit(runMigrate(log))
	}

	cfg, err := config.Load()
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(1)
	}
	if err := cfg.Validate(); err != nil {
		log.Error("config invalid", "err", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := postgres.Migrate(ctx, pool); err != nil {
		log.Error("migrate", "err", err)
		os.Exit(1)
	}

	// --- Stores ---
	sectorStore := postgres.NewSectorStore(pool)
	orgStore := postgres.NewOrganizationStore(pool)
	userStore := postgres.NewUserStore(pool)
	roleStore := postgres.NewRoleStore(pool)
	eventStore := postgres.NewEventStore(pool)
	eventUpdateStore := postgres.NewEventUpdateStore(pool)
	statusReportStore := postgres.NewStatusReportStore(pool)
	campaignStore := postgres.NewCampaignStore(pool)
	correlationStore := postgres.NewCorrelationStore(pool)
	relationshipStore := postgres.NewEventRelationshipStore(pool)
	attachmentStore := postgres.NewAttachmentStore(pool)
	tlpRedStore := postgres.NewTLPRedStore(pool)
	auditStore := postgres.NewAuditStore(pool)

	// --- Authorizer ---
	az := authz.NewCedarAuthorizer(
		func(sectorID uuid.UUID) string {
			sec, err := sectorStore.GetByID(context.Background(), sectorID)
			if err != nil || sec == nil {
				return ""
			}
			return sec.AncestryPath
		},
		func(orgID uuid.UUID) uuid.UUID {
			org, err := orgStore.GetByID(context.Background(), orgID)
			if err != nil || org == nil {
				return uuid.Nil
			}
			return org.SectorID
		},
	)

	// --- Services ---
	auditSvc := service.NewAuditService(auditStore, log)
	mailFrom := os.Getenv("SMTP_FROM")
	if mailFrom == "" {
		mailFrom = "noreply@fresnel.local"
	}
	mailer, err := mail.New(ctx, mail.Config{
		SESRegion:    cfg.SESRegion,
		SMTPHost:     cfg.SMTPHost,
		SMTPPort:     cfg.SMTPPort,
		SMTPUsername: cfg.SMTPUsername,
		SMTPPassword: cfg.SMTPPassword,
		From:         mailFrom,
	}, log)
	if err != nil {
		log.Error("mail setup", "err", err)
		os.Exit(1)
	}
	nudgeStore := postgres.NewNudgeStore(pool)
	nudgeSvc := service.NewNudgeService(
		nudgeStore, eventStore, eventUpdateStore, userStore, roleStore, orgStore, sectorStore,
		mailer, auditSvc, log, cfg.AppPublicURL,
	)
	nudgeSvc.Start(context.Background())
	defer nudgeSvc.Stop()

	sectorSvc := service.NewSectorService(sectorStore, az, auditSvc)
	eventSvc := service.NewEventService(eventStore, eventUpdateStore, sectorStore, tlpRedStore, az, auditSvc, nudgeSvc)
	statusReportSvc := service.NewStatusReportService(statusReportStore, sectorStore, tlpRedStore, az, auditSvc)
	campaignSvc := service.NewCampaignService(campaignStore, eventStore, sectorStore, tlpRedStore, az, auditSvc)
	orgSvc := service.NewOrganizationService(orgStore, sectorStore, az, auditSvc)
	userSvc := service.NewUserService(userStore, roleStore, az, auditSvc)
	corrSvc := service.NewCorrelationService(correlationStore, relationshipStore, eventStore, sectorStore, tlpRedStore, az, auditSvc)
	var scanner *clamav.Client
	if cfg.ClamAVAddress != "" {
		scanner = clamav.NewTCPClient(cfg.ClamAVAddress)
		log.Info("ClamAV enabled", "address", cfg.ClamAVAddress)
	} else {
		log.Warn("ClamAV disabled (CLAMAV_ADDRESS not set)")
	}
	attachSvc := service.NewAttachmentService(attachmentStore, eventStore, sectorStore, tlpRedStore, scanner, az, auditSvc, cfg.AttachmentDir)
	dashboardSvc := service.NewDashboardService(sectorStore, orgStore, statusReportStore, az, cfg.DashboardCacheTTL)

	svc := apphttp.Services{
		Events:        eventSvc,
		StatusReports: statusReportSvc,
		Campaigns:     campaignSvc,
		Sectors:       sectorSvc,
		Orgs:          orgSvc,
		Users:         userSvc,
		Correlations:  corrSvc,
		Attachments:   attachSvc,
		Audit:         auditSvc,
		Dashboard:     dashboardSvc,
	}

	lk := apphttp.Lookups{
		Orgs:    orgStore,
		Sectors: sectorStore,
		Users:   userStore,
		TLPRed:  tlpRedStore,
		Authz:   az,
	}

	handler, err := apphttp.NewRouter(log, cfg, pool, svc, lk)
	if err != nil {
		log.Error("router", "err", err)
		os.Exit(1)
	}
	srv := apphttp.NewServer(cfg.ListenAddr, log, handler)

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Error("server", "err", err)
			os.Exit(1)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown", "err", err)
	}
}

func runMigrate(log *slog.Logger) int {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Error("DATABASE_URL required")
		return 1
	}
	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, dsn)
	if err != nil {
		log.Error("database", "err", err)
		return 1
	}
	defer pool.Close()
	if err := postgres.Migrate(ctx, pool); err != nil {
		log.Error("migrate", "err", err)
		return 1
	}
	log.Info("migrations applied")
	return 0
}
