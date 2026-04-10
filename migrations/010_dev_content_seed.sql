-- Comprehensive development content: events across all orgs, event updates,
-- status reports per org, campaigns with linked events, correlations.
-- All UUIDs use the b4*/b5*/b6* range to avoid clashing with earlier seeds.
-- Idempotent: ON CONFLICT DO NOTHING throughout.

SET search_path TO fresnel, public;

-- ============================================================================
-- EVENTS (12 additional events spread across all 7 organizations)
-- ============================================================================

-- Dept of Technology (b0..0010) — Federal / Government
INSERT INTO events (id, source_instance, sector_context, title, description, event_type, submitter_id, organization_id, tlp, impact, status, created_at) VALUES
('b4000000-0000-4000-8000-000000000001'::uuid, 'local', 'b0000000-0000-4000-8000-000000000002'::uuid,
 'Unauthorised access to legacy HR system',
 'Audit logs show an external IP authenticated via a dormant service account. The account has been disabled and credential rotation is underway. Forensic imaging of the host is in progress.',
 'UNAUTHORIZED_ACCESS', 'b1000000-0000-4000-8000-000000000006'::uuid, 'b0000000-0000-4000-8000-000000000010'::uuid,
 'AMBER', 'HIGH', 'INVESTIGATING', now() - interval '6 days'),

('b4000000-0000-4000-8000-000000000002'::uuid, 'local', 'b0000000-0000-4000-8000-000000000002'::uuid,
 'Certificate transparency alert for gov subdomain',
 'CT monitor detected issuance of a certificate for portal.internal.gov by an unauthorised CA. Domain registrar contacted; pre-existing HSTS pins prevented user impact.',
 'OTHER', 'b1000000-0000-4000-8000-000000000005'::uuid, 'b0000000-0000-4000-8000-000000000010'::uuid,
 'GREEN', 'LOW', 'RESOLVED', now() - interval '14 days')
ON CONFLICT (id) DO NOTHING;

-- National Security Agency (b0..0011) — Federal / Government
INSERT INTO events (id, source_instance, sector_context, title, description, event_type, submitter_id, organization_id, tlp, impact, status, created_at) VALUES
('b4000000-0000-4000-8000-000000000003'::uuid, 'local', 'b0000000-0000-4000-8000-000000000002'::uuid,
 'Spear-phishing with weaponised PDF attachment',
 'Three recipients in the strategic analysis unit received PDF documents exploiting CVE-2025-XXXXX. Sandbox detonation confirmed C2 callback. All endpoints quarantined within 8 minutes.',
 'PHISHING', 'b1000000-0000-4000-8000-000000000008'::uuid, 'b0000000-0000-4000-8000-000000000011'::uuid,
 'AMBER_STRICT', 'CRITICAL', 'MITIGATING', now() - interval '3 days'),

('b4000000-0000-4000-8000-000000000004'::uuid, 'local', 'b0000000-0000-4000-8000-000000000002'::uuid,
 'DNS exfiltration via encoded TXT queries',
 'Network analytics identified anomalous TXT query patterns to a recently registered domain. Traffic volume suggests approximately 12 MB exfiltrated over 72 hours.',
 'DATA_BREACH', 'b1000000-0000-4000-8000-000000000008'::uuid, 'b0000000-0000-4000-8000-000000000011'::uuid,
 'RED', 'CRITICAL', 'INVESTIGATING', now() - interval '1 day')
ON CONFLICT (id) DO NOTHING;

-- State IT Authority (b0..0012) — State / Government
INSERT INTO events (id, source_instance, sector_context, title, description, event_type, submitter_id, organization_id, tlp, impact, status, created_at) VALUES
('b4000000-0000-4000-8000-000000000005'::uuid, 'local', 'b0000000-0000-4000-8000-000000000003'::uuid,
 'Misconfigured cloud storage bucket publicly accessible',
 'Routine external scan found an S3-compatible bucket exposing non-sensitive operational documents. Bucket ACL corrected within 20 minutes of discovery. No PII confirmed in exposed objects.',
 'VULNERABILITY', 'b1000000-0000-4000-8000-000000000004'::uuid, 'b0000000-0000-4000-8000-000000000012'::uuid,
 'GREEN', 'LOW', 'RESOLVED', now() - interval '10 days')
ON CONFLICT (id) DO NOTHING;

-- Central Bank (b0..0013) — Finance
INSERT INTO events (id, source_instance, sector_context, title, description, event_type, submitter_id, organization_id, tlp, impact, status, created_at) VALUES
('b4000000-0000-4000-8000-000000000006'::uuid, 'local', 'b0000000-0000-4000-8000-000000000004'::uuid,
 'Credential stuffing against online banking API',
 'Automated attack using breached credential lists from third-party data brokers. WAF rate limiting activated; 14 customer accounts locked as precaution. No confirmed account takeover.',
 'UNAUTHORIZED_ACCESS', 'b1000000-0000-4000-8000-000000000004'::uuid, 'b0000000-0000-4000-8000-000000000013'::uuid,
 'AMBER', 'MODERATE', 'OPEN', now() - interval '2 days'),

('b4000000-0000-4000-8000-000000000007'::uuid, 'local', 'b0000000-0000-4000-8000-000000000004'::uuid,
 'SWIFT messaging anomaly under investigation',
 'Reconciliation flagged two outbound MT103 messages with non-standard field formatting. Likely benign (software update) but under review per mandatory incident process.',
 'OTHER', 'b1000000-0000-4000-8000-000000000004'::uuid, 'b0000000-0000-4000-8000-000000000013'::uuid,
 'AMBER_STRICT', 'HIGH', 'INVESTIGATING', now() - interval '12 hours')
ON CONFLICT (id) DO NOTHING;

-- Financial Regulatory Authority (b0..0014) — Finance
INSERT INTO events (id, source_instance, sector_context, title, description, event_type, submitter_id, organization_id, tlp, impact, status, created_at) VALUES
('b4000000-0000-4000-8000-000000000008'::uuid, 'local', 'b0000000-0000-4000-8000-000000000004'::uuid,
 'Phishing kit impersonating regulatory portal',
 'Takedown request submitted to hosting provider. Lookalike domain registered 5 days ago with Let''s Encrypt certificate. Social media alert published to regulated entities.',
 'PHISHING', 'b1000000-0000-4000-8000-000000000004'::uuid, 'b0000000-0000-4000-8000-000000000014'::uuid,
 'GREEN', 'MODERATE', 'MITIGATING', now() - interval '4 days')
ON CONFLICT (id) DO NOTHING;

-- National Grid Operator (b0..0015) — Energy / Critical Infrastructure
INSERT INTO events (id, source_instance, sector_context, title, description, event_type, submitter_id, organization_id, tlp, impact, status, created_at) VALUES
('b4000000-0000-4000-8000-000000000009'::uuid, 'local', 'b0000000-0000-4000-8000-000000000006'::uuid,
 'SCADA network scan from compromised contractor VPN',
 'IDS alerted on port scans targeting Modbus/TCP endpoints from a contractor VPN segment. Contractor credentials revoked. No evidence of process manipulation.',
 'UNAUTHORIZED_ACCESS', 'b1000000-0000-4000-8000-000000000004'::uuid, 'b0000000-0000-4000-8000-000000000015'::uuid,
 'AMBER_STRICT', 'CRITICAL', 'INVESTIGATING', now() - interval '1 day'),

('b4000000-0000-4000-8000-000000000010'::uuid, 'local', 'b0000000-0000-4000-8000-000000000006'::uuid,
 'Firmware integrity check failure on substation RTUs',
 'Scheduled integrity validation of remote terminal unit firmware flagged hash mismatches on 3 of 47 units. No operational impact. Physical inspection team dispatched.',
 'VULNERABILITY', 'b1000000-0000-4000-8000-000000000004'::uuid, 'b0000000-0000-4000-8000-000000000015'::uuid,
 'AMBER', 'HIGH', 'OPEN', now() - interval '8 hours')
ON CONFLICT (id) DO NOTHING;

-- Telecom Authority (b0..0016) — Telecommunications / Critical Infrastructure
INSERT INTO events (id, source_instance, sector_context, title, description, event_type, submitter_id, organization_id, tlp, impact, status, created_at) VALUES
('b4000000-0000-4000-8000-000000000011'::uuid, 'local', 'b0000000-0000-4000-8000-000000000007'::uuid,
 'BGP hijack affecting national prefix space',
 'Route collector observed unauthorised origination of national /16 prefixes from an AS not in the authorised set. RPKI ROV rejected at 73% of peers. ISP coordination underway.',
 'DDOS', 'b1000000-0000-4000-8000-000000000004'::uuid, 'b0000000-0000-4000-8000-000000000016'::uuid,
 'GREEN', 'HIGH', 'MITIGATING', now() - interval '5 hours'),

('b4000000-0000-4000-8000-000000000012'::uuid, 'local', 'b0000000-0000-4000-8000-000000000007'::uuid,
 'SS7 probing against mobile subscriber database',
 'Signalling firewall blocked anomalous SRI-SM and PSI queries from an international roaming partner. Pattern consistent with location tracking reconnaissance.',
 'UNAUTHORIZED_ACCESS', 'b1000000-0000-4000-8000-000000000004'::uuid, 'b0000000-0000-4000-8000-000000000016'::uuid,
 'AMBER', 'MODERATE', 'OPEN', now() - interval '2 days')
ON CONFLICT (id) DO NOTHING;


-- ============================================================================
-- EVENT UPDATES (timeline entries for selected events)
-- ============================================================================

INSERT INTO event_updates (id, event_id, author_id, body, tlp, impact_change, status_change, created_at) VALUES
-- Updates on "Unauthorised access to legacy HR system"
('b4100000-0000-4000-8000-000000000001'::uuid, 'b4000000-0000-4000-8000-000000000001'::uuid,
 'b1000000-0000-4000-8000-000000000005'::uuid,
 'Dormant service account confirmed as a remnant from a 2019 contractor integration. Password last changed 2021-03-14. Account disabled in AD and LDAP.',
 'AMBER', NULL, NULL, now() - interval '5 days 18 hours'),

('b4100000-0000-4000-8000-000000000002'::uuid, 'b4000000-0000-4000-8000-000000000001'::uuid,
 'b1000000-0000-4000-8000-000000000005'::uuid,
 'Forensic imaging complete. No lateral movement detected. Attacker IP traced to a residential proxy service. Escalating impact to CRITICAL pending data classification review.',
 'AMBER', 'CRITICAL', NULL, now() - interval '4 days'),

-- Updates on "Credential phishing against executive mailboxes" (from 009 seed)
('b4100000-0000-4000-8000-000000000003'::uuid, 'b3000000-0000-4000-8000-000000000001'::uuid,
 'b1000000-0000-4000-8000-000000000005'::uuid,
 'OAuth consent phishing page taken down by hosting provider. 2 of 11 targeted users had granted consent tokens; tokens revoked. Recommending mandatory MFA review.',
 'AMBER', NULL, NULL, now() - interval '2 days'),

-- Updates on "Spear-phishing with weaponised PDF"
('b4100000-0000-4000-8000-000000000004'::uuid, 'b4000000-0000-4000-8000-000000000003'::uuid,
 'b1000000-0000-4000-8000-000000000008'::uuid,
 'C2 infrastructure mapped to 4 IPs across 2 ASNs. Indicator sharing in progress with sector partners under TLP:AMBER:STRICT. Yara rules deployed to all endpoints.',
 'AMBER_STRICT', NULL, NULL, now() - interval '2 days'),

('b4100000-0000-4000-8000-000000000005'::uuid, 'b4000000-0000-4000-8000-000000000003'::uuid,
 'b1000000-0000-4000-8000-000000000008'::uuid,
 'All 3 quarantined endpoints reimaged and returned to service. No evidence of persistent implant. Moving to MITIGATING while monitoring for reinfection.',
 'AMBER_STRICT', NULL, 'MITIGATING', now() - interval '1 day'),

-- Updates on "SCADA network scan"
('b4100000-0000-4000-8000-000000000006'::uuid, 'b4000000-0000-4000-8000-000000000009'::uuid,
 'b1000000-0000-4000-8000-000000000004'::uuid,
 'Contractor notified per incident response agreement. Their SOC confirmed the VPN account was compromised via a personal device. Joint forensic review initiated.',
 'AMBER_STRICT', NULL, NULL, now() - interval '18 hours'),

-- Updates on "BGP hijack"
('b4100000-0000-4000-8000-000000000007'::uuid, 'b4000000-0000-4000-8000-000000000011'::uuid,
 'b1000000-0000-4000-8000-000000000004'::uuid,
 'Originating AS identified as a bullet-proof hosting provider. Upstream transit providers contacted for prefix filtering. RPKI coverage being extended to remaining /24 announcements.',
 'GREEN', NULL, NULL, now() - interval '3 hours'),

-- Updates on "Credential stuffing against online banking"
('b4100000-0000-4000-8000-000000000008'::uuid, 'b4000000-0000-4000-8000-000000000006'::uuid,
 'b1000000-0000-4000-8000-000000000004'::uuid,
 'Analysis of 48-hour attack window: 2.3M authentication attempts from 14K unique IPs. Credential list likely sourced from the May 2025 SocialPlatform breach. Affected customers notified.',
 'AMBER', NULL, NULL, now() - interval '1 day')
ON CONFLICT (id) DO NOTHING;


-- ============================================================================
-- STATUS REPORTS (one per org with active events)
-- ============================================================================

INSERT INTO status_reports (id, source_instance, sector_context, scope_type, scope_ref, title, body, period_covered_start, period_covered_end, as_of, assessed_status, impact, tlp, author_id, organization_id) VALUES

-- National Security Agency
('b5000000-0000-4000-8000-000000000001'::uuid, 'local', 'b0000000-0000-4000-8000-000000000002'::uuid,
 'ORG', 'b0000000-0000-4000-8000-000000000011'::uuid,
 'National Security Agency — weekly posture',
 'Two high-severity incidents in progress. Spear-phishing campaign contained but C2 infrastructure still active. DNS exfiltration investigation ongoing with data classification pending. Insider threat case (from prior week) referred to internal affairs. Overall posture: IMPAIRED.',
 now() - interval '7 days', now(), now() - interval '2 hours',
 'IMPAIRED', 'CRITICAL', 'AMBER_STRICT',
 'b1000000-0000-4000-8000-000000000008'::uuid, 'b0000000-0000-4000-8000-000000000011'::uuid),

-- State IT Authority
('b5000000-0000-4000-8000-000000000002'::uuid, 'local', 'b0000000-0000-4000-8000-000000000003'::uuid,
 'ORG', 'b0000000-0000-4000-8000-000000000012'::uuid,
 'State IT Authority — weekly posture',
 'Cloud storage misconfiguration resolved. No ongoing incidents. Preventive measures: automated bucket policy scanner deployed to CI/CD pipeline. Posture: NORMAL.',
 now() - interval '7 days', now(), now() - interval '4 hours',
 'NORMAL', 'LOW', 'GREEN',
 'b1000000-0000-4000-8000-000000000004'::uuid, 'b0000000-0000-4000-8000-000000000012'::uuid),

-- Central Bank
('b5000000-0000-4000-8000-000000000003'::uuid, 'local', 'b0000000-0000-4000-8000-000000000004'::uuid,
 'ORG', 'b0000000-0000-4000-8000-000000000013'::uuid,
 'Central Bank — weekly posture',
 'Credential stuffing campaign ongoing with WAF mitigation in place. SWIFT anomaly under mandatory investigation — likely benign. Customer impact limited to 14 precautionary account locks. Posture: DEGRADED.',
 now() - interval '7 days', now(), now() - interval '1 hour',
 'DEGRADED', 'MODERATE', 'AMBER',
 'b1000000-0000-4000-8000-000000000004'::uuid, 'b0000000-0000-4000-8000-000000000013'::uuid),

-- Financial Regulatory Authority
('b5000000-0000-4000-8000-000000000004'::uuid, 'local', 'b0000000-0000-4000-8000-000000000004'::uuid,
 'ORG', 'b0000000-0000-4000-8000-000000000014'::uuid,
 'Financial Regulatory Authority — weekly posture',
 'Phishing kit takedown in progress. Social media advisory issued to regulated entities. No confirmed credential compromise. Posture: DEGRADED while takedown is pending.',
 now() - interval '7 days', now(), now() - interval '3 hours',
 'DEGRADED', 'MODERATE', 'GREEN',
 'b1000000-0000-4000-8000-000000000004'::uuid, 'b0000000-0000-4000-8000-000000000014'::uuid),

-- National Grid Operator
('b5000000-0000-4000-8000-000000000005'::uuid, 'local', 'b0000000-0000-4000-8000-000000000006'::uuid,
 'ORG', 'b0000000-0000-4000-8000-000000000015'::uuid,
 'National Grid Operator — weekly posture',
 'Two active incidents: SCADA network scan from compromised contractor VPN (investigation ongoing, no process impact) and firmware integrity failures on 3 RTUs (physical inspection dispatched). OT network segmentation held. Posture: IMPAIRED.',
 now() - interval '7 days', now(), now() - interval '30 minutes',
 'IMPAIRED', 'CRITICAL', 'AMBER_STRICT',
 'b1000000-0000-4000-8000-000000000004'::uuid, 'b0000000-0000-4000-8000-000000000015'::uuid),

-- Telecom Authority
('b5000000-0000-4000-8000-000000000006'::uuid, 'local', 'b0000000-0000-4000-8000-000000000007'::uuid,
 'ORG', 'b0000000-0000-4000-8000-000000000016'::uuid,
 'Telecom Authority — weekly posture',
 'BGP hijack mitigation ongoing — RPKI ROV effective at majority of peers, coordination with upstream transits in progress. SS7 probing blocked by signalling firewall; roaming partner notified. Posture: DEGRADED.',
 now() - interval '7 days', now(), now() - interval '1 hour',
 'DEGRADED', 'HIGH', 'GREEN',
 'b1000000-0000-4000-8000-000000000004'::uuid, 'b0000000-0000-4000-8000-000000000016'::uuid),

-- Sector-level report: Government
('b5000000-0000-4000-8000-000000000010'::uuid, 'local', 'b0000000-0000-4000-8000-000000000001'::uuid,
 'SECTOR', 'b0000000-0000-4000-8000-000000000001'::uuid,
 'Government sector — weekly consolidated posture',
 'Three orgs reporting: Dept of Technology (DEGRADED — phishing contained, RDP exposure under watch), National Security Agency (IMPAIRED — active spear-phishing and DNS exfiltration), State IT Authority (NORMAL). Sector posture reflects the weighted average. Recommend sector-wide dormant-account audit.',
 now() - interval '7 days', now(), now() - interval '1 hour',
 'DEGRADED', 'HIGH', 'AMBER',
 'b1000000-0000-4000-8000-000000000002'::uuid, 'b0000000-0000-4000-8000-000000000010'::uuid)

ON CONFLICT (id) DO NOTHING;

-- Link events to status reports
INSERT INTO status_report_events (status_report_id, event_id) VALUES
-- NSA report ← its events
('b5000000-0000-4000-8000-000000000001'::uuid, 'b4000000-0000-4000-8000-000000000003'::uuid),
('b5000000-0000-4000-8000-000000000001'::uuid, 'b4000000-0000-4000-8000-000000000004'::uuid),
('b5000000-0000-4000-8000-000000000001'::uuid, 'b3000000-0000-4000-8000-000000000005'::uuid),
-- Central Bank report ← its events
('b5000000-0000-4000-8000-000000000003'::uuid, 'b4000000-0000-4000-8000-000000000006'::uuid),
('b5000000-0000-4000-8000-000000000003'::uuid, 'b4000000-0000-4000-8000-000000000007'::uuid),
-- Grid Operator report ← its events
('b5000000-0000-4000-8000-000000000005'::uuid, 'b4000000-0000-4000-8000-000000000009'::uuid),
('b5000000-0000-4000-8000-000000000005'::uuid, 'b4000000-0000-4000-8000-000000000010'::uuid),
-- Telecom report ← its events
('b5000000-0000-4000-8000-000000000006'::uuid, 'b4000000-0000-4000-8000-000000000011'::uuid),
('b5000000-0000-4000-8000-000000000006'::uuid, 'b4000000-0000-4000-8000-000000000012'::uuid),
-- Government sector report ← key events from child orgs
('b5000000-0000-4000-8000-000000000010'::uuid, 'b3000000-0000-4000-8000-000000000001'::uuid),
('b5000000-0000-4000-8000-000000000010'::uuid, 'b4000000-0000-4000-8000-000000000001'::uuid),
('b5000000-0000-4000-8000-000000000010'::uuid, 'b4000000-0000-4000-8000-000000000003'::uuid)
ON CONFLICT (status_report_id, event_id) DO NOTHING;


-- ============================================================================
-- CAMPAIGNS (3 campaigns grouping related events)
-- ============================================================================

INSERT INTO campaigns (id, title, description, tlp, status, created_by, organization_id, created_at) VALUES

('b6000000-0000-4000-8000-000000000001'::uuid,
 'Coordinated credential harvesting wave',
 'Multiple organisations across government and finance sectors have reported credential-based attacks in the same week. Techniques include OAuth consent phishing, credential stuffing from third-party breach data, and targeted spear-phishing with weaponised attachments. Shared infrastructure (proxy services, bullet-proof hosting) suggests a common operator or tooling supply chain.',
 'AMBER', 'ACTIVE',
 'b1000000-0000-4000-8000-000000000002'::uuid,
 'b0000000-0000-4000-8000-000000000010'::uuid,
 now() - interval '3 days'),

('b6000000-0000-4000-8000-000000000002'::uuid,
 'Critical infrastructure OT/ICS reconnaissance',
 'Energy and telecommunications sectors reporting scanning and probing activity against operational technology systems. SCADA port scans, RTU firmware anomalies, and signalling protocol reconnaissance observed in the same period. No confirmed process manipulation, but the pattern warrants coordinated monitoring.',
 'AMBER_STRICT', 'ACTIVE',
 'b1000000-0000-4000-8000-000000000004'::uuid,
 'b0000000-0000-4000-8000-000000000010'::uuid,
 now() - interval '1 day'),

('b6000000-0000-4000-8000-000000000003'::uuid,
 'Financial sector regulatory impersonation',
 'Phishing kits and lookalike domains impersonating financial regulators detected across the sector. Takedown operations in progress. This campaign tracks the infrastructure, IOCs, and affected regulated entities.',
 'GREEN', 'ACTIVE',
 'b1000000-0000-4000-8000-000000000004'::uuid,
 'b0000000-0000-4000-8000-000000000010'::uuid,
 now() - interval '4 days')

ON CONFLICT (id) DO NOTHING;

-- Link events to campaigns
INSERT INTO campaign_events (campaign_id, event_id, linked_by, linked_at) VALUES
-- Credential harvesting campaign
('b6000000-0000-4000-8000-000000000001'::uuid, 'b3000000-0000-4000-8000-000000000001'::uuid, 'b1000000-0000-4000-8000-000000000002'::uuid, now() - interval '3 days'),
('b6000000-0000-4000-8000-000000000001'::uuid, 'b4000000-0000-4000-8000-000000000003'::uuid, 'b1000000-0000-4000-8000-000000000002'::uuid, now() - interval '2 days'),
('b6000000-0000-4000-8000-000000000001'::uuid, 'b4000000-0000-4000-8000-000000000006'::uuid, 'b1000000-0000-4000-8000-000000000002'::uuid, now() - interval '2 days'),
('b6000000-0000-4000-8000-000000000001'::uuid, 'b4000000-0000-4000-8000-000000000001'::uuid, 'b1000000-0000-4000-8000-000000000002'::uuid, now() - interval '1 day'),
-- OT/ICS reconnaissance campaign
('b6000000-0000-4000-8000-000000000002'::uuid, 'b4000000-0000-4000-8000-000000000009'::uuid, 'b1000000-0000-4000-8000-000000000004'::uuid, now() - interval '1 day'),
('b6000000-0000-4000-8000-000000000002'::uuid, 'b4000000-0000-4000-8000-000000000010'::uuid, 'b1000000-0000-4000-8000-000000000004'::uuid, now() - interval '8 hours'),
('b6000000-0000-4000-8000-000000000002'::uuid, 'b4000000-0000-4000-8000-000000000011'::uuid, 'b1000000-0000-4000-8000-000000000004'::uuid, now() - interval '5 hours'),
('b6000000-0000-4000-8000-000000000002'::uuid, 'b4000000-0000-4000-8000-000000000012'::uuid, 'b1000000-0000-4000-8000-000000000004'::uuid, now() - interval '2 hours'),
-- Financial regulatory impersonation campaign
('b6000000-0000-4000-8000-000000000003'::uuid, 'b4000000-0000-4000-8000-000000000008'::uuid, 'b1000000-0000-4000-8000-000000000004'::uuid, now() - interval '4 days')
ON CONFLICT (campaign_id, event_id) DO NOTHING;


-- ============================================================================
-- CORRELATIONS (analyst-created links between related events)
-- ============================================================================

-- event_a_id < event_b_id is enforced by CHECK constraint, so order UUIDs accordingly
INSERT INTO correlations (id, event_a_id, event_b_id, label, correlation_type, created_by_user, created_at) VALUES

('b6100000-0000-4000-8000-000000000001'::uuid,
 'b3000000-0000-4000-8000-000000000001'::uuid, 'b4000000-0000-4000-8000-000000000003'::uuid,
 'Both events use credential harvesting techniques targeting government personnel in the same week',
 'MANUAL', 'b1000000-0000-4000-8000-000000000002'::uuid, now() - interval '2 days'),

('b6100000-0000-4000-8000-000000000002'::uuid,
 'b3000000-0000-4000-8000-000000000002'::uuid, 'b4000000-0000-4000-8000-000000000001'::uuid,
 'RDP exposure and unauthorised access may share common attacker infrastructure (same residential proxy ASN)',
 'MANUAL', 'b1000000-0000-4000-8000-000000000005'::uuid, now() - interval '4 days'),

('b6100000-0000-4000-8000-000000000003'::uuid,
 'b4000000-0000-4000-8000-000000000009'::uuid, 'b4000000-0000-4000-8000-000000000010'::uuid,
 'SCADA scan and firmware anomalies at the same facility within 48 hours — possible causal relationship',
 'MANUAL', 'b1000000-0000-4000-8000-000000000004'::uuid, now() - interval '6 hours'),

('b6100000-0000-4000-8000-000000000004'::uuid,
 'b4000000-0000-4000-8000-000000000011'::uuid, 'b4000000-0000-4000-8000-000000000012'::uuid,
 'BGP hijack and SS7 probing both target telecom infrastructure in the same timeframe — coordinated reconnaissance?',
 'MANUAL', 'b1000000-0000-4000-8000-000000000004'::uuid, now() - interval '2 hours')

ON CONFLICT (id) DO NOTHING;


-- ============================================================================
-- TLP:RED RECIPIENTS (for the RED events)
-- ============================================================================

-- DNS exfiltration (b4..0004) is TLP:RED — grant visibility to platform root and gov sector root
INSERT INTO tlp_red_recipients (id, resource_type, resource_id, recipient_user_id, granted_by) VALUES
('b6200000-0000-4000-8000-000000000001'::uuid, 'event', 'b4000000-0000-4000-8000-000000000004'::uuid,
 'b1000000-0000-4000-8000-000000000001'::uuid, 'b1000000-0000-4000-8000-000000000008'::uuid),
('b6200000-0000-4000-8000-000000000002'::uuid, 'event', 'b4000000-0000-4000-8000-000000000004'::uuid,
 'b1000000-0000-4000-8000-000000000002'::uuid, 'b1000000-0000-4000-8000-000000000008'::uuid),
('b6200000-0000-4000-8000-000000000003'::uuid, 'event', 'b4000000-0000-4000-8000-000000000004'::uuid,
 'b1000000-0000-4000-8000-000000000008'::uuid, 'b1000000-0000-4000-8000-000000000008'::uuid)
ON CONFLICT (resource_type, resource_id, recipient_user_id) DO NOTHING;

-- Insider threat (b3..0005 from 009) is also TLP:RED — grant to platform root
INSERT INTO tlp_red_recipients (id, resource_type, resource_id, recipient_user_id, granted_by) VALUES
('b6200000-0000-4000-8000-000000000004'::uuid, 'event', 'b3000000-0000-4000-8000-000000000005'::uuid,
 'b1000000-0000-4000-8000-000000000001'::uuid, 'b1000000-0000-4000-8000-000000000008'::uuid),
('b6200000-0000-4000-8000-000000000005'::uuid, 'event', 'b3000000-0000-4000-8000-000000000005'::uuid,
 'b1000000-0000-4000-8000-000000000008'::uuid, 'b1000000-0000-4000-8000-000000000008'::uuid)
ON CONFLICT (resource_type, resource_id, recipient_user_id) DO NOTHING;
