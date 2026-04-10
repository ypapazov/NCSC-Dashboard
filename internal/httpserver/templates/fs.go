package templates

import (
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/google/uuid"

	htmltpl "fresnel/templates"
)

// Parse returns the root template bundle for HTML pages.
func Parse() (*template.Template, error) {
	return template.New("root").Funcs(funcMap()).ParseFS(htmltpl.FS,
		"layouts/base.html",
		"dashboard/*.html",
		"events/*.html",
		"status_reports/*.html",
		"campaigns/*.html",
		"admin/*.html",
		"audit/*.html",
		"partials/*.html",
		"errors/*.html",
	)
}

func funcMap() template.FuncMap {
	return template.FuncMap{
		"lower": func(v any) string {
			return strings.ToLower(fmt.Sprintf("%v", v))
		},
		"deref": func(v interface{}) string {
			if v == nil {
				return ""
			}
			switch s := v.(type) {
			case *string:
				if s == nil {
					return ""
				}
				return *s
			case string:
				return s
			default:
				return ""
			}
		},
		"fmtTime": func(t time.Time) string {
			if t.IsZero() {
				return "—"
			}
			return t.Format("2 Jan 2006 15:04 UTC")
		},
		"fmtTimestamp": func(t time.Time) string {
			if t.IsZero() {
				return "—"
			}
			return t.Format("2006-01-02T15:04:05Z")
		},
		"fmtDate": func(t time.Time) string {
			if t.IsZero() {
				return ""
			}
			return t.Format("2006-01-02")
		},
		"fmtJSON": func(v interface{}) string {
			b, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				return fmt.Sprintf("%v", v)
			}
			return string(b)
		},
		"fmtUser": func(v interface{}) string {
			switch u := v.(type) {
			case uuid.UUID:
				return u.String()[:8] + "…"
			case string:
				if len(u) > 8 {
					return u[:8] + "…"
				}
				return u
			default:
				return fmt.Sprintf("%v", v)
			}
		},
		"fmtBytes": func(b int64) string {
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
		},
	}
}
