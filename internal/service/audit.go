package service

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"fresnel/internal/domain"
	"fresnel/internal/storage"

	"github.com/google/uuid"
)

type AuditService struct {
	store  storage.AuditStore
	logger *slog.Logger
}

func NewAuditService(store storage.AuditStore, logger *slog.Logger) *AuditService {
	return &AuditService{store: store, logger: logger}
}

func (s *AuditService) Log(ctx context.Context, auth *domain.AuthContext, action, resourceType string, resourceID *uuid.UUID, severity domain.AuditSeverity, detail map[string]any) {
	entry := &domain.AuditEntry{
		ID:           uuid.New(),
		Timestamp:    time.Now().UTC(),
		ActorID:      auth.UserID,
		ActorType:    "user",
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Detail:       detail,
		Severity:     severity,
	}
	if auth.RootScope != nil {
		entry.ScopeType = auth.RootScope.Type
		entry.ScopeID = &auth.RootScope.ID
	}

	if r, ok := ctx.Value(httpRequestKey).(*http.Request); ok {
		entry.IPAddress = r.RemoteAddr
		entry.UserAgent = r.Header.Get("User-Agent")
	}

	if err := s.store.Insert(ctx, entry); err != nil {
		s.logger.Error("audit log failed", "action", action, "resource_type", resourceType, "err", err)
	}
}

func (s *AuditService) List(ctx context.Context, filter domain.AuditFilter) (*domain.ListResult[*domain.AuditEntry], error) {
	filter.Normalize()
	return s.store.List(ctx, filter)
}

type ctxKey string

const httpRequestKey ctxKey = "http_request"

func ContextWithRequest(ctx context.Context, r *http.Request) context.Context {
	return context.WithValue(ctx, httpRequestKey, r)
}
