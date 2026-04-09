package templates

import (
	"html/template"
	"strings"

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
		"lower": strings.ToLower,
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
	}
}
