package mail

import (
	"context"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"
)

type Mailer struct {
	Host    string
	Port    int
	From    string
	Logger  *slog.Logger
	enabled bool
}

func NewMailer(host string, port int, from string, log *slog.Logger) *Mailer {
	return &Mailer{
		Host:    host,
		Port:    port,
		From:    from,
		Logger:  log,
		enabled: host != "",
	}
}

func (m *Mailer) Send(_ context.Context, to, subject, body string) error {
	if !m.enabled {
		m.Logger.Warn("email not sent (SMTP not configured)", "to", to, "subject", subject)
		return nil
	}

	addr := fmt.Sprintf("%s:%d", m.Host, m.Port)
	msg := buildMessage(m.From, to, subject, body)

	if err := smtp.SendMail(addr, nil, m.From, []string{to}, []byte(msg)); err != nil {
		m.Logger.Error("email send failed", "to", to, "subject", subject, "err", err)
		return fmt.Errorf("send email: %w", err)
	}
	m.Logger.Info("email sent", "to", to, "subject", subject)
	return nil
}

func (m *Mailer) IsEnabled() bool { return m.enabled }

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
