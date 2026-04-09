package handlers

import (
	"net/http"
)

// FederationStub returns 501 Not Implemented for all federation endpoints.
// Federation is planned for a future milestone.
func FederationStub(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusNotImplemented, map[string]string{
		"error":   "not_implemented",
		"message": "federation endpoints are not yet available",
	})
}
