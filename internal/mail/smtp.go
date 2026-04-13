package mail

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"strings"
)

// SMTPMailer sends email via a standard SMTP relay.
// Used on-prem (vSphere) or with any SMTP-compatible service.
type SMTPMailer struct {
	host     string
	port     int
	from     string
	username string
	password string
	logger   *slog.Logger
}

func NewSMTPMailer(host string, port int, from, username, password string, log *slog.Logger) *SMTPMailer {
	return &SMTPMailer{
		host:     host,
		port:     port,
		from:     from,
		username: username,
		password: password,
		logger:   log,
	}
}

func (m *SMTPMailer) Send(_ context.Context, to, subject, body string) error {
	addr := net.JoinHostPort(m.host, fmt.Sprintf("%d", m.port))
	msg := buildMessage(m.from, to, subject, body)

	var auth smtp.Auth
	if m.username != "" {
		auth = smtp.PlainAuth("", m.username, m.password, m.host)
	}

	if err := smtp.SendMail(addr, auth, m.from, []string{to}, []byte(msg)); err != nil {
		m.logger.Error("email send failed", "to", to, "subject", subject, "err", err)
		return fmt.Errorf("send email: %w", err)
	}
	m.logger.Info("email sent", "to", to, "subject", subject)
	return nil
}

func buildMessage(from, to, subject, body string) string {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	return b.String()
}
