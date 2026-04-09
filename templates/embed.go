package htmltpl

import "embed"

// FS holds HTML layouts, pages, and partials (embedded from this directory).
//
//go:embed layouts/*.html dashboard/*.html events/*.html status_reports/*.html campaigns/*.html admin/*.html audit/*.html partials/*.html errors/*.html
var FS embed.FS
