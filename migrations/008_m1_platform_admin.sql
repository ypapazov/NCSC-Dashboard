-- Platform administrator row for OIDC email linking (matches Keycloak user in fresnel-realm.json).
-- References CERT.bg (from 006a_seed_sectors.sql) as the bootstrap org.
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
