-- Fresnel application schema
CREATE SCHEMA IF NOT EXISTS fresnel;

SET search_path TO fresnel, public;

CREATE TABLE sectors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_sector_id UUID REFERENCES sectors(id),
    name TEXT NOT NULL,
    ancestry_path TEXT NOT NULL DEFAULT '/',
    depth INTEGER NOT NULL DEFAULT 1,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (depth <= 5)
);

CREATE INDEX idx_sectors_parent ON sectors(parent_sector_id);
CREATE INDEX idx_sectors_ancestry ON sectors(ancestry_path text_pattern_ops);

CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sector_id UUID NOT NULL REFERENCES sectors(id),
    name TEXT NOT NULL,
    timezone TEXT NOT NULL DEFAULT 'UTC',
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    keycloak_sub TEXT UNIQUE NOT NULL,
    display_name TEXT NOT NULL,
    email TEXT NOT NULL,
    primary_org_id UUID NOT NULL REFERENCES organizations(id),
    timezone TEXT NOT NULL DEFAULT 'UTC',
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE org_sector_memberships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    sector_id UUID NOT NULL REFERENCES sectors(id),
    root_user_id UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(organization_id, sector_id)
);

CREATE TABLE user_org_memberships (
    user_id UUID NOT NULL REFERENCES users(id),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    assigned_by UUID NOT NULL REFERENCES users(id),
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, organization_id)
);

CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_instance TEXT NOT NULL DEFAULT 'local',
    sector_context UUID NOT NULL REFERENCES sectors(id),
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    event_type TEXT NOT NULL,
    submitter_id UUID NOT NULL REFERENCES users(id),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    tlp TEXT NOT NULL,
    impact TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'OPEN',
    intel_source TEXT NOT NULL DEFAULT 'Manual',
    target TEXT NOT NULL DEFAULT '',
    original_event_date TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE event_revisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id),
    revision_number INTEGER NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    event_type TEXT NOT NULL,
    tlp TEXT NOT NULL,
    impact TEXT NOT NULL,
    status TEXT NOT NULL,
    changed_by UUID NOT NULL REFERENCES users(id),
    changed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(event_id, revision_number)
);

CREATE TABLE event_updates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id),
    author_id UUID NOT NULL REFERENCES users(id),
    body TEXT NOT NULL,
    tlp TEXT NOT NULL,
    impact_change TEXT,
    status_change TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE status_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_instance TEXT NOT NULL DEFAULT 'local',
    sector_context UUID NOT NULL REFERENCES sectors(id),
    scope_type TEXT NOT NULL,
    scope_ref UUID NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    period_covered_start TIMESTAMPTZ NOT NULL,
    period_covered_end TIMESTAMPTZ NOT NULL,
    as_of TIMESTAMPTZ NOT NULL,
    published_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    assessed_status TEXT NOT NULL,
    impact TEXT NOT NULL,
    tlp TEXT NOT NULL,
    author_id UUID NOT NULL REFERENCES users(id),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE status_report_revisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status_report_id UUID NOT NULL REFERENCES status_reports(id),
    revision_number INTEGER NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    assessed_status TEXT NOT NULL,
    impact TEXT NOT NULL,
    tlp TEXT NOT NULL,
    changed_by UUID NOT NULL REFERENCES users(id),
    changed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(status_report_id, revision_number)
);

CREATE TABLE status_report_events (
    status_report_id UUID NOT NULL REFERENCES status_reports(id),
    event_id UUID NOT NULL REFERENCES events(id),
    PRIMARY KEY (status_report_id, event_id)
);

CREATE INDEX idx_status_report_events_event ON status_report_events(event_id);

CREATE TABLE campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    tlp TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE',
    created_by UUID NOT NULL REFERENCES users(id),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE campaign_events (
    campaign_id UUID NOT NULL REFERENCES campaigns(id),
    event_id UUID NOT NULL REFERENCES events(id),
    linked_by UUID NOT NULL REFERENCES users(id),
    linked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (campaign_id, event_id)
);

CREATE TABLE correlations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_a_id UUID NOT NULL REFERENCES events(id),
    event_b_id UUID NOT NULL REFERENCES events(id),
    label TEXT NOT NULL,
    correlation_type TEXT NOT NULL DEFAULT 'MANUAL',
    created_by_user UUID REFERENCES users(id),
    created_by_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (event_a_id < event_b_id)
);

CREATE TABLE event_relationships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_event_id UUID NOT NULL REFERENCES events(id),
    target_event_id UUID NOT NULL REFERENCES events(id),
    label TEXT NOT NULL,
    created_by_user UUID REFERENCES users(id),
    created_by_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id),
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    storage_path TEXT NOT NULL,
    scan_status TEXT NOT NULL DEFAULT 'pending',
    uploaded_by UUID NOT NULL REFERENCES users(id),
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE tlp_red_recipients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_type TEXT NOT NULL,
    resource_id UUID NOT NULL,
    recipient_user_id UUID NOT NULL REFERENCES users(id),
    granted_by UUID NOT NULL REFERENCES users(id),
    granted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(resource_type, resource_id, recipient_user_id)
);

CREATE INDEX idx_tlp_red_resource ON tlp_red_recipients(resource_type, resource_id);

CREATE TABLE platform_config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_by UUID REFERENCES users(id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE nudge_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id),
    recipient_id UUID NOT NULL REFERENCES users(id),
    nudge_type TEXT NOT NULL,
    escalation_level INTEGER,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_nudge_event_date ON nudge_log(event_id, sent_at DESC);

CREATE TABLE escalation_state (
    event_id UUID PRIMARY KEY REFERENCES events(id),
    current_level INTEGER NOT NULL DEFAULT 0,
    escalated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_response_at TIMESTAMPTZ
);

CREATE TABLE status_formulas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_type TEXT NOT NULL,
    node_id UUID,
    starlark_source TEXT NOT NULL,
    set_by UUID NOT NULL REFERENCES users(id),
    set_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(node_type, node_id)
);
