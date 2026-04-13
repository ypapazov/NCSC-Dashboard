package mail

import (
	"context"
	"log/slog"
)

// Sender is the interface consumed by the rest of the application.
type Sender interface {
	Send(ctx context.Context, to, subject, body string) error
}

// disabledMailer is returned when no email backend is configured.
type disabledMailer struct {
	logger *slog.Logger
}

func (m *disabledMailer) Send(_ context.Context, to, subject, _ string) error {
	m.logger.Warn("email not sent (no backend configured)", "to", to, "subject", subject)
	return nil
}

// Config holds the settings needed to construct a Sender.
type Config struct {
	// SES mode: set SESRegion to a non-empty AWS region (e.g. "eu-west-2").
	// Uses EC2 instance-role credentials — no static keys needed.
	SESRegion string

	// SMTP mode: set SMTPHost to a non-empty hostname.
	// Falls back to this when SESRegion is empty (on-prem / vSphere).
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string

	From string
}

// New creates the appropriate Sender based on configuration.
// Priority: SES API (if SESRegion set) > SMTP (if SMTPHost set) > disabled.
func New(ctx context.Context, cfg Config, log *slog.Logger) (Sender, error) {
	if cfg.SESRegion != "" {
		log.Info("email backend: SES API", "region", cfg.SESRegion, "from", cfg.From)
		return NewSESMailer(ctx, cfg.SESRegion, cfg.From, log)
	}
	if cfg.SMTPHost != "" {
		log.Info("email backend: SMTP", "host", cfg.SMTPHost, "port", cfg.SMTPPort, "from", cfg.From)
		return NewSMTPMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.From, cfg.SMTPUsername, cfg.SMTPPassword, log), nil
	}
	log.Warn("email disabled (neither SES_REGION nor SMTP_HOST configured)")
	return &disabledMailer{logger: log}, nil
}
