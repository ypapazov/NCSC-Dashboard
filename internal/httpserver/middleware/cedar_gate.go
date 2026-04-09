package middleware

import "net/http"

// CedarGate is a coarse-grained pre-filter middleware placeholder.
//
// The real per-resource Cedar authorization happens in the service layer.
// This middleware is a passthrough that can be enhanced later with
// route-level checks (e.g. "does this user have ANY role that could
// possibly do this action on this resource TYPE?").
func CedarGate(next http.Handler) http.Handler {
	return next
}
