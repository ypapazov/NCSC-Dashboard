CREATE SCHEMA IF NOT EXISTS fresnel_audit;

SET search_path TO fresnel_audit, public;

CREATE TABLE audit_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT now(),
    actor_id UUID NOT NULL,
    actor_type TEXT NOT NULL,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id UUID,
    scope_type TEXT,
    scope_id UUID,
    detail JSONB NOT NULL DEFAULT '{}',
    severity TEXT NOT NULL DEFAULT 'INFO',
    ip_address INET,
    user_agent TEXT
);

CREATE INDEX idx_audit_timestamp ON fresnel_audit.audit_entries(timestamp DESC);
CREATE INDEX idx_audit_actor ON fresnel_audit.audit_entries(actor_id);
CREATE INDEX idx_audit_resource ON fresnel_audit.audit_entries(resource_type, resource_id);
CREATE INDEX idx_audit_scope ON fresnel_audit.audit_entries(scope_type, scope_id);
