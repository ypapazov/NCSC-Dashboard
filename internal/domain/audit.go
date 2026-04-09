package domain

import (
	"time"

	"github.com/google/uuid"
)

type AuditEntry struct {
	ID           uuid.UUID      `json:"id"`
	Timestamp    time.Time      `json:"timestamp"`
	ActorID      uuid.UUID      `json:"actor_id"`
	ActorType    string         `json:"actor_type"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resource_type"`
	ResourceID   *uuid.UUID     `json:"resource_id,omitempty"`
	ScopeType    string         `json:"scope_type,omitempty"`
	ScopeID      *uuid.UUID     `json:"scope_id,omitempty"`
	Detail       map[string]any `json:"detail"`
	Severity     AuditSeverity  `json:"severity"`
	IPAddress    string         `json:"ip_address,omitempty"`
	UserAgent    string         `json:"user_agent,omitempty"`
}

type AuditFilter struct {
	ActorID      *uuid.UUID
	ResourceType string
	ResourceID   *uuid.UUID
	ScopeType    string
	ScopeID      *uuid.UUID
	Severity     *AuditSeverity
	DateFrom     *time.Time
	DateTo       *time.Time
	Pagination
}
