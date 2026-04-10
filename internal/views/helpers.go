package views

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"fresnel/internal/domain"

	"github.com/google/uuid"
)

// Suppress unused import warnings for packages used only in .templ files.
var _ = strings.ToLower

func Lower(v any) string {
	return strings.ToLower(fmt.Sprintf("%v", v))
}

func FmtTime(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Format("2 Jan 2006 15:04 UTC")
}

func FmtTimestamp(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Format("2006-01-02T15:04:05Z")
}

func FmtDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

func FmtUser(id uuid.UUID) string {
	s := id.String()
	if len(s) > 8 {
		return s[:8] + "…"
	}
	return s
}

func FmtBytes(b int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case b >= gb:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func FmtJSON(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

func attachmentSeverityClass(status domain.ScanStatus) string {
	switch status {
	case domain.ScanClean:
		return "info"
	case domain.ScanQuarantined:
		return "high"
	default:
		return "medium"
	}
}

func safeEventField(e *domain.Event, fn func(*domain.Event) string) string {
	if e == nil {
		return ""
	}
	return fn(e)
}

func safeReportField(r *domain.StatusReport, fn func(*domain.StatusReport) string) string {
	if r == nil {
		return ""
	}
	return fn(r)
}

func safeCampaignField(c *domain.Campaign, fn func(*domain.Campaign) string) string {
	if c == nil {
		return ""
	}
	return fn(c)
}

func RelativeTime(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", h)
	case d < 30*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	default:
		return FmtDate(t)
	}
}
