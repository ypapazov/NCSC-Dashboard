package static

import "embed"

// Files contains vendored frontend assets (HTMX, CSS) and generated help pages.
//
//go:embed htmx.min.js keycloak.min.js app.js cytoscape.min.js graph.js campaign-selection.js vis-timeline.min.js timeline.js css/*.css flags/*.svg all:help
var Files embed.FS
