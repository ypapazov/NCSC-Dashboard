# Fresnel — Architecture Document

**Version**: 0.1
**Status**: Draft — accompanies Requirements Specification v0.1
**Prerequisite**: Read the Requirements Specification first. This document does not restate domain definitions.

---

## 1. Architecture Principles

1. **Security enforcement is layered.** Coarse-grained authorization at the HTTP boundary. Fine-grained filtering at the storage boundary. Defense in depth, not a single chokepoint.
2. **Storage is abstracted.** All data access goes through Go interfaces. Implementations are swappable. The PoC uses PostgreSQL for everything; production may split databases by concern.
3. **The API is the product.** The UI is a consumer of the API, not a separate system. There is one set of endpoints, two representations (HTML, JSON). No backend-for-frontend pattern.
4. **Deferred features are architectural boundaries, not TODOs.** Federation, sovereign mode, and agents are not implemented, but the boundaries where they attach are explicit interfaces in the codebase.
5. **Pinned, minimal dependencies.** Go modules with pinned versions. Every dependency justified. No transitive dependency sprawl.

---

## 2. High-Level Component Architecture

```
                                    ┌─────────────────────┐
                                    │     Internet         │
                                    └─────────┬───────────┘
                                              │
                                    ┌─────────▼───────────┐
                                    │   nginx              │
                                    │   - TLS termination  │
                                    │   - Security headers │
                                    │   - Rate limiting    │
                                    │   - fail2ban hook    │
                                    │   - ModSecurity/CRS  │
                                    └─────────┬───────────┘
                                              │ (plaintext HTTP
                                              │  on internal network)
                                    ┌─────────▼───────────┐
                                    │   Fresnel API Server │
                                    │   (Go binary)        │
                                    │                      │
                                    │  ┌─── HTTP Layer ──┐ │
                                    │  │ Routing          │ │
                                    │  │ Auth Middleware   │ │◄─── OIDC token
                                    │  │ Cedar Gate       │ │     validation
                                    │  │ Content Neg.     │ │     (Keycloak)
                                    │  └───────┬──────────┘ │
                                    │          │            │
                                    │  ┌───────▼──────────┐ │
                                    │  │ Service Layer     │ │
                                    │  │ (Business Logic)  │ │
                                    │  │ Cedar Row Filter  │ │
                                    │  │ Starlark Engine   │ │
                                    │  └───────┬──────────┘ │
                                    │          │            │
                                    │  ┌───────▼──────────┐ │
                                    │  │ Storage Layer     │ │
                                    │  │ (Interfaces)      │ │
                                    │  │ Auth-unaware      │ │
                                    │  └───────┬──────────┘ │
                                    └──────────┼────────────┘
                              ┌────────────────┼────────────────┐
                              │                │                │
                    ┌─────────▼──┐   ┌────────▼───────┐  ┌────▼──────┐
                    │ PostgreSQL  │   │   Keycloak      │  │  ClamAV   │
                    │ + pgvector  │   │   (OIDC/SAML)   │  │  (scan)   │
                    └────────────┘   └────────────────┘  └───────────┘

                    ┌─────────────┐
                    │ Ollama      │   (Phase 2 — container present
                    │ (LLM)      │    but no pipeline connected)
                    └─────────────┘
```

### 2.1 Component Responsibilities

| Component | Responsibility | PoC Implementation |
|---|---|---|
| nginx | TLS termination, security headers (CSP etc.), rate limiting, WAF (ModSecurity + OWASP CRS), reverse proxy, fail2ban integration | Single nginx container with static config |
| Fresnel API Server | All application logic: routing, authentication, authorization, business rules, content negotiation, template rendering, Starlark execution | Single Go binary |
| PostgreSQL | Primary data store for all domain objects, audit log, Cedar policies | Single PostgreSQL 16 instance with pgvector extension |
| Keycloak | Identity provider: OIDC/SAML (Authorization Code + PKCE, public client), user store, TOTP MFA, brute-force protection, SSO session management, token issuance | Single Keycloak container, single realm |
| ClamAV | Virus scanning for file uploads | clamd daemon, API server connects via socket |
| Ollama | LLM inference for AI agents (Phase 2) | Container present in compose file, no application integration |

---

## 3. Layered Architecture (API Server Internals)

The API server has three layers with distinct responsibilities and a strict dependency direction: HTTP → Service → Storage. No layer may bypass the one below it.

### 3.1 HTTP Layer

Responsibilities:
- Route registration and request dispatch.
- Authentication: validate OIDC Bearer token (JWT) from the `Authorization` header, extract user identity and claims, build AuthContext from DB.
- **Coarse-grained authorization (Cedar Gate)**: evaluate "Can this principal perform this action on this resource type?" This is a binary allow/deny check *before* any business logic executes.
- Content negotiation: inspect `Accept` header, select renderer (HTML template or JSON serializer).
- Request/response logging.
- Input deserialization and basic validation (field types, required fields, size limits).

**The Cedar Gate at this layer answers questions like:**
- Can User X create events? (yes/no)
- Can User X access the audit log endpoint? (yes/no)
- Can User X modify policies for Org Y? (yes/no)

It does **not** answer: "Which specific events can User X see?" That requires knowledge of the data, which is the storage layer's responsibility.

```go
// Middleware chain (conceptual)
router.Use(RequestLogger)
router.Use(OIDCTokenValidator)    // validate Bearer JWT (JWKS) → AuthContext from DB
router.Use(CedarGate)             // AuthContext + route → permit/deny
router.Use(ContentNegotiator)     // sets renderer on context
```

**AuthContext**: The token validator produces an AuthContext that flows through the entire request lifecycle:

```go
type AuthContext struct {
    UserID              uuid.UUID
    PrimaryOrgID        uuid.UUID
    OrgMemberships      []uuid.UUID
    ActiveOrgContext     uuid.UUID       // selected by user for multi-org
    AdministrativeScope []ScopeEntry     // org/sector admin scopes
    IsRoot              bool
    RootScope           *ScopeEntry      // nil if not root
    Roles               []RoleAssignment
}
```

This struct is **derived from the JWT `sub` claim and a DB lookup on every request**. The token provides identity; the database provides roles, memberships, and root designations. Because access tokens are short-lived (10 min), changes to a user's roles or memberships propagate within one token lifetime. The `ActiveOrgContext` is read from an `X-Fresnel-Org` request header set by the UI's org context selector.

### 3.2 Service Layer

Responsibilities:
- Business logic: event lifecycle, status report creation, campaign management, correlation rules.
- Validation: business rules (TLP cannot be less restrictive on child than parent, status transitions, sector context matching).
- **Row-level authorization**: Evaluates every data object returned from the storage layer against Cedar before returning it to the HTTP layer. This is the single source of truth for "can this user see this specific record?" — Cedar decides, not SQL.
- Starlark formula execution for dashboard status computation.
- Nudge/escalation scheduling logic.
- Coordination between storage interfaces (e.g., creating an event and its initial audit entry atomically).
- Invoking ClamAV for attachment scanning.

The service layer receives an `AuthContext` from the HTTP layer and uses it to evaluate Cedar policies against each data object. The storage layer is authorization-unaware — it returns candidate data based on functional filters (sector, date range, status, etc.), and the service layer filters the results through Cedar.

This means Cedar is the **sole authorization engine**. There are no parallel SQL-based access control checks that could diverge from policy. The trade-off is performance: list operations may fetch more rows from the database than the user is ultimately permitted to see. At PoC scale (~10k events, ~100 users) this is acceptable. If it becomes a bottleneck, the storage layer can add Cedar-informed query hints as a performance optimization — but correctness always flows from Cedar evaluation, not from SQL predicates.

```go
// Service layer interface example
type EventService interface {
    Create(ctx context.Context, auth AuthContext, input CreateEventInput) (*Event, error)
    GetByID(ctx context.Context, auth AuthContext, id uuid.UUID) (*Event, error)
    List(ctx context.Context, auth AuthContext, filters EventFilters) (*EventPage, error)
    Update(ctx context.Context, auth AuthContext, id uuid.UUID, input UpdateEventInput) (*Event, error)
    Delete(ctx context.Context, auth AuthContext, id uuid.UUID) error
}

// Internal to service layer — Cedar evaluation on results
type CedarEvaluator interface {
    // Evaluate whether a principal can perform an action on a specific resource.
    // Returns permit/deny. Called per-row for list operations.
    IsPermitted(auth AuthContext, action string, resource CedarResource) (bool, error)

    // Batch evaluation for list operations — same semantics, amortized overhead.
    FilterPermitted(auth AuthContext, action string, resources []CedarResource) ([]CedarResource, error)
}
```

### 3.3 Storage Layer

Responsibilities:
- Data persistence and retrieval.
- **Functional query filtering**: Filters by domain attributes (sector context, date ranges, status, event type, organization, etc.) but **not** by authorization. The storage layer has no concept of who is asking — it returns all data matching the functional criteria.
- Transaction management.
- Audit log writes (append-only, separate schema).

The storage layer is deliberately authorization-unaware. This keeps it simple, testable, and decoupled from the policy engine. Authorization is the service layer's job.

```go
// Storage interfaces — one per aggregate root
// Note: no AuthContext parameter. Storage is auth-unaware.
type EventStore interface {
    Create(ctx context.Context, tx Tx, event *Event) error
    GetByID(ctx context.Context, id uuid.UUID) (*Event, error)
    List(ctx context.Context, filters EventFilters, page Pagination) ([]*Event, int, error)
    Update(ctx context.Context, tx Tx, event *Event) error
    Delete(ctx context.Context, tx Tx, id uuid.UUID) error
}

type StatusReportStore interface {
    Create(ctx context.Context, tx Tx, report *StatusReport) error
    GetByID(ctx context.Context, id uuid.UUID) (*StatusReport, error)
    List(ctx context.Context, filters ReportFilters, page Pagination) ([]*StatusReport, int, error)
    // ...
}

type AuditStore interface {
    Append(ctx context.Context, entry AuditEntry) error  // no Update, no Delete
    Query(ctx context.Context, filters AuditFilters, page Pagination) ([]*AuditEntry, int, error)
}

type PolicyStore interface {
    ListPolicies(ctx context.Context, scope ScopeEntry) ([]*CedarPolicy, error)
    CreatePolicy(ctx context.Context, tx Tx, policy *CedarPolicy) error
    UpdatePolicy(ctx context.Context, tx Tx, policy *CedarPolicy) error
    DeletePolicy(ctx context.Context, tx Tx, id uuid.UUID) error
}

// Transaction interface
type Tx interface {
    Commit() error
    Rollback() error
}

type TxManager interface {
    Begin(ctx context.Context) (Tx, error)
}
```

**Interface segregation**: Each store interface is independent. In the PoC, they all share one PostgreSQL connection pool. In production, `AuditStore` could point to a separate database, `EventStore` could be backed by a different engine, and federation could introduce a `FederatedEventStore` that aggregates local and remote results.

---

## 4. Authorization Architecture (Cedar as Single Source of Truth)

### 4.1 Tier 1: Cedar Gate (HTTP Middleware)

Evaluated on every request before the handler executes.

**Inputs:**
- Principal: derived from AuthContext (user ID, roles, org memberships, root status).
- Action: derived from HTTP method + route (e.g., `POST /api/v1/events` → `Action::Create` on `ResourceType::Event`).
- Resource: the resource *type* and, when available from the URL, the resource *scope* (e.g., org ID from the path).

**Evaluation**: Calls the Cedar engine (linked as a Go library via `cedar-go`) with the principal, action, and resource. The policy set is loaded from the database and cached in memory, invalidated on policy change events.

**Outcome**: `Permit` or `Deny`. Deny → 403 Forbidden, request stops. Permit → request proceeds to handler.

**What Tier 1 catches:**
- Unauthorized action types (a Viewer trying to POST an event).
- Out-of-scope access (an Org Admin trying to manage a different org's policies).
- Unauthenticated requests (no valid token → 401 before Cedar even runs).

**What Tier 1 cannot catch:**
- Row-level visibility (which specific events can this user see in a list?).
- TLP-based filtering (requires knowing the TLP of each record).
- Dynamic conditions (TLP:RED recipient lists).

These are handled by Tier 2.

### 4.2 Tier 2: Cedar Evaluation in Service Layer (Post-Query Filtering)

For all read operations that return domain objects, the service layer evaluates each result against Cedar **after** the storage layer returns it.

**Flow for list operations:**

```
1. HTTP Layer:    Cedar Gate permits "User X can list events" → proceed
2. Storage Layer: SELECT events WHERE sector_context = ? AND status = ? ... (functional filters only)
3. Service Layer: For each returned event, evaluate Cedar:
                  IsPermitted(auth, "view", event_as_cedar_resource) → permit/deny
                  Return only permitted events to caller
```

**Flow for single-resource operations:**

```
1. HTTP Layer:    Cedar Gate permits "User X can view events" → proceed
2. Storage Layer: SELECT event WHERE id = ?
3. Service Layer: IsPermitted(auth, "view", event_as_cedar_resource)
                  If denied → return 404 (not 403, to avoid leaking existence)
                  If permitted → return event
```

**Why this model:**
- **Single source of truth.** Cedar policies are the only place authorization logic lives. No parallel SQL predicates that could diverge.
- **Testable.** Authorization correctness is verified by testing Cedar policies against a matrix of principal/action/resource combinations. Storage tests are pure data tests.
- **Future-proof.** When sovereign mode, federation, or custom policies are added, only Cedar policies change. No SQL query builders to update.

**Performance trade-off:** List operations may fetch more rows from the database than the user is authorized to see, then discard the unauthorized ones. At PoC scale this is acceptable. Two mitigations if it matters later:

1. **Functional pre-filtering**: The storage layer can apply broad functional filters that are *not* authorization logic but happen to reduce the candidate set (e.g., filtering by sector_context, which the user selected, is a functional filter — not an authorization filter).
2. **Cedar-informed query hints**: As an optimization (not for correctness), the service layer can derive broad hints from the user's known memberships and pass them as functional filters. Correctness still flows from Cedar evaluation, not from the hints.

### 4.3 CedarEvaluator Implementation

```go
type CedarEvaluator interface {
    // Single evaluation
    IsPermitted(auth AuthContext, action string, resource CedarResource) (bool, error)

    // Batch evaluation for list operations
    FilterPermitted(auth AuthContext, action string, resources []CedarResource) ([]CedarResource, error)
}

// CedarResource wraps any domain object for Cedar evaluation
type CedarResource struct {
    Type           string            // "Event", "StatusReport", "Campaign", etc.
    ID             uuid.UUID
    OwnerOrgID     uuid.UUID
    SectorContext  uuid.UUID
    TLP            string
    Attributes     map[string]string // additional Cedar-relevant attributes
}
```

`FilterPermitted` is not N sequential calls — it evaluates the cached policy set against each resource in a tight loop. Cedar evaluation is in-memory and fast (microseconds per decision). For 500 candidate events, this adds ~1-5ms, well within the performance budget.

### 4.4 Cedar Policy Storage and Caching

- Policies stored in PostgreSQL (`cedar_policies` table: id, scope, cedar_text, created_by, created_at, updated_at).
- On startup, the API server loads all policies into an in-memory Cedar policy set.
- On policy change (create/update/delete), an invalidation signal refreshes the in-memory set.
- For single-node PoC, invalidation is a simple in-process event.
- For multi-node (future), invalidation would use PostgreSQL LISTEN/NOTIFY or a lightweight message bus.

---

## 5. Data Architecture

### 5.1 Database Schema (Logical)

Three logical schemas within one PostgreSQL instance:

| Schema | Contents | Access Pattern |
|---|---|---|
| `fresnel` | Events, status reports, campaigns, correlations, relationships, event updates, attachments metadata, TLP:RED recipients, org hierarchy, user-org memberships, platform config, nudge/escalation state | Full CRUD, auth-scoped reads |
| `fresnel_iam` | Cedar policies, role assignments, root designations, access grants | Read-heavy (cached), infrequent writes |
| `fresnel_audit` | Immutable audit entries | Append-only. The application database role has INSERT but not UPDATE or DELETE on this schema. |

The three schemas share one PostgreSQL instance for PoC. The storage interfaces abstract this — splitting to separate databases later requires only new interface implementations, not service layer changes.

### 5.2 Key Tables (Simplified)

```sql
-- fresnel schema

CREATE TABLE sectors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_sector_id UUID REFERENCES sectors(id),  -- NULL for top-level sectors
    name TEXT NOT NULL,
    ancestry_path TEXT NOT NULL DEFAULT '/',         -- materialized path, e.g., '/platform/gov/federal/'
    depth INTEGER NOT NULL DEFAULT 1,               -- nesting level (1 = top-level)
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (depth <= 5)                              -- max nesting depth
);

CREATE INDEX idx_sectors_parent ON sectors(parent_sector_id);
CREATE INDEX idx_sectors_ancestry ON sectors(ancestry_path text_pattern_ops);  -- prefix search

CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sector_id UUID NOT NULL REFERENCES sectors(id), -- direct parent sector
    name TEXT NOT NULL,
    timezone TEXT NOT NULL DEFAULT 'UTC',             -- nudge EOB fallback; defaults to platform_config.default_timezone
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Multi-sector membership (org can appear under multiple sectors at any level)
CREATE TABLE org_sector_memberships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    sector_id UUID NOT NULL REFERENCES sectors(id),
    root_user_id UUID REFERENCES users(id),         -- sector-specific root for this org
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(organization_id, sector_id)
);

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    keycloak_sub TEXT UNIQUE NOT NULL,        -- OIDC subject claim
    display_name TEXT NOT NULL,
    email TEXT NOT NULL,
    primary_org_id UUID NOT NULL REFERENCES organizations(id),
    timezone TEXT NOT NULL DEFAULT 'UTC',
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
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
    scope_type TEXT NOT NULL,                  -- 'ORG', 'SECTOR'
    scope_ref UUID NOT NULL,                   -- references the scoped entity
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    period_covered_start TIMESTAMPTZ NOT NULL,
    period_covered_end TIMESTAMPTZ NOT NULL,
    as_of TIMESTAMPTZ NOT NULL,
    published_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    assessed_status TEXT NOT NULL,             -- NORMAL, DEGRADED, IMPAIRED, CRITICAL, UNKNOWN
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
    CHECK (event_a_id < event_b_id)            -- canonical ordering, prevents duplicates
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
    storage_path TEXT NOT NULL,                -- internal filesystem path
    scan_status TEXT NOT NULL DEFAULT 'pending', -- pending, clean, quarantined
    uploaded_by UUID NOT NULL REFERENCES users(id),
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE tlp_red_recipients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_type TEXT NOT NULL,                -- 'EVENT', 'STATUS_REPORT', 'EVENT_UPDATE'
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
-- Seeded on init: ('default_timezone', 'UTC')

CREATE TABLE nudge_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id),
    recipient_id UUID NOT NULL REFERENCES users(id),
    nudge_type TEXT NOT NULL,                  -- 'DAILY', 'WEEKLY', 'DIGEST', 'ESCALATION'
    escalation_level INTEGER,                  -- NULL for non-escalation nudges
    sent_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_nudge_event_date ON nudge_log(event_id, sent_at DESC);

CREATE TABLE escalation_state (
    event_id UUID PRIMARY KEY REFERENCES events(id),
    current_level INTEGER NOT NULL DEFAULT 0,  -- 0=contributors, 1=org root, 2+=ancestor sector roots
    escalated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_response_at TIMESTAMPTZ
);

-- Formula storage for Starlark status aggregation
CREATE TABLE status_formulas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_type TEXT NOT NULL,                   -- 'SECTOR', 'PLATFORM'
    node_id UUID,                              -- NULL for platform-level formula
    starlark_source TEXT NOT NULL,             -- the Starlark code
    set_by UUID NOT NULL REFERENCES users(id),
    set_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(node_type, node_id)
);
```

```sql
-- fresnel_iam schema

CREATE TABLE cedar_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scope_type TEXT NOT NULL,                  -- 'PLATFORM', 'SECTOR', 'ORG'
    scope_id UUID,                             -- NULL for platform-wide
    policy_template TEXT NOT NULL,             -- template identifier
    cedar_text TEXT NOT NULL,                  -- rendered Cedar policy
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE role_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    role TEXT NOT NULL,
    scope_type TEXT NOT NULL,
    scope_id UUID NOT NULL,
    assigned_by UUID NOT NULL,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(user_id, role, scope_type, scope_id)
);

CREATE TABLE root_designations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    scope_type TEXT NOT NULL,                  -- 'PLATFORM', 'SECTOR', 'ORG'
    scope_id UUID,
    designated_by UUID NOT NULL,               -- self or parent root
    designated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(scope_type, scope_id)               -- one root per scope
);
```

```sql
-- fresnel_audit schema
-- Application DB role: INSERT only. No UPDATE, no DELETE.

CREATE TABLE audit_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT now(),
    actor_id UUID NOT NULL,
    actor_type TEXT NOT NULL,                  -- 'USER', 'SYSTEM_AGENT', 'SYSTEM'
    action TEXT NOT NULL,                      -- 'CREATE', 'UPDATE', 'DELETE', 'LOGIN', 'POLICY_CHANGE', etc.
    resource_type TEXT NOT NULL,
    resource_id UUID,
    scope_type TEXT,
    scope_id UUID,
    detail JSONB NOT NULL DEFAULT '{}',        -- before/after state, additional context
    severity TEXT NOT NULL DEFAULT 'INFO',     -- INFO, WARN, HIGH, CRITICAL
    ip_address INET,
    user_agent TEXT
);

CREATE INDEX idx_audit_timestamp ON audit_entries(timestamp DESC);
CREATE INDEX idx_audit_actor ON audit_entries(actor_id);
CREATE INDEX idx_audit_resource ON audit_entries(resource_type, resource_id);
CREATE INDEX idx_audit_scope ON audit_entries(scope_type, scope_id);
```

### 5.3 pgvector (Phase 2 Readiness)

```sql
-- Installed but not actively populated in PoC
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE event_embeddings (
    event_id UUID PRIMARY KEY REFERENCES fresnel.events(id),
    embedding vector(768),                     -- dimension depends on model choice
    model_version TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_embeddings_ivfflat ON event_embeddings
    USING ivfflat (embedding vector_cosine_ops);
```

---

## 6. Authentication Flow

### 6.1 Model

The Fresnel API server is a **pure resource server**. Keycloak is a black box identity provider — the same integration pattern as Auth0, Okta, or any other OIDC provider. The browser handles the entire OIDC lifecycle via `keycloak-js`; the server only validates JWT access tokens.

| Parameter | Value | Managed By |
|---|---|---|
| OIDC flow | Authorization Code + PKCE (public client) | keycloak-js (browser) |
| Access token TTL | 10 minutes | Keycloak realm config |
| SSO session | 8 hours (= effective session length) | Keycloak |
| Token storage | In-memory (JS), never persisted to disk | keycloak-js |
| Token refresh | Silent refresh via Keycloak SSO session | keycloak-js |
| Session expiry | SSO session timeout → redirect to Keycloak login | keycloak-js |
| Logout | `keycloak.logout()` → Keycloak revokes session | keycloak-js |
| Role/membership changes | Propagate on next token refresh (≤ 10 minutes) | Keycloak + AuthContext DB lookup |
| Server-side auth state | **None.** Server validates Bearer JWTs and builds AuthContext from DB per request. | — |
| Client secret | **None.** Public client with PKCE replaces confidential client. | — |

### 6.2 Architecture

```
┌──────────────────────────────────────────────────────────────────────────┐
│  Browser                                                                 │
│                                                                          │
│  ┌─────────────┐    ┌──────────┐    ┌────────────────────────────────┐  │
│  │ keycloak-js  │    │  HTMX    │    │  App Shell (base.html)         │  │
│  │              │    │          │    │  Served once, unauthenticated  │  │
│  │ OIDC + PKCE  │    │ Bearer   │    │  Bootstraps keycloak-js        │  │
│  │ Token mgmt   │───►│ header   │───►│  + HTMX content loading        │  │
│  │ Silent renew │    │ on every │    └────────────────────────────────┘  │
│  └──────┬───────┘    │ request  │                                        │
│         │            └────┬─────┘                                        │
└─────────┼─────────────────┼──────────────────────────────────────────────┘
          │ OIDC            │ Authorization: Bearer <jwt>
          │ authz+token     │
          ▼                 ▼
┌─────────────────┐  ┌──────────────────────────────────┐
│    Keycloak      │  │   Fresnel API Server (Go)         │
│    (IdP)         │  │                                    │
│                  │  │  JWKS validation → AuthContext     │
│  Login, MFA,     │  │  (from DB, per request)            │
│  SSO session,    │  │                                    │
│  token issuance  │  │  No code exchange, no refresh,     │
│                  │  │  no cookies, no CSRF, no session   │
└─────────────────┘  └──────────────────────────────────┘
```

### 6.3 Request Flow

```
On each API request:
  1. Read Authorization: Bearer <jwt> header
  2. Validate JWT signature locally (Keycloak JWKS, cached)
  3. Check expiry (with 60s grace)
  4. Extract `sub` claim → query app DB for user, memberships, roles, root designations
  5. Build AuthContext → proceed to Cedar Gate
  6. If no Bearer or invalid → 401 JSON response
     (keycloak-js handles re-login or token refresh client-side)
```

The server is **fully stateless**. No cookies, no session store, no token cache, no refresh logic. All authentication state lives in the browser (managed by keycloak-js) and Keycloak (SSO session).

### 6.4 Login Flow

```
User                Browser (keycloak-js)     nginx         API Server        Keycloak
 │                    │                         │               │                │
 │  navigate to /     │                         │               │                │
 │───────────────────►│                         │               │                │
 │                    │  GET / (unauthenticated) │               │                │
 │                    │─────────────────────────►│──────────────►│                │
 │                    │  ◄── app shell HTML ─────│◄──────────────│                │
 │                    │                         │               │                │
 │                    │  keycloak.init()         │               │                │
 │                    │  PKCE: generate          │               │                │
 │                    │  code_verifier +         │               │                │
 │                    │  code_challenge          │               │                │
 │                    │─────────────────────────────────────────────────────────►│
 │                    │  302 → /auth?code_challenge=...&code_challenge_method=S256
 │                    │                         │               │                │
 │  Keycloak login (TOTP MFA)                  │               │                │
 │◄───────────────────│─────────────────────────────────────────────────────────►│
 │  credentials+TOTP  │                         │               │                │
 │───────────────────►│─────────────────────────────────────────────────────────►│
 │                    │                         │               │                │
 │                    │  302 + auth code         │               │                │
 │                    │◄────────────────────────────────────────────────────────│
 │                    │                         │               │                │
 │                    │  POST /token             │               │                │
 │                    │  code + code_verifier    │               │                │
 │                    │  (no client_secret)      │               │                │
 │                    │─────────────────────────────────────────────────────────►│
 │                    │  ◄── access_token ──────────────────────────────────────│
 │                    │  (stored in JS memory)   │               │                │
 │                    │                         │               │                │
 │                    │  HTMX GET /api/v1/dashboard              │                │
 │                    │  Authorization: Bearer <jwt>             │                │
 │                    │─────────────────────────►│──────────────►│                │
 │                    │                         │  validate JWT  │                │
 │                    │                         │  build AuthCtx │                │
 │                    │  ◄── HTML fragment ──────│◄──────────────│                │
 │                    │                         │               │                │
 │  rendered page     │                         │               │                │
 │◄───────────────────│                         │               │                │
```

### 6.5 Logout

User clicks logout → `keycloak.logout({ redirectUri: origin })` → browser redirects to Keycloak's end_session_endpoint → Keycloak destroys SSO session → browser redirected back to app → keycloak-js sees no session → redirects to login. The server has no logout endpoint and performs no action.

### 6.6 External SSO

External SSO is **entirely a Keycloak configuration concern**. The Fresnel application is unaware of upstream identity providers.

Keycloak supports identity brokering: an upstream OIDC or SAML IdP is configured in the Keycloak admin console. When a user from an organization with an external IdP authenticates, Keycloak redirects to their IdP, receives the assertion, maps claims, and issues Fresnel-scoped tokens. The API server sees only Keycloak-issued tokens regardless of the upstream provider.

Adding or removing an external SSO provider is a Keycloak admin configuration change — zero application code, zero deployment, zero downtime.

### 6.7 API (JSON) Authentication

API consumers (automation, agents) obtain tokens directly from Keycloak's token endpoint (client credentials grant with a separate confidential client, or resource owner password grant for service accounts) and send the access token as a Bearer header. Same validation path — JWKS, claims extraction, AuthContext. The API consumer manages its own token lifecycle.

### 6.8 CSRF

Not applicable. The server does not use cookies for authentication. All authenticated requests carry an `Authorization: Bearer` header, which is not automatically attached by the browser. This eliminates CSRF as a threat class — no CSRF middleware is needed.

---

## 7. Content Negotiation & Rendering

### 7.1 Dual Representation

Every endpoint serves both HTML (HTMX fragments) and JSON from the same handler:

```go
func (h *EventHandler) List(w http.ResponseWriter, r *http.Request) {
    auth := middleware.GetAuthContext(r.Context())
    filters := parseEventFilters(r)

    events, total, err := h.eventService.List(r.Context(), auth, filters)
    if err != nil {
        h.renderError(w, r, err)
        return
    }

    switch middleware.GetContentType(r.Context()) {
    case ContentTypeHTML:
        h.templates.Render(w, "events/list", map[string]any{
            "events": events,
            "total":  total,
            "filters": filters,
            "auth":   auth,
        })
    case ContentTypeJSON:
        json.NewEncoder(w).Encode(EventListResponse{
            Events: events,
            Total:  total,
        })
    }
}
```

### 7.2 HTMX Integration

The UI is an app shell served once (unauthenticated), with `keycloak-js` handling OIDC login (Authorization Code + PKCE). After authentication, all content is loaded and navigated via HTMX. A `htmx:configRequest` listener attaches `Authorization: Bearer <token>` to every request automatically.

- Navigation: `hx-get` with `hx-push-url` for URL updates without full reloads.
- Forms: `hx-post` / `hx-put` with `hx-target` for inline feedback.
- Dashboard updates: `hx-trigger="every 60s"` for periodic refresh of status tree (or manual refresh button).
- Side panel: `hx-get="/api/v1/dashboard/{type}/{id}/timeline" hx-target="#side-panel"` for timeline loading on node selection.
- Token refresh: `keycloak-js` silently refreshes the access token via the Keycloak SSO session. A 401 response triggers a token refresh attempt, falling back to re-login.

### 7.3 Markdown Editor

The event/report creation forms include a WYSIWYG Markdown editor. This is the one significant client-side JavaScript component.

Recommended approach: **Milkdown** (MIT licensed, ProseMirror-based, Markdown-native, plugin architecture). The editor produces Markdown source; the server stores Markdown; the server renders Markdown to HTML for display using a sanitizing renderer (goldmark with a strict HTML sanitizer — no raw HTML tags, no scripts, no event handlers pass through).

**CSP compatibility**: The editor must work under the strict CSP. Milkdown does not require `unsafe-eval`. It does require `unsafe-inline` for styles (already allowed in the CSP for HTMX/Markdown rendering).

---

## 8. Starlark Integration

### 8.1 Execution Model

The `go-starlark` library is embedded in the API server. Formula execution happens synchronously during dashboard requests, with caching.

```go
type FormulaEngine interface {
    // Evaluate a status aggregation formula
    Evaluate(formula string, children []ChildStatus) (AssessedStatus, error)

    // Validate a formula without executing it
    Validate(formula string) error
}

type ChildStatus struct {
    ID          uuid.UUID
    Status      AssessedStatus
    Weight      float64
    LastUpdated time.Time
}
```

### 8.2 Execution Constraints

- **Timeout**: 100ms maximum per formula evaluation. Starlark guarantees termination but complex formulas might be slow.
- **Memory**: Limited memory allocation per evaluation (configurable, default 1MB).
- **No I/O**: Starlark has no filesystem, network, or system access by design.
- **Input**: Only the `children` list is provided. No access to event content, user data, or anything outside the formula's scope.
- **Output**: Must return a valid AssessedStatus string or the default formula is used as fallback.

### 8.3 Caching

Dashboard status is computed and cached per-node with a configurable TTL (default 60 seconds). Cache invalidation on:
- New status report for a child node.
- Event status/impact change in a child node.
- Formula change for the node.

---

## 9. Nudge Scheduler

### 9.1 Implementation

A goroutine within the API server process that runs on a cron-like schedule. Not a separate service for PoC.

```go
type NudgeScheduler struct {
    eventStore  EventStore
    userStore   UserStore
    auditStore  AuditStore
    mailer      Mailer
    // ...
}

// Runs every 15 minutes, checks if any user's EOB has arrived
func (n *NudgeScheduler) Tick(ctx context.Context) error {
    // 1. Find users whose EOB is now (within the 15-minute window)
    // 2. For each user, find their open events with impact >= threshold
    // 3. Check last update timestamp against nudge rules
    // 4. Check escalation state
    // 5. Send emails via Mailer interface
    // 6. Log nudge/escalation to audit
}
```

### 9.2 Mailer Interface

```go
type Mailer interface {
    Send(ctx context.Context, to string, subject string, body string) error
}
```

PoC implementation: SMTP relay. Interface allows swapping to a queue-based sender later.

---

## 10. Deployment Architecture (PoC)

### 10.1 Docker Compose

```yaml
# Conceptual — not the actual file
services:
  nginx:
    image: nginx:1.27-alpine         # pinned
    # TLS certs mounted, config with security headers, rate limiting, ModSecurity
    ports: ["443:443"]
    depends_on: [api]

  api:
    build: ./                          # Fresnel API server (Go)
    environment:
      DATABASE_URL: postgres://...
      KEYCLOAK_ISSUER: https://...            # internal (JWKS validation)
      KEYCLOAK_EXTERNAL_URL: https://...      # browser-facing (keycloak-js config)
      KEYCLOAK_CLIENT_ID: fresnel-app         # public OIDC client
      CLAMAV_SOCKET: /var/run/clamav/clamd.sock
    depends_on: [postgres, keycloak, clamav]

  postgres:
    image: pgvector/pgvector:pg16      # PostgreSQL 16 + pgvector, pinned
    volumes: [pgdata:/var/lib/postgresql/data]

  keycloak:
    image: quay.io/keycloak/keycloak:26.0  # pinned
    # Configured with Fresnel realm, OIDC public client (PKCE), TOTP enforcement

  clamav:
    image: clamav/clamav:1.4           # pinned
    volumes: [clamsock:/var/run/clamav]

  ollama:                              # Present but not connected
    image: ollama/ollama:0.4           # pinned
    profiles: ["ai"]                   # only starts with --profile ai

volumes:
  pgdata:
  clamsock:
```

### 10.2 vSphere Deployment

Single VM running Docker Compose. VM requirements (PoC):
- 4 vCPU, 16 GB RAM, 100 GB disk (SSD-backed for database performance).
- Ubuntu 24.04 LTS or AlmaLinux 9 (stable, long-support).
- LUKS full-disk encryption.
- Firewall: nftables with explicit allow rules for 443/tcp inbound, SMTP outbound (port 25/587), and DNS.

### 10.3 Backup

- PostgreSQL `pg_dump` daily via cron, stored to a separate disk or NFS mount.
- Keycloak exports realm config nightly.
- Retention: 30 days of daily backups for PoC.

---

## 11. Dependency Manifest

All dependencies pinned to specific versions in `go.mod`. Key dependencies:

| Dependency | Purpose | Version Strategy |
|---|---|---|
| Go standard library | HTTP server, crypto, JSON, templates | Go 1.23.x (latest stable) |
| `github.com/cedar-policy/cedar-go` | Cedar policy evaluation | Pin to latest stable |
| `go.starlark.net` | Starlark formula execution | Pin to latest stable |
| `github.com/jackc/pgx/v5` | PostgreSQL driver | Pin to latest stable |
| `github.com/Masterminds/squirrel` | SQL query builder | Pin to latest stable |
| `github.com/yuin/goldmark` | Markdown → HTML rendering | Pin to latest stable |
| `github.com/microcosm-cc/bluemonday` | HTML sanitization (post-Markdown render) | Pin to latest stable |
| `github.com/google/uuid` | UUID generation | Pin to latest stable |
| HTMX | Frontend interactivity | Vendored JS file (not CDN), pinned version |
| keycloak-js | OIDC Authorization Code + PKCE client | Loaded from the Keycloak instance (`/js/keycloak.min.js`); vendor for air-gapped deploy |
| Milkdown | Markdown WYSIWYG editor | Vendored JS bundle, pinned version |

**No external CDN dependencies.** HTMX and Milkdown are vendored into the binary. `keycloak-js` is served by the Keycloak instance (part of the deployment). For fully air-gapped deployment, vendor `keycloak.min.js` into the static directory.

---

## 12. Interface Boundaries for Deferred Features

These are the explicit points where Phase 1/2 features attach to the PoC architecture. They exist as interfaces or stubs, not implementations.

### 12.1 Federation

- `source_instance` field on all domain objects (populated as "local" in PoC).
- API endpoints `/api/v1/federation/*` registered, return 501.
- `EventStore` interface can be wrapped by a `FederatedEventStore` that aggregates local and remote results.
- Keycloak configured with a single realm; federation adds trust relationships to remote Keycloak instances.

### 12.2 Sovereign Mode

- Cedar policy schema includes `forbid` policy templates scoped to org data plane.
- Adding sovereign mode is a Cedar policy change — `forbid` policies for external principals on the org's resources. The `CedarEvaluator.FilterPermitted` call in the service layer enforces it automatically with no code changes.
- Dashboard rendering already handles "restricted" status display path (returns UNKNOWN for nodes the user can't see).

### 12.3 AI Agents

- `SystemAgent` principal type defined in Cedar schema (no policies referencing it in PoC).
- `event_embeddings` table created but unpopulated.
- Ollama container in compose file under the `ai` profile.
- Agent 1 attaches as: a background goroutine that reads from `EventStore`, writes to `CorrelationStore` with `correlation_type = 'SUGGESTED'`, and publishes alerts through a `NotificationService` interface (stubbed in PoC).

### 12.4 Break-Glass

- Audit entry severity `CRITICAL` and action type `BREAK_GLASS` defined in the audit schema.
- API endpoint `/api/v1/orgs/{id}/break-glass` registered, returns 501.
- `Mailer` interface used for break-glass notifications (same as nudge system).

### 12.5 Firewall Integration

- No interface defined in PoC. This is explicitly deferred to the point where the API contract between app and firewall is designed, which requires security review beyond PoC scope.

---

## 13. Risks and Architectural Concerns

**Risk 1 — Post-Query Filtering Performance at Scale.** Cedar evaluates every candidate row returned by the storage layer. At PoC scale (~10k events, ~500 candidate rows per query), this adds ~1-5ms — negligible. At production scale (~100k events), large unfiltered queries could become expensive. Mitigation: functional pre-filtering in the storage layer (sector context, date ranges, status filters) reduces candidate sets. Cedar-informed query hints can be added as a performance optimization if needed, but correctness always flows from Cedar, not SQL. Accepted for now — monitor and optimize when data justifies it.

**Risk 2 — Starlark DoS via Formula Complexity.** Although Starlark guarantees termination and we impose a 100ms timeout, a node with hundreds of children and a complex formula could still consume CPU. Mitigation: limit children-per-formula to a configurable maximum (default 500), and cache aggressively. Accepted for now.

**Risk 3 — Single Process Scheduler.** The nudge scheduler runs as a goroutine in the API server. If the server restarts during the EOB window, some nudges may be missed or duplicated. Mitigation: nudge state (last nudge sent per event) is stored in the database, and the scheduler is idempotent — it checks "was a nudge already sent today for this event?" before sending.

**Risk 4 — Markdown Rendering and CSP.** HTMX injects HTML fragments into the DOM, which is inherently safe under CSP. The Markdown rendering pipeline is the critical XSS surface. Mitigation: goldmark renders Markdown to HTML, then bluemonday sanitizes with a strict allowlist (no `on*` attributes, no `script` tags, no `javascript:` URLs, no raw HTML passthrough). **The Markdown editor library selection must be validated against the CSP policy before adoption.** Specifically: the editor must function without `unsafe-eval` and must not require CDN-loaded resources. This is an explicit selection criterion, not an afterthought. Test the full pipeline with adversarial Markdown input as part of the security validation.

**Risk 5 — Keycloak Version Coupling.** Keycloak's OIDC behavior and admin API change between major versions. Pinning to a specific version is necessary, but Keycloak's security patches sometimes require version bumps that change behavior. Mitigation: abstract Keycloak interaction behind an interface (`IdentityProvider`) with integration tests against the specific pinned version.

---

*This architecture document is ready for review. The next work item is implementation planning: task breakdown, dependency order, and development sequence.*