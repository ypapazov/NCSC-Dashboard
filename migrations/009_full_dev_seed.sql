-- Full development hierarchy: sectors, organizations, test users, IAM, sample events/reports.
-- UUIDs use b0* (structure), b1* (users), b21*/b22* (IAM), b30* (demo content) to avoid clashing with 007/008 (a0*).
-- Idempotent: ON CONFLICT DO NOTHING / email-based user dedupe vs prior seeds.

SET search_path TO fresnel, public;

-- Sectors (parents before children)
INSERT INTO fresnel.sectors (id, parent_sector_id, name, ancestry_path, depth)
VALUES
    ('b0000000-0000-4000-8000-000000000001'::uuid, NULL, 'Government', '/gov/', 1),
    ('b0000000-0000-4000-8000-000000000004'::uuid, NULL, 'Finance', '/finance/', 1),
    ('b0000000-0000-4000-8000-000000000005'::uuid, NULL, 'Critical Infrastructure', '/critinfra/', 1)
ON CONFLICT (id) DO NOTHING;

INSERT INTO fresnel.sectors (id, parent_sector_id, name, ancestry_path, depth)
VALUES
    ('b0000000-0000-4000-8000-000000000002'::uuid, 'b0000000-0000-4000-8000-000000000001'::uuid, 'Federal', '/gov/federal/', 2),
    ('b0000000-0000-4000-8000-000000000003'::uuid, 'b0000000-0000-4000-8000-000000000001'::uuid, 'State', '/gov/state/', 2),
    ('b0000000-0000-4000-8000-000000000006'::uuid, 'b0000000-0000-4000-8000-000000000005'::uuid, 'Energy', '/critinfra/energy/', 2),
    ('b0000000-0000-4000-8000-000000000007'::uuid, 'b0000000-0000-4000-8000-000000000005'::uuid, 'Telecommunications', '/critinfra/telecom/', 2)
ON CONFLICT (id) DO NOTHING;

-- Organizations
INSERT INTO fresnel.organizations (id, sector_id, name)
VALUES
    ('b0000000-0000-4000-8000-000000000010'::uuid, 'b0000000-0000-4000-8000-000000000002'::uuid, 'Department of Technology'),
    ('b0000000-0000-4000-8000-000000000011'::uuid, 'b0000000-0000-4000-8000-000000000002'::uuid, 'National Security Agency'),
    ('b0000000-0000-4000-8000-000000000012'::uuid, 'b0000000-0000-4000-8000-000000000003'::uuid, 'State IT Authority'),
    ('b0000000-0000-4000-8000-000000000013'::uuid, 'b0000000-0000-4000-8000-000000000004'::uuid, 'Central Bank'),
    ('b0000000-0000-4000-8000-000000000014'::uuid, 'b0000000-0000-4000-8000-000000000004'::uuid, 'Financial Regulatory Authority'),
    ('b0000000-0000-4000-8000-000000000015'::uuid, 'b0000000-0000-4000-8000-000000000006'::uuid, 'National Grid Operator'),
    ('b0000000-0000-4000-8000-000000000016'::uuid, 'b0000000-0000-4000-8000-000000000007'::uuid, 'Telecom Authority')
ON CONFLICT (id) DO NOTHING;

-- Platform root user already exists from 008 as a0..0004 (admin@fresnel.local).
-- We skip re-inserting it and reference a0..0004 directly for IAM below.
INSERT INTO fresnel.users (id, keycloak_sub, display_name, email, primary_org_id)
VALUES
    ('b1000000-0000-4000-8000-000000000002'::uuid, 'placeholder-gov-root', 'Government Sector Root', 'gov-root@fresnel.local', 'b0000000-0000-4000-8000-000000000010'::uuid),
    ('b1000000-0000-4000-8000-000000000003'::uuid, 'placeholder-fed-root', 'Federal Sector Root', 'fed-root@fresnel.local', 'b0000000-0000-4000-8000-000000000010'::uuid),
    ('b1000000-0000-4000-8000-000000000004'::uuid, 'placeholder-orga-root', 'Org A Root', 'orga-root@fresnel.local', 'b0000000-0000-4000-8000-000000000010'::uuid),
    ('b1000000-0000-4000-8000-000000000005'::uuid, 'placeholder-orga-admin', 'Org A Admin', 'orga-admin@fresnel.local', 'b0000000-0000-4000-8000-000000000010'::uuid),
    ('b1000000-0000-4000-8000-000000000006'::uuid, 'placeholder-orga-contrib', 'Org A Contributor', 'orga-contrib@fresnel.local', 'b0000000-0000-4000-8000-000000000010'::uuid),
    ('b1000000-0000-4000-8000-000000000007'::uuid, 'placeholder-orga-viewer', 'Org A Viewer', 'orga-viewer@fresnel.local', 'b0000000-0000-4000-8000-000000000010'::uuid),
    ('b1000000-0000-4000-8000-000000000008'::uuid, 'placeholder-orgb-root', 'Org B Root', 'orgb-root@fresnel.local', 'b0000000-0000-4000-8000-000000000011'::uuid)
ON CONFLICT ((lower(email))) DO NOTHING;

INSERT INTO fresnel.user_org_memberships (user_id, organization_id, assigned_by)
SELECT m.user_id, m.organization_id, m.user_id
FROM (VALUES
    ('a0000000-0000-4000-8000-000000000004'::uuid, 'b0000000-0000-4000-8000-000000000010'::uuid),
    ('b1000000-0000-4000-8000-000000000002'::uuid, 'b0000000-0000-4000-8000-000000000010'::uuid),
    ('b1000000-0000-4000-8000-000000000003'::uuid, 'b0000000-0000-4000-8000-000000000010'::uuid),
    ('b1000000-0000-4000-8000-000000000004'::uuid, 'b0000000-0000-4000-8000-000000000010'::uuid),
    ('b1000000-0000-4000-8000-000000000005'::uuid, 'b0000000-0000-4000-8000-000000000010'::uuid),
    ('b1000000-0000-4000-8000-000000000006'::uuid, 'b0000000-0000-4000-8000-000000000010'::uuid),
    ('b1000000-0000-4000-8000-000000000007'::uuid, 'b0000000-0000-4000-8000-000000000010'::uuid),
    ('b1000000-0000-4000-8000-000000000008'::uuid, 'b0000000-0000-4000-8000-000000000011'::uuid)
) AS m(user_id, organization_id)
WHERE EXISTS (SELECT 1 FROM fresnel.users u WHERE u.id = m.user_id)
ON CONFLICT (user_id, organization_id) DO NOTHING;

SET search_path TO fresnel_iam, fresnel, public;

INSERT INTO fresnel_iam.role_assignments (id, user_id, role, scope_type, scope_id, assigned_by)
SELECT a.id, a.user_id, a.role, a.scope_type, a.scope_id, a.assigned_by
FROM (VALUES
    ('b2100000-0000-4000-8000-000000000000'::uuid, 'a0000000-0000-4000-8000-000000000004'::uuid, 'PLATFORM_ROOT', 'PLATFORM', 'b0000000-0000-4000-8000-000000000010'::uuid, 'a0000000-0000-4000-8000-000000000004'::uuid),
    ('b2100000-0000-4000-8000-000000000001'::uuid, 'b1000000-0000-4000-8000-000000000002'::uuid, 'SECTOR_ROOT', 'SECTOR', 'b0000000-0000-4000-8000-000000000001'::uuid, 'b1000000-0000-4000-8000-000000000002'::uuid),
    ('b2100000-0000-4000-8000-000000000002'::uuid, 'b1000000-0000-4000-8000-000000000003'::uuid, 'SECTOR_ROOT', 'SECTOR', 'b0000000-0000-4000-8000-000000000002'::uuid, 'b1000000-0000-4000-8000-000000000003'::uuid),
    ('b2100000-0000-4000-8000-000000000003'::uuid, 'b1000000-0000-4000-8000-000000000004'::uuid, 'ORG_ROOT', 'ORG', 'b0000000-0000-4000-8000-000000000010'::uuid, 'b1000000-0000-4000-8000-000000000004'::uuid),
    ('b2100000-0000-4000-8000-000000000004'::uuid, 'b1000000-0000-4000-8000-000000000005'::uuid, 'ORG_ADMIN', 'ORG', 'b0000000-0000-4000-8000-000000000010'::uuid, 'b1000000-0000-4000-8000-000000000005'::uuid),
    ('b2100000-0000-4000-8000-000000000005'::uuid, 'b1000000-0000-4000-8000-000000000006'::uuid, 'CONTRIBUTOR', 'ORG', 'b0000000-0000-4000-8000-000000000010'::uuid, 'b1000000-0000-4000-8000-000000000006'::uuid),
    ('b2100000-0000-4000-8000-000000000006'::uuid, 'b1000000-0000-4000-8000-000000000007'::uuid, 'VIEWER', 'ORG', 'b0000000-0000-4000-8000-000000000010'::uuid, 'b1000000-0000-4000-8000-000000000007'::uuid),
    ('b2100000-0000-4000-8000-000000000007'::uuid, 'b1000000-0000-4000-8000-000000000008'::uuid, 'ORG_ROOT', 'ORG', 'b0000000-0000-4000-8000-000000000011'::uuid, 'b1000000-0000-4000-8000-000000000008'::uuid)
) AS a(id, user_id, role, scope_type, scope_id, assigned_by)
WHERE EXISTS (SELECT 1 FROM fresnel.users u WHERE u.id = a.user_id)
ON CONFLICT (user_id, role, scope_type, scope_id) DO NOTHING;

INSERT INTO fresnel_iam.root_designations (id, user_id, scope_type, scope_id, designated_by)
SELECT d.id, d.user_id, d.scope_type, d.scope_id, d.designated_by
FROM (VALUES
    ('b2200000-0000-4000-8000-000000000001'::uuid, 'a0000000-0000-4000-8000-000000000004'::uuid, 'PLATFORM', NULL::uuid, 'a0000000-0000-4000-8000-000000000004'::uuid),
    ('b2200000-0000-4000-8000-000000000002'::uuid, 'b1000000-0000-4000-8000-000000000002'::uuid, 'SECTOR', 'b0000000-0000-4000-8000-000000000001'::uuid, 'b1000000-0000-4000-8000-000000000002'::uuid),
    ('b2200000-0000-4000-8000-000000000003'::uuid, 'b1000000-0000-4000-8000-000000000003'::uuid, 'SECTOR', 'b0000000-0000-4000-8000-000000000002'::uuid, 'b1000000-0000-4000-8000-000000000003'::uuid),
    ('b2200000-0000-4000-8000-000000000004'::uuid, 'b1000000-0000-4000-8000-000000000004'::uuid, 'ORG', 'b0000000-0000-4000-8000-000000000010'::uuid, 'b1000000-0000-4000-8000-000000000004'::uuid),
    ('b2200000-0000-4000-8000-000000000005'::uuid, 'b1000000-0000-4000-8000-000000000008'::uuid, 'ORG', 'b0000000-0000-4000-8000-000000000011'::uuid, 'b1000000-0000-4000-8000-000000000008'::uuid)
) AS d(id, user_id, scope_type, scope_id, designated_by)
WHERE EXISTS (SELECT 1 FROM fresnel.users u WHERE u.id = d.user_id)
ON CONFLICT (id) DO NOTHING;

SET search_path TO fresnel, public;

-- Demo events (submitter: Org A admin when present)
INSERT INTO fresnel.events (
    id, source_instance, sector_context, title, description, event_type,
    submitter_id, organization_id, tlp, impact, status
)
SELECT e.id, e.source_instance, e.sector_context, e.title, e.description, e.event_type,
       e.submitter_id, e.organization_id, e.tlp, e.impact, e.status
FROM (VALUES
    (
        'b3000000-0000-4000-8000-000000000001'::uuid,
        'local',
        'b0000000-0000-4000-8000-000000000002'::uuid,
        'Credential phishing against executive mailboxes',
        'Multiple staff reported suspicious OAuth consent prompts mimicking the internal SSO portal. IOCs shared with sector partners.',
        'PHISHING',
        'b1000000-0000-4000-8000-000000000005'::uuid,
        'b0000000-0000-4000-8000-000000000010'::uuid,
        'AMBER',
        'MODERATE',
        'INVESTIGATING'
    ),
    (
        'b3000000-0000-4000-8000-000000000002'::uuid,
        'local',
        'b0000000-0000-4000-8000-000000000002'::uuid,
        'Ransomware precursor: abnormal RDP exposure',
        'Honeypot RDP endpoints saw coordinated password spraying from a known bulletproof hoster ASN.',
        'RANSOMWARE',
        'b1000000-0000-4000-8000-000000000005'::uuid,
        'b0000000-0000-4000-8000-000000000010'::uuid,
        'AMBER_STRICT',
        'HIGH',
        'OPEN'
    ),
    (
        'b3000000-0000-4000-8000-000000000003'::uuid,
        'local',
        'b0000000-0000-4000-8000-000000000002'::uuid,
        'Supply chain package typosquat',
        'Build pipeline flagged a near-name dependency published 48h ago; no production deploys affected.',
        'SUPPLY_CHAIN',
        'b1000000-0000-4000-8000-000000000004'::uuid,
        'b0000000-0000-4000-8000-000000000010'::uuid,
        'GREEN',
        'LOW',
        'MITIGATING'
    ),
    (
        'b3000000-0000-4000-8000-000000000004'::uuid,
        'local',
        'b0000000-0000-4000-8000-000000000002'::uuid,
        'DDoS against citizen-facing portal',
        'Mitigation in place; origin shield and rate limits holding; post-incident review scheduled.',
        'DDOS',
        'b1000000-0000-4000-8000-000000000005'::uuid,
        'b0000000-0000-4000-8000-000000000010'::uuid,
        'CLEAR',
        'INFO',
        'RESOLVED'
    ),
    (
        'b3000000-0000-4000-8000-000000000005'::uuid,
        'local',
        'b0000000-0000-4000-8000-000000000002'::uuid,
        'Insider data exfiltration investigation',
        'Anomalous bulk export from internal wiki; account suspended pending HR and legal review.',
        'INSIDER_THREAT',
        'b1000000-0000-4000-8000-000000000008'::uuid,
        'b0000000-0000-4000-8000-000000000011'::uuid,
        'RED',
        'CRITICAL',
        'INVESTIGATING'
    ),
    (
        'b3000000-0000-4000-8000-000000000006'::uuid,
        'local',
        'b0000000-0000-4000-8000-000000000002'::uuid,
        'Zero-day exploitation attempt (blocked)',
        'WAF blocked chained exploit against edge routers; vendor PSIRT engaged under TLP:AMBER.',
        'VULNERABILITY',
        'b1000000-0000-4000-8000-000000000008'::uuid,
        'b0000000-0000-4000-8000-000000000011'::uuid,
        'AMBER',
        'HIGH',
        'OPEN'
    )
) AS e(id, source_instance, sector_context, title, description, event_type, submitter_id, organization_id, tlp, impact, status)
WHERE EXISTS (SELECT 1 FROM fresnel.users u WHERE u.id = e.submitter_id)
ON CONFLICT (id) DO NOTHING;

INSERT INTO fresnel.status_reports (
    id, source_instance, sector_context, scope_type, scope_ref, title, body,
    period_covered_start, period_covered_end, as_of, assessed_status, impact, tlp,
    author_id, organization_id
)
SELECT s.id, s.source_instance, s.sector_context, s.scope_type, s.scope_ref, s.title, s.body,
       s.period_covered_start, s.period_covered_end, s.as_of, s.assessed_status, s.impact, s.tlp,
       s.author_id, s.organization_id
FROM (VALUES
    (
        'b3000000-0000-4000-8000-000000000010'::uuid,
        'local',
        'b0000000-0000-4000-8000-000000000002'::uuid,
        'ORG',
        'b0000000-0000-4000-8000-000000000010'::uuid,
        'Department of Technology — weekly operational posture',
        'Summary: phishing campaign contained; DDoS on citizen portal resolved; supply chain alert cleared after dependency removal. Risk outlook stable with elevated vigilance on RDP exposure.',
        '2026-04-01 00:00:00+00'::timestamptz,
        '2026-04-08 23:59:59+00'::timestamptz,
        '2026-04-09 12:00:00+00'::timestamptz,
        'DEGRADED',
        'MODERATE',
        'AMBER',
        'b1000000-0000-4000-8000-000000000004'::uuid,
        'b0000000-0000-4000-8000-000000000010'::uuid
    )
) AS s(id, source_instance, sector_context, scope_type, scope_ref, title, body,
       period_covered_start, period_covered_end, as_of, assessed_status, impact, tlp,
       author_id, organization_id)
WHERE EXISTS (SELECT 1 FROM fresnel.users u WHERE u.id = s.author_id)
ON CONFLICT (id) DO NOTHING;

INSERT INTO fresnel.status_report_events (status_report_id, event_id)
SELECT l.status_report_id, l.event_id
FROM (VALUES
    ('b3000000-0000-4000-8000-000000000010'::uuid, 'b3000000-0000-4000-8000-000000000001'::uuid),
    ('b3000000-0000-4000-8000-000000000010'::uuid, 'b3000000-0000-4000-8000-000000000002'::uuid),
    ('b3000000-0000-4000-8000-000000000010'::uuid, 'b3000000-0000-4000-8000-000000000004'::uuid)
) AS l(status_report_id, event_id)
WHERE EXISTS (SELECT 1 FROM fresnel.status_reports r WHERE r.id = l.status_report_id)
  AND EXISTS (SELECT 1 FROM fresnel.events ev WHERE ev.id = l.event_id)
ON CONFLICT (status_report_id, event_id) DO NOTHING;
