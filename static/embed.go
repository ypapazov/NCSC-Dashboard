package static

import "embed"

// Files contains vendored frontend assets (HTMX, CSS).
//
//go:embed htmx.min.js keycloak.min.js app.js cytoscape.min.js graph.js css/*.css
var Files embed.FS
