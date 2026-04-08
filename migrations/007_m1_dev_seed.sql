-- Optional dev hierarchy + user for M1 login testing.
-- Create a matching Keycloak user with the same email; OIDC will link keycloak_sub on first login.

SET search_path TO fresnel, public;

INSERT INTO fresnel.sectors (id, parent_sector_id, name, ancestry_path, depth)
VALUES (
    'a0000000-0000-4000-8000-000000000001'::uuid,
    NULL,
    'Development',
    '/dev/',
    1
) ON CONFLICT (id) DO NOTHING;

INSERT INTO fresnel.organizations (id, sector_id, name)
VALUES (
    'a0000000-0000-4000-8000-000000000002'::uuid,
    'a0000000-0000-4000-8000-000000000001'::uuid,
    'Dev Organization'
) ON CONFLICT (id) DO NOTHING;

INSERT INTO fresnel.users (id, keycloak_sub, display_name, email, primary_org_id)
VALUES (
    'a0000000-0000-4000-8000-000000000003'::uuid,
    'bootstrap-dev-placeholder',
    'Dev User',
    'dev@fresnel.local',
    'a0000000-0000-4000-8000-000000000002'::uuid
) ON CONFLICT (id) DO NOTHING;

INSERT INTO fresnel.user_org_memberships (user_id, organization_id, assigned_by)
VALUES (
    'a0000000-0000-4000-8000-000000000003'::uuid,
    'a0000000-0000-4000-8000-000000000002'::uuid,
    'a0000000-0000-4000-8000-000000000003'::uuid
) ON CONFLICT (user_id, organization_id) DO NOTHING;
