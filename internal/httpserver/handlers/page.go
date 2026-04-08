package handlers

import "fresnel/internal/domain"

// PageData is passed to HTML templates.
type PageData struct {
	User *domain.AuthContext
}
