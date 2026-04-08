package htmltpl

import "embed"

// FS holds HTML layouts and pages (embedded from this directory).
//
//go:embed layouts/*.html dashboard/*.html
var FS embed.FS
