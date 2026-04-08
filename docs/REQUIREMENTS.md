# Fresnel — Platform Requirements Specification

*A Fresnel lens lets the lighthouse cast light further and more efficiently.*

**Version**: 0.1
**Status**: Specification draft — pending final review before architecture phase
**Scope**: Cyber situational awareness platform for hierarchical multi-organization operational status tracking and sharing

---

## 1. Product Definition

### 1.1 What Fresnel Is

A **cyber situational awareness platform** for tracking and sharing operational status across a hierarchical structure of sectors, verticals, and organizations. It serves decision-makers — people involved in cyber governance and organizational impact assessment who need to understand *the situation*, not the technical details.

Users care about: outcomes, recovery objectives, impact, timelines, affected services, process status, cross-organizational awareness, and trends over time.

Users do not care about: indicators of compromise, IP addresses, STIX objects, ATT&CK techniques, malware hashes, or analyst-grade threat intelligence.

### 1.2 What Fresnel Is Not

- **Not a Threat Intelligence Platform (TIP).** It does not ingest, store, or share IOCs, TTPs, or technical indicators. It is not a replacement for MISP, OpenCTI, or ThreatConnect.
- **Not a STIX/TAXII endpoint.** If integration with STIX/TAXII ecosystems is needed in the future, it would be through an external adapter, not native support.
- **Not a SIEM/SOAR tool.** It does not collect logs, trigger automated responses, or integrate with detection infrastructure.

Fresnel is the **dashboard layer above all of the above** — the place where decision-makers get situational awareness without drowning in technical detail.

### 1.3 Design Philosophy

- **Security is non-negotiable.** Functionality is PoC-grade; security is production-grade. The platform must be usable for demonstrations with real stakeholders handling real or realistic data.
- **Architecture survives; code doesn't.** The PoC code can be thrown out. The architecture, data model, API contracts, and authorization schema cannot. Every feature — including deferred ones — must be a short distance from implementation given the architecture.
- **On-prem sovereign.** Zero public cloud dependencies. Bundled identity provider. Deployable on vSphere. Internet-facing by default but isolatable.

---

## 2. Organizational Model

### 2.1 Hierarchy

```
Platform (single instance, single root)
  └── Sector (e.g., "Government", "Finance", "Critical Infrastructure")
        └── Vertical (e.g., "Federal", "State", "Municipal")
              └── Organization (e.g., "Department of X", "Agency Y")
```

- Hierarchy is fixed at three levels below platform. Organizations do not nest further.
- **An organization can be a member of multiple sectors.** Each sector membership requires a distinct root user for that org within that sector's governance context.
- Each organizational unit (platform, sector, vertical, org) has: unique ID, display name, status (active/inactive/suspended), creation timestamp.

### 2.2 Root User Model

Every organizational unit has a **root user** — a principal with maximum authority over their constituency, covering both data plane and control plane.

| Scope | Root Authority |
|---|---|
| Platform Root | Full control and data plane access across the entire platform. Can reassign any root. |
| Sector Root | Full control and data plane access within their sector, including all verticals and orgs. |
| Vertical Root | Full control and data plane access within their vertical and its orgs. |
| Org Root | Full control and data plane access within their organization. |

**Root reassignment rules:**
- A root can reassign themselves (nominate a successor).
- The root of the parent unit can reassign any child root.
- Both operations are audited as high-severity events.
- No group vote mechanism. Accountability flows through the hierarchy.
- Detailed succession procedures (root incapacitation, contested reassignment) deferred to Phase 1.

**Multi-sector org governance:**
- If Org A is in both Sector X and Sector Y, Org A has two sector-specific root contexts.
- Org A also has a single primary Org Root who governs internal affairs (user management, policy, data).
- **Events and status reports are sector-bound.** When a user in a multi-sector org creates an event or status report, they select which sector context it belongs to. Sector X's root can only see Org A's events created within Sector X's context.
- The Org Root can see all events across all sector contexts (they govern the org, not the sector).
- The UI presents a sector context selector for users in multi-sector orgs.

### 2.3 Default Trust Model (PoC)

For the PoC, the trust model is **full hierarchical visibility by default**:

- Root users have full data-plane and control-plane access to everything within their constituency.
- No sovereign mode in the PoC. All participating organizations accept hierarchical visibility as a condition of participation.
- This is acceptable because initial users either trust the platform or only post content they consider shareable.

**Architectural constraint**: The data model and Cedar policy schema must be designed so that sovereign mode and federation can be activated without data migration or schema changes. The authorization layer evaluates policies dynamically and does not hardcode the "root sees everything" assumption.

### 2.4 Sovereign Mode (Phase 2 — Architect For, Do Not Implement)

An organization activates sovereign mode to restrict data-plane access from all principals outside the organization, including parent roots.

When sovereign mode is active:
- External principals (including sector/platform roots) cannot read event content, status reports, or attachments owned by the sovereign org.
- External principals *can* see audit log metadata (who did what, when — not event content or IDs). This is an accepted inference channel: activity patterns are visible upward, content is not.
- Campaigns containing the sovereign org's events show: "N organizations have restricted content." Org admins of non-sovereign orgs in the campaign can see *which* organizations have restricted content and reach out diplomatically.
- Correlations to sovereign events are invisible to unauthorized principals.
- The hierarchical dashboard shows the sovereign org's status as "restricted" rather than its actual computed status, unless the org has published a shareable status report.

### 2.5 Federation Model (Phase 2 — Architect For, Do Not Implement)

Federation allows an organization to run its own Fresnel instance on its own infrastructure while participating in the broader platform ecosystem.

#### 2.5.1 Federation Scope

- Federation operates **at the organization level only.**
- A federated org runs a complete Fresnel instance locally (its own database, Keycloak, AI agents).
- The federated instance connects to the hub via a federation protocol.

#### 2.5.2 Federation Trust Model

**Outbound (federated org → hub):**
- The federated org chooses what to publish. Published events/status reports appear on the hub as first-class objects owned by the federated org.
- The org can sanitize before publishing. Published content can have different TLP/impact than the internal version.
- The org controls when to publish, what to publish, and can retract.

**Inbound (hub → federated org):**
- The hub sends the federated org events/reports shared via normal access policies.
- The hub can send **information requests** — pattern-based queries. The federated instance evaluates locally and the org admin approves/rejects responses.

#### 2.5.3 Federation Protocol Requirements (Design-Level)

- Mutual TLS authentication between instances.
- Signed event payloads (non-repudiation).
- Asynchronous request-response for information requests.
- Heartbeat/presence signaling.
- Event/report lifecycle synchronization.
- **Instance discovery**: Manual. Federation is established by admin action on both sides — exchange of endpoint URLs, mTLS certificates, and trust configuration. No automated discovery.

API design constraints for federation readiness:
- Event and status report IDs are globally unique UUIDs.
- All resources include a `source_instance` field (default: local).
- API supports filtering/scoping by source instance.
- Federation endpoints reserved in URL schema (return 501 in PoC).

### 2.6 Break-Glass Procedure (Phase 1 — Architect For, Do Not Implement in PoC)

Break-glass allows a parent root to override sovereign mode when an org is non-responsive. It is **not** applicable when the org has actively responded and rejected a request.

#### 2.6.1 Preconditions

Break-glass can only be initiated when:
1. A request for access/information was sent to the org root.
2. The org root has **not responded** within a defined window (configurable, default 48 hours).
3. The requesting principal is the direct parent root.

Break-glass **cannot** be initiated when:
- The org root responded and explicitly rejected the request.
- The requesting principal is not in the direct chain of command.

#### 2.6.2 Procedure

1. Parent root initiates, providing written justification.
2. High-severity audit event generated, visible to org root, all org admins, and platform root.
3. Notification sent to org root and all org admins (email).
4. After cooling-off period (configurable, default 4 hours), parent root gains time-limited data-plane access (configurable, default 24 hours).
5. All access during break-glass window logged with elevated audit detail.
6. After window expires, access reverts. Org can extend, grant permanent access, or re-deny.

#### 2.6.3 Abuse Prevention

- Break-glass events visible to platform root for oversight.
- Repeated break-glass against the same org triggers automatic review flag.
- Org root can escalate complaints to platform root.

---

## 3. Domain Model

### 3.1 Conceptual Hierarchy

The domain model has two primary dimensions:

**Vertical (hierarchical):**
```
Status Reports (situational state of an org, vertical, or sector)
  └── Events (operational incidents, disruptions, situations)
        └── Event Updates (chronological progress log on a specific event)
```

**Horizontal (cross-cutting):**
```
Campaigns (group related events across sectors and orgs)
```

Status Reports are the highest-level first-class objects. They represent the *assessed state* of an organizational unit at a point in time. Events are the underlying operational detail. Event Updates are the granular progress log within an event. Campaigns cut across the hierarchy, grouping related events regardless of sector.

### 3.2 Status Reports

A Status Report is a periodic situational assessment that assigns an operational state to an organization, vertical, or sector. It is the primary object displayed on the hierarchical dashboard.

| Field | Type | Description |
|---|---|---|
| id | UUID | Globally unique |
| source_instance | String | Instance ID (federation-ready). Default: local. |
| sector_context | Sector ref | Which sector this report belongs to. Immutable after creation. |
| scope | Enum + Ref | What this report covers: ORG (+ org ref), VERTICAL (+ vertical ref), or SECTOR (+ sector ref) |
| title | String | Report title (e.g., "Weekly Sector Status — Finance") |
| body | Rich Text (Markdown) | Narrative assessment |
| period_covered_start | Timestamp | Start of the period this report covers |
| period_covered_end | Timestamp | End of the period this report covers |
| as_of | Timestamp | When the assessment was made (may differ from publication time — a report written Monday morning about Sunday's state) |
| published_at | Timestamp | When the report was published on the platform |
| status | Enum | The assessed operational state: NORMAL, DEGRADED, IMPAIRED, CRITICAL, UNKNOWN |
| impact | Enum | CRITICAL (black), HIGH (red), MEDIUM (amber), LOW (yellow), INFO (green) |
| tlp | Enum | Sharing restriction: RED, AMBER_STRICT, AMBER, GREEN, CLEAR |
| author | User ref | Who wrote the report |
| organization | Org ref | Owning organization (even for vertical/sector reports, someone owns authorship) |
| referenced_events | List[Event ref] | Events discussed or informing this report |
| revision_history | List[Revision] | Full edit history, immutable |

**Who can create Status Reports:**
- Org Root / Org Admin: Reports scoped to their org.
- Vertical Root: Reports scoped to their vertical.
- Sector Root: Reports scoped to their sector.
- Content Admin: Reports at any scope.
- Cedar policies govern all access.

**Status Report vs. Event distinction:**
- A Status Report is an *assessment*: "Here is the state of Org X as of Monday 09:00."
- An Event is an *occurrence*: "This specific incident happened and here is its lifecycle."
- Status Reports reference events. Events do not reference status reports.

### 3.3 Events

An Event represents a specific operational incident, disruption, or situation.

| Field | Type | Description |
|---|---|---|
| id | UUID | Globally unique |
| source_instance | String | Instance ID. Default: local. |
| sector_context | Sector ref | Sector this event belongs to. Immutable after creation. |
| title | String | Short summary, required |
| description | Rich Text (Markdown) | Detailed body, required |
| event_type | Enum | Classification — see Section 3.3.1 |
| submitter | User ref | Creating user |
| organization | Org ref | Owning organization |
| tlp | Enum | RED, AMBER_STRICT, AMBER, GREEN, CLEAR |
| impact | Enum | CRITICAL (black), HIGH (red), MEDIUM (amber), LOW (yellow), INFO (green) |
| status | Enum | OPEN → INVESTIGATING → MITIGATING → RESOLVED → CLOSED |
| created_at | Timestamp | Immutable |
| updated_at | Timestamp | Last modification |
| campaigns | List[Campaign ref] | Zero or more (many-to-many) |
| correlations | List[Correlation ref] | Zero or more |
| attachments | List[Attachment] | See Section 3.8 |
| revision_history | List[Revision] | Full edit history, immutable |

Edits create a new revision. Full history retained, only platform root can purge.

Who can edit: submitter, org admins, org root, vertical admins (within scope), content admins. Governed by Cedar.

#### 3.3.1 Event Type Classification

Basic taxonomy for categorizing events. Supports search, filtering, and AI correlation quality.

| Category | Types |
|---|---|
| Security Incident | Ransomware, Data Breach, Unauthorized Access, DDoS / Availability Attack, Phishing / Social Engineering, Supply Chain Compromise, Insider Threat, Malware / Wiper, Account Compromise, Vulnerability Exploitation |
| Service Disruption | Outage — Planned Maintenance, Outage — Unplanned, Degraded Performance, Capacity Issue, Connectivity Loss |
| Operational | Policy Change, Configuration Error, Third-Party Dependency Failure, Regulatory / Compliance Event |
| Environmental | Natural Disaster Impact, Power Disruption, Facility Issue |
| Advisory | Threat Advisory, Vulnerability Advisory, Situational Awareness, General Notice |

- Events require exactly one type at creation.
- The type list is configurable by platform admins (new types can be added; built-in types cannot be removed, only hidden).
- Types are used as filter/facet dimensions and as features for AI correlation.

### 3.4 Event Updates

Chronological progress entries attached to a specific event. These are the granular "what's happening now" log.

| Field | Type | Description |
|---|---|---|
| id | UUID | System-generated |
| event | Event ref | Parent event |
| author | User ref | Posting user |
| body | Rich Text (Markdown) | Update content |
| tlp | Enum | Can be more restrictive than parent event, never less |
| impact_change | Enum (nullable) | If present, updates the event's current impact |
| status_change | Enum (nullable) | If present, updates the event's current status |
| timestamp | Timestamp | When posted |

### 3.5 Campaigns

Campaigns group related events across time, organizations, and sectors. They represent the horizontal/cross-cutting dimension.

| Field | Type | Description |
|---|---|---|
| id | UUID | Globally unique |
| title | String | Campaign name |
| description | Rich Text (Markdown) | Context and summary |
| tlp | Enum | Independent sharing restriction |
| status | Enum | ACTIVE / CLOSED |
| events | List[Event ref] | Many-to-many |
| created_by | User ref | Creator |
| organization | Org ref | Owning organization |

- An event can belong to multiple campaigns.
- Campaign visibility is independent from member event visibility.
- When events are hidden by access policies: "N organizations have restricted content" (count of orgs, not events).
- Campaigns may span multiple sectors. The dashboard displays campaigns as a separate horizontal structure alongside the sector hierarchy.

### 3.6 Correlations

Links between events indicating a relationship.

| Field | Type | Description |
|---|---|---|
| id | UUID | System-generated |
| event_a | Event ref | First event |
| event_b | Event ref | Second event |
| label | String | Free-text relationship description |
| type | Enum | MANUAL, SUGGESTED, CONFIRMED |
| created_by | User ref or Agent ref | Who/what created the link |
| created_at | Timestamp | When |

- Bidirectional.
- Suggested correlations (from AI) require analyst confirmation before becoming visible to others.
- Visibility: user sees only correlations where they have access to both linked events.

### 3.7 Event Relationships

Labeled, directional relationships between events supporting the deferred sanitization workflow and general cross-referencing.

| Field | Type | Description |
|---|---|---|
| source_event | Event ref | Origin event |
| target_event | Event ref | Derived/related event |
| label | String | Relationship type (e.g., "sanitized_version", "derived_from", "supersedes", custom) |
| created_by | User ref or Agent ref | Creator |
| visibility | Policy-evaluated | Visible only if user can see both events, except for "sanitized_version" where the source is hidden and only the target is shown |

### 3.8 Attachments

| Constraint | Value |
|---|---|
| Maximum file size | 25 MB per attachment |
| Maximum attachments per event | 10 |
| Allowed file types | Images (PNG, JPEG, GIF, SVG), Documents (PDF, TXT, MD, CSV), Archives (ZIP, TAR.GZ) |
| Blocked file types | Executables (EXE, DLL, SH, BAT, PS1), Office macros (DOCM, XLSM), Scripts (JS, PY, RB) |
| Virus scanning | ClamAV on-prem. All uploads scanned before storage. Quarantine and notify on detection. |
| Storage | Local filesystem with encryption at rest. Path not web-accessible — served through authenticated API endpoint. |

### 3.9 Sharing Level (TLP)

Standard TLP v2.0 as information-sharing restriction.

| Level | Semantics |
|---|---|
| TLP:RED | Named recipients only (specified per-object) |
| TLP:AMBER+STRICT | Owning organization only |
| TLP:AMBER | Owning org + orgs with explicit access grants |
| TLP:GREEN | All authenticated platform users |
| TLP:CLEAR | Unrestricted within the platform |

**TLP vs. org deny policy interaction**: Forbid policies always override permit. If an event is TLP:GREEN but the org has sovereign mode active, external users cannot see it. The UI surfaces this conflict to the submitter: "Your organization's sharing policy restricts external access. External users will not see this event despite its TLP:GREEN marking."

### 3.10 Impact Rating

Separate from TLP. Indicates operational severity.

| Level | Color | Meaning |
|---|---|---|
| Critical | Black | Total service loss, catastrophic operational impact |
| High | Red | Major degradation, significant operational consequences |
| Medium | Amber | Partial impact, workarounds available |
| Low | Yellow | Minor, largely contained |
| Informational | Green | No direct impact, awareness only |

### 3.11 Assessed Status (for Status Reports and Dashboard)

Distinct from Impact (which is per-event severity). Assessed Status is the overall operational state of an organizational unit.

| Status | Meaning |
|---|---|
| NORMAL | Operations within expected parameters |
| DEGRADED | Some services impacted, operations continuing with workarounds |
| IMPAIRED | Significant impact, reduced operational capability |
| CRITICAL | Severe impact, major operational disruption |
| UNKNOWN | Status not assessed or information unavailable |

---

## 4. Visualization Layer

### 4.1 Hierarchical Dashboard (Primary View)

The main interface is a hierarchical dashboard showing the assessed operational state of the entire platform at a glance.

#### 4.1.1 Structure

```
┌─────────────────────────────────────────────────────────────┐
│  GLOBAL STATUS: [computed]                                  │
│                                                             │
│  ┌─── Sector: Government [computed] ──────────────────┐     │
│  │  ├── Vertical: Federal [computed]                  │     │
│  │  │     ├── Org A: NORMAL                           │     │
│  │  │     ├── Org B: DEGRADED                         │     │
│  │  │     └── Org C: CRITICAL                         │     │
│  │  └── Vertical: State [computed]                    │     │
│  │        └── Org D: IMPAIRED                         │     │
│  └────────────────────────────────────────────────────┘     │
│                                                             │
│  ┌─── Sector: Finance [computed] ─────────────────────┐     │
│  │  ...                                               │     │
│  └────────────────────────────────────────────────────┘     │
│                                                             │
│  ═══ Campaigns ═══════════════════════════════════════════   │
│  ├── Campaign: "Q4 Infrastructure Upgrade" (ACTIVE)         │
│  ├── Campaign: "Regional Connectivity Event" (ACTIVE)       │
│  └── Campaign: "March Incident Series" (CLOSED)             │
└─────────────────────────────────────────────────────────────┘
```

- Each node shows its assessed status (color-coded).
- **Computed statuses** for verticals, sectors, and global are derived from child statuses using a configurable formula (see Section 4.1.3).
- The hierarchy is collapsible/expandable.
- Campaigns appear as a separate horizontal section below (or beside) the hierarchy, since they span sectors.

#### 4.1.2 Side Panel (Timeline View)

Selecting any node (org, vertical, sector, or campaign) opens a side panel showing a **chronological timeline** combining:

- Status Reports scoped to that node.
- Events owned by or scoped to that node (subject to the viewer's access permissions).

Timeline entries are interleaved chronologically and visually distinguished by type (status report vs. event). Each entry shows: title, timestamp, impact/status badge, TLP badge, and a brief excerpt.

Clicking a timeline entry navigates to a **dedicated detail page** showing full content, revision history, linked events/campaigns/correlations, and (for events) the event update log.

#### 4.1.3 Status Aggregation Formulas (Starlark)

Computed statuses (vertical, sector, global) are derived from child node statuses using formulas.

**Default formula**: Weighted average, mapping statuses to numeric values (NORMAL=0, DEGRADED=1, IMPAIRED=2, CRITICAL=3, UNKNOWN=null/excluded) and applying thresholds.

**Custom formulas** are written in **Starlark** (the deterministic, sandboxed subset of Python used by Bazel). This provides:

- Safe execution: no I/O, no imports, no infinite loops (Starlark guarantees termination).
- Familiar syntax for anyone who knows Python.
- Deterministic results — same inputs always produce the same output.

**Formula interface:**

```python
def compute_status(children):
    """
    children: list of dicts, each with:
        - "id": string (child node ID)
        - "status": string ("NORMAL", "DEGRADED", "IMPAIRED", "CRITICAL", "UNKNOWN")
        - "weight": float (default 1.0, configurable per child)
        - "last_updated": string (ISO timestamp of last status report)
    
    Returns: string — one of "NORMAL", "DEGRADED", "IMPAIRED", "CRITICAL", "UNKNOWN"
    """
    pass
```

- Formulas are set per-node (a sector can have a different formula than another sector).
- Only root users for that node (or parent roots) can modify the formula.
- Formula changes are audited.
- A default formula is always available as fallback.

**PoC scope**: The default formula is implemented. The Starlark execution engine is integrated. Custom formula editing UI is minimal (text editor with validation). The formula library (pre-built alternatives) is deferred.

#### 4.1.4 Campaign View

Clicking a campaign in the horizontal section opens a view showing:
- Campaign metadata (title, description, TLP, status).
- All linked events (subject to access control), grouped by organization and sector.
- A mini-timeline of event status changes within the campaign.
- The "N organizations have restricted content" indicator if applicable.

### 4.2 Event Detail Page

Full detail view for a single event:
- All fields from the event data model.
- Event update log (chronological).
- Correlations (linked events, with access-controlled visibility).
- Campaign memberships.
- Event relationships (sanitized versions, derived events).
- Revision history (expandable diff view).

### 4.3 Status Report Detail Page

Full detail view for a status report:
- All fields from the status report data model.
- Referenced events (clickable, access-controlled).
- Revision history.

### 4.4 Correlation Graph (Phase 2)

Visual graph representation of event correlations and relationships. Deferred but the data model fully supports it.

---

## 5. Identity & Access Management

### 5.1 Authentication

| Requirement | Detail |
|---|---|
| Primary | SSO via SAML 2.0 and/or OIDC with bundled Keycloak |
| Fallback | Local username/password with mandatory TOTP MFA |
| MFA | Required for all users. TOTP (RFC 6238) baseline. WebAuthn/FIDO2 deferred. |
| Brute-force | Account lockout after 5 failed attempts (configurable), exponential backoff, CAPTCHA after 3 failures |
| Sessions | Configurable timeout (default 8 hours), concurrent session limit (default 3), admin-forced logout |

Keycloak is bundled and is the default IdP. Federated instances (Phase 2) use their own Keycloak with trust established via OIDC federation or SAML metadata exchange.

### 5.2 Authorization (Cedar)

#### 5.2.1 Principal Types

| Type | Description |
|---|---|
| User | Human principal with org membership(s) and role assignments |
| Root | Special user designation with maximum authority within scope |
| SystemAgent | Non-human principal for AI agents, distinct trust level |
| FederatedAgent | SystemAgent from a remote instance (Phase 2) |

**Critical design constraint**: `user.organization_memberships` and `user.administrative_scope` are separate, non-conflatable attributes. Data-plane policies check membership. Control-plane policies check administrative scope. Root users have both, but this dual-access is expressed through explicit policy, not conflation of the attributes.

#### 5.2.2 Resource Types

Status Reports, Events, Event Updates, Campaigns, Correlations, Organizations, Verticals, Sectors, IAM Functions, Audit Logs, Agent Configurations, Formulas.

#### 5.2.3 Actions

`view`, `create`, `edit`, `delete`, `link`, `manage_members`, `manage_policies`, `export`, `break_glass`, `configure_agent`, `configure_formula`.

#### 5.2.4 Role Hierarchy

| Role | Scope | Data Plane | Control Plane |
|---|---|---|---|
| Platform Root | Global | Full | Full |
| Sector Root | Sector | Full within sector | Full within sector |
| Vertical Root | Vertical | Full within vertical | Full within vertical |
| Org Root | Organization | Full within org | Full within org |
| Content Admin | Global | Edit/moderate any event/report | None |
| Org Admin | Organization | Full within org | User management within org |
| Contributor | Organization | Create/edit own events, add correlations | None |
| Viewer | Variable | Read-only per policy | None |
| Liaison | Cross-org | Read access to assigned orgs' shared content | None |

#### 5.2.5 Grant Semantics

1. Default deny.
2. Permit policies grant access (role-based, scoped to organizational unit).
3. Forbid policies deny unconditionally (always override permit).
4. TLP levels enforced as Cedar conditions on permits.
5. Root roles inherit permits for entire constituency.

#### 5.2.6 Control Plane Event Notifications

Sector roots receive email notifications for high-importance control-plane events within their sector:
- Permission updates on events or entire orgs.
- Root reassignments.
- Policy changes by org roots.

These notifications are configurable — sector roots can snooze specific notification categories. The notification is about the *action occurring*, not about any resulting policy conflict.

#### 5.2.7 Policy Management (PoC)

- Fixed policy templates, parameterized by org/role/scope.
- No free-form Cedar editing.
- All policy changes audited.

### 5.3 User-to-Organization Membership

- A user belongs to one primary organization.
- A user can be assigned to additional organizations via explicit admin action.
- Multi-org users have permissions evaluated per-org context.
- UI presents an "active context" selector for multi-org users.

---

## 6. AI & Correlation Module

### 6.1 Manual Correlation (PoC Baseline)

- Users with `link` permission create/remove correlations between accessible events.
- Free-text label for relationship description.
- Correlation graph navigable from event detail.
- Keyword/tag search assists discovery.

### 6.2 Agent Architecture

Two agent types, both asynchronous hooks. Fresnel functions fully without them.

#### 6.2.1 Agent 1 — Privileged Analyst (Phase 2 — Architect For)

**Purpose**: Cross-organizational event correlation.

**Access model**: System-level read access to all events on the local instance. Cannot create or modify events. Outputs only — suggested correlations and alerts.

**Output filtering**: Before any output reaches a user, the authorization layer evaluates whether the recipient has access to the informing events. Outputs are filtered or redacted per-recipient. If filtering removes all meaningful content, the output is silently dropped.

**Sovereign mode interaction**:
- Orgs in sovereign mode can opt out of Agent 1 processing entirely (events excluded from corpus).
- Alternatively, orgs can opt in to Agent 1 processing while in sovereign mode — the agent reads their events for correlation, but outputs are filtered by authorization before delivery.
- **Default for sovereign orgs: opt-out.** Opt-in requires explicit action by the org root.

**Isolation**: Agent 1 runs as an isolated service with no interactive/ad-hoc query capability. It processes on a schedule or on event triggers. No user can use it as an oracle to probe across access boundaries.

**Infrastructure**: Ollama with configurable model (HuggingFace pull — model selection is a configuration option). CPU-only inference as baseline (slower, background batch processing). GPU acceleration when available (faster, near-real-time). Architecturally isolated — removable without affecting core platform.

**Embedding storage**: pgvector extension within PostgreSQL. Embeddings generated on event creation/update.

#### 6.2.2 Agent 2 — User-Scoped Assistant (Phase 2)

Delegated user credentials. Same permissions and rate limits as the user. Assists with drafting, searching, summarizing within the user's scope.

### 6.3 Federation Agent Interaction (Phase 2)

Hub Agent 1 cannot directly access federated org data. Instead:

1. Hub Agent 1 detects a correlation opportunity in hub-local data.
2. Sends a **correlation query** to the federated instance (pattern description only — no source event content from other orgs).
3. Federated instance's local Agent 1 evaluates against local data.
4. Response scoped by org's sharing policies, queued for org admin approval (unless matching events are already shareable).
5. Hub integrates the response.

**Privacy constraint**: Query descriptions must use abstract pattern language, not reproduce or closely paraphrase source event content. The query template is constrained to prevent indirect information leakage. This is a recognized hard problem — the query-template design requires dedicated attention during Phase 2 design.

---

## 7. Nudge & Escalation System

### 7.1 Core Purpose

Ensure situational information stays current. Nudges are not pressure to resolve — they're requests to update.

### 7.2 Nudge Rules

| Event Impact | Nudge Frequency | Trigger |
|---|---|---|
| CRITICAL or HIGH | Daily | No update today, sent at end of business day (user's timezone) |
| MEDIUM | Daily | No update today, sent at end of business day |
| LOW | Weekly | No update in 7 days |
| INFO | None | No nudge |

All open events with impact > INFO also receive a **weekly digest nudge** regardless of last update time.

### 7.3 Escalation Chain

```
Event contributors → Org Root → Vertical Root → Sector Root → Platform Root
```

- Each level gets 1 business day to respond (any status update on the event from that scope counts).
- If the chain reaches platform root with no response, nudging continues weekly as a combined email. No further escalation — if this happens, the platform has been abandoned.

### 7.4 Constraints

- Email only for PoC.
- No snooze, no acknowledgment. The expected response is a status update.
- End of business day: per user timezone (from profile). Fallback: org timezone.
- Requires on-prem SMTP relay.

---

## 8. API Design

### 8.1 Transport & Content Negotiation

Single set of RESTful endpoints, dual representations:

| Accept Header | Response Format | Consumer |
|---|---|---|
| `text/html` | HTML fragments (HTMX) | Browser UI |
| `application/json` | JSON | Automation, integrations, agents |

Full automation potential from day one. The UI is not a separate app — it's a consumer of the same API.

### 8.2 Design Principles

- All state-changing operations via standard HTTP methods.
- All endpoints authenticated (Bearer token from Keycloak).
- All endpoints authorized via Cedar policy evaluation.
- CSRF protection via token headers on HTML responses.
- Pagination, filtering, sorting consistent across all collections.

### 8.3 Core Endpoints (PoC)

```
/api/v1/status-reports              GET, POST
/api/v1/status-reports/{id}         GET, PUT, DELETE

/api/v1/events                      GET, POST
/api/v1/events/{id}                 GET, PUT, DELETE
/api/v1/events/{id}/updates         GET, POST
/api/v1/events/{id}/correlations    GET, POST, DELETE
/api/v1/events/{id}/relationships   GET, POST, DELETE

/api/v1/campaigns                   GET, POST
/api/v1/campaigns/{id}              GET, PUT, DELETE
/api/v1/campaigns/{id}/events       GET, POST, DELETE

/api/v1/orgs                        GET
/api/v1/orgs/{id}                   GET
/api/v1/orgs/{id}/members           GET, POST, DELETE
/api/v1/orgs/{id}/policies          GET, POST, PUT, DELETE

/api/v1/sectors                     GET
/api/v1/sectors/{id}                GET
/api/v1/sectors/{id}/formula        GET, PUT

/api/v1/verticals                   GET
/api/v1/verticals/{id}              GET

/api/v1/users                       GET, POST
/api/v1/users/{id}                  GET, PUT
/api/v1/users/me                    GET

/api/v1/audit                       GET (filtered by scope, authorized by role)
/api/v1/dashboard                   GET (hierarchical status tree)
/api/v1/dashboard/{node_type}/{id}/timeline  GET (timeline for a node)

/api/v1/health                      GET (unauthenticated)
```

Federation stubs (return 501 in PoC):
```
/api/v1/federation/events
/api/v1/federation/queries
/api/v1/federation/heartbeat
```

Webhook stubs (return 501 in PoC):
```
/api/v1/webhooks
```

---

## 9. Security Requirements

### 9.1 Software-Layer Security

| Requirement | Detail |
|---|---|
| Encryption at rest | Database-level encryption (LUKS on storage or PostgreSQL TDE) |
| Encryption in transit | TLS 1.3 mandatory on all connections |
| Input validation | All user input sanitized. Markdown rendered safely (no XSS through Markdown injection). |
| CSRF | Token-based on all state-changing HTML operations |
| Rate limiting | Per-user and per-IP, configurable, enforced at reverse proxy |
| Dependency security | Minimal footprint. No known critical CVEs at deployment. |
| Secrets management | No hardcoded secrets. Encrypted config or on-prem vault. |
| Audit logging | All auth events, authorization decisions, and data mutations. Append-only, immutable. |

### 9.2 Security Headers

All HTTP responses must include:

| Header | Value | Purpose |
|---|---|---|
| Content-Security-Policy | `default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; frame-ancestors 'none'; form-action 'self'; base-uri 'self'` | XSS mitigation, framing prevention. Inline styles allowed for HTMX/Markdown rendering. Tighten further based on actual requirements during implementation. |
| Strict-Transport-Security | `max-age=63072000; includeSubDomains; preload` | Force HTTPS. 2-year max-age. |
| X-Content-Type-Options | `nosniff` | Prevent MIME sniffing |
| X-Frame-Options | `DENY` | Legacy framing prevention (supplement to CSP frame-ancestors) |
| Referrer-Policy | `strict-origin-when-cross-origin` | Limit referrer leakage |
| Permissions-Policy | `camera=(), microphone=(), geolocation=(), payment=()` | Disable unused browser APIs |
| X-XSS-Protection | `0` | Disabled — CSP is the proper defense. Legacy XSS filters can introduce vulnerabilities. |
| Cache-Control | `no-store` on authenticated responses | Prevent caching of sensitive data |

**Certificate pinning**: Not implemented as an HTTP header — HPKP (HTTP Public Key Pinning) was deprecated by browsers due to the risk of self-denial-of-service (pin the wrong cert and you're locked out). For production, the equivalent protection is achieved through:
- Certificate Transparency (CT) log monitoring — detect unauthorized cert issuance.
- DANE/TLSA records if DNS infrastructure supports it.
- For internal/federated connections: mTLS with explicit trust stores (effectively pinning at the application layer).

The reverse proxy configuration should enforce these headers globally so application code cannot accidentally omit them.

### 9.3 Infrastructure-Layer Security

| Requirement | PoC Implementation | Future |
|---|---|---|
| Reverse proxy | nginx with rate limiting, TLS termination, security headers | HAProxy with advanced traffic shaping |
| DDoS mitigation | nginx connection limits, fail2ban | Programmatic firewall integration (nftables) — see Section 9.4 |
| WAF | ModSecurity with OWASP CRS on nginx | Dedicated WAF or tuned ruleset |
| Virus scanning | ClamAV for attachment uploads | Same, with signature update automation |
| Network segmentation | Single VM, process-level isolation | Separate VMs for app, DB, Keycloak, agents |

### 9.4 Firewall Integration Aspiration (Phase 2 — Significant Caveats)

The platform may programmatically control firewall rules (blackholing malicious IPs before traffic reaches the app). Design constraints:

- One-way channel only: app can tell the firewall "block this IP." Cannot query, modify, or delete existing rules.
- No whitelist modification, no rule deletion without platform root MFA.
- If the app is compromised, the attacker can block legitimate users but cannot whitelist their own traffic.
- Interface contract to be documented but not implemented for PoC.

---

## 10. Non-Functional Requirements

| Requirement | PoC Target | Architecture Target |
|---|---|---|
| Deployment | Single vSphere VM, Docker Compose | Multi-VM, container orchestration |
| Availability | Single node, no HA | Active-passive failover minimum |
| Performance | < 500ms page load (10k events), < 200ms auth decisions, < 100ms dashboard render | Same under 10x load |
| Scalability | ~100 users, ~10k events | ~1,000 users, ~100k events |
| Backup | Daily DB backup, RPO < 24h | Continuous replication, RPO < 1h |
| Observability | Structured logging, health endpoint | Prometheus metrics, Grafana dashboards |
| Audit retention | Immutable, no archival | Review archival policy when storage becomes a concern |
| LLM | CPU baseline: batch/background (minutes). GPU: near-real-time (seconds). Model selection via config (HuggingFace pull). | Same with queue management |

---

## 11. PoC Scope Definition

### 11.1 Implemented in PoC

| Feature | Scope |
|---|---|
| Status Reports | Full CRUD, scope (org/vertical/sector), period/as-of timestamps, assessed status |
| Events | Full CRUD, revision history, TLP, impact, status lifecycle, event type classification |
| Event Updates | Create, view, TLP/impact/status changes |
| Campaigns | Full CRUD, event linking, cross-sector display |
| Manual correlations | Create, view, delete |
| Event relationships | Create, view (supports future sanitization workflow) |
| Hierarchical dashboard | Global → sector → vertical → org tree with computed statuses, side panel timeline, campaign section |
| Status aggregation | Default formula + Starlark engine for custom formulas |
| IAM | Keycloak integration, TOTP MFA, Cedar policy evaluation with templates |
| Root user model | Full hierarchy, root self-assignment and parent assignment |
| API | Full REST + HTMX, content negotiation, all core endpoints |
| Audit logging | All mutations and auth events, append-only, immutable |
| Nudge system | Tiered email nudges (daily for MEDIUM+, weekly for LOW), escalation chain, weekly digest |
| Security | TLS 1.3, CSP + full security headers, CSRF, rate limiting, brute-force protection, input validation, ClamAV |
| Multi-org membership | Assignment operation, context switching |
| Attachments | Upload with type/size restrictions, ClamAV scanning |
| Deployment | Docker Compose on single vSphere VM |
| Rich text | Markdown with user-friendly WYSIWYG editor (Markdown source hidden by default) |

### 11.2 Phase 1 (Short Distance from PoC)

| Feature | Provision |
|---|---|
| Break-glass procedure | Audit event types defined, API endpoint stubbed, notification template exists. Implement procedure, cooling-off logic, and time-limited access. |
| Root succession procedures | Detailed governance rules for contested/emergency reassignment. |
| Correlation graph visualization | Data model supports it; implement UI. |

### 11.3 Phase 2 (Deferred — Architected For)

| Feature | Architectural Provision |
|---|---|
| Sovereign mode | Cedar schema supports org-level forbid. UI conflict surfacing designed. Data model unchanged. |
| Federation | UUIDs globally unique, source_instance field present, API stubs return 501, federation endpoints reserved. |
| Agent 1 (privileged analyst) | SystemAgent principal in Cedar, pgvector installed, Ollama in compose file, correlation pipeline not built. |
| Agent 2 (user-scoped assistant) | Delegated auth flow in Keycloak designed, API contract defined. |
| AI-assisted sanitization | Event relationship model supports "sanitized_version" label, agent hook interface defined. |
| Firewall integration | Interface contract documented, not connected. |
| Advanced notifications | Webhook endpoint reserved, notification preference model in user schema. |
| Formula library | Pre-built Starlark formula alternatives for different aggregation strategies. |

### 11.4 Out of Scope (Not Architected For)

| Item | Rationale |
|---|---|
| STIX/TAXII | Not a TIP. External adapter if ever needed. |
| Automated feed ingestion | Events are human-created or AI-assisted. |
| Playbook/response workflow | Tracks status, does not prescribe response. |
| Mobile app | HTMX UI is responsive, no native target. |

---

## 12. Issues & Risks

### 12.1 Identified Risks

**Risk 1 — PoC Scope Discipline.** Even with Phase 1/2 deferrals, the PoC includes: status reports, events, campaigns, correlations, hierarchical dashboard with Starlark formulas, full IAM with Cedar, Keycloak, HTMX+JSON API, nudge system with escalation, ClamAV, and comprehensive security hardening. This is substantial. The minimum viable PoC, if schedule pressure hits: events + IAM + API + dashboard. Status reports, campaigns, nudging, and Starlark are additive layers.

**Risk 2 — Cedar Policy Complexity.** The policy model supports root hierarchy, org-level controls, TLP enforcement, multi-org membership, sector-scoped events, and (deferred) sovereign mode. Testing all interaction paths is non-trivial. Build a policy test suite as part of the PoC.

**Risk 3 — Keycloak Operational Burden.** Bundling Keycloak means owning its deployment and configuration. Minimal config for PoC (single realm, basic theme) is manageable; federation makes it significantly more complex.

**Risk 4 — Sector Context Immutability.** Events are sector-bound with immutable sector context. Wrong selection means creating a new event and closing the original. UI must make sector selection prominent with confirmation for multi-sector org users.

**Risk 5 — Agent 1 Query Privacy (Phase 2).** Correlation queries from hub to federated instances may leak information through pattern descriptions. Query template design requires dedicated attention.

**Risk 6 — Starlark Formula Security.** Although Starlark is sandboxed and guarantees termination, formulas run with access to child node status data. A malicious formula can't escape the sandbox, but a poorly written one could produce misleading dashboard status. Validation, testing, and a "preview before activate" workflow mitigate this.

**Risk 7 — Markdown XSS Surface.** Markdown rendered as HTML creates an XSS vector if not properly sanitized. The rendering library must strip all HTML tags, scripts, and event handlers. CSP provides defense-in-depth but the sanitizer is the primary defense.

---

*This specification is ready for final review. No open questions block the architecture phase. The next work item is the architecture document.*