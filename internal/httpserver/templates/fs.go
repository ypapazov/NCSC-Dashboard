package templates

import (
	"html/template"

	htmltpl "fresnel/templates"
)

// Parse returns the root template bundle for HTML pages.
func Parse() (*template.Template, error) {
	return template.New("root").ParseFS(htmltpl.FS,
		"layouts/base.html",
		"dashboard/index.html",
	)
}
