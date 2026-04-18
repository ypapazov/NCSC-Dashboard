CREATE SCHEMA IF NOT EXISTS fresnel_iam;

SET search_path TO fresnel_iam, fresnel, public;

CREATE TABLE cedar_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scope_type TEXT NOT NULL,
    scope_id UUID,
    policy_template TEXT NOT NULL,
    cedar_text TEXT NOT NULL,
    created_by UUID REFERENCES fresnel.users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE role_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES fresnel.users(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    scope_type TEXT NOT NULL,
    scope_id UUID NOT NULL,
    assigned_by UUID REFERENCES fresnel.users(id) ON DELETE SET NULL,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(user_id, role, scope_type, scope_id)
);

CREATE TABLE root_designations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES fresnel.users(id) ON DELETE CASCADE,
    scope_type TEXT NOT NULL,
    scope_id UUID,
    designated_by UUID REFERENCES fresnel.users(id) ON DELETE SET NULL,
    designated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(scope_type, scope_id)
);
