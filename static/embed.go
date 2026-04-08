package static

import "embed"

// Files contains vendored frontend assets (HTMX, CSS).
//
//go:embed htmx.min.js css/*.css
var Files embed.FS
