-- Platform administrator: user row, org membership, PLATFORM_ROOT role, and root designation.
-- References CERT.bg (from 006a_seed_sectors.sql) as the bootstrap org.
-- This file must NOT be a _dev_ seed — it runs in production.
SET search_path TO fresnel, public;

INSERT INTO fresnel.users (id, keycloak_sub, display_name, email, primary_org_id)
VALUES (
    'a0000000-0000-4000-8000-000000000004'::uuid,
    'bootstrap-platform-admin-placeholder',
    'Platform Administrator',
    'admin@cyberbg.local',
    'b0000000-0000-4000-8000-000000000020'::uuid
) ON CONFLICT (id) DO NOTHING;

INSERT INTO fresnel.user_org_memberships (user_id, organization_id, assigned_by)
VALUES (
    'a0000000-0000-4000-8000-000000000004'::uuid,
    'b0000000-0000-4000-8000-000000000020'::uuid,
    'a0000000-0000-4000-8000-000000000004'::uuid
) ON CONFLICT (user_id, organization_id) DO NOTHING;

-- IAM: grant PLATFORM_ROOT role
SET search_path TO fresnel_iam, fresnel, public;

INSERT INTO fresnel_iam.role_assignments (id, user_id, role, scope_type, scope_id, assigned_by)
VALUES (
    'a0000000-0000-4000-8000-000000000010'::uuid,
    'a0000000-0000-4000-8000-000000000004'::uuid,
    'PLATFORM_ROOT',
    'PLATFORM',
    'b0000000-0000-4000-8000-000000000020'::uuid,
    'a0000000-0000-4000-8000-000000000004'::uuid
) ON CONFLICT (user_id, role, scope_type, scope_id) DO NOTHING;

INSERT INTO fresnel_iam.root_designations (id, user_id, scope_type, scope_id, designated_by)
VALUES (
    'a0000000-0000-4000-8000-000000000011'::uuid,
    'a0000000-0000-4000-8000-000000000004'::uuid,
    'PLATFORM',
    NULL,
    'a0000000-0000-4000-8000-000000000004'::uuid
) ON CONFLICT (id) DO NOTHING;
