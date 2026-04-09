# Fresnel — Implementation Plan

**Version**: 0.1
**Status**: Pending approval
**Prerequisites**: Requirements Specification v0.1.1, Architecture Document v0.1

---

## 1. Project Structure

```
fresnel/
├── cmd/
│   └── fresnel/
│       └── main.go                    # Entry point: config, wiring, server start
├── internal/
│   ├── config/
│   │   └── config.go                  # Env-based configuration struct
│   ├── domain/                        # Pure domain types — no dependencies
│   │   ├── event.go
│   │   ├── statusreport.go
│   │   ├── campaign.go
│   │   ├── correlation.go
│   │   ├── organization.go
│   │   ├── sector.go
│   │   ├── user.go
│   │   ├── audit.go
│   │   ├── enums.go                   # TLP, Impact, Status, AssessedStatus, EventType
│   │   └── auth.go                    # AuthContext, ScopeEntry, RoleAssignment
│   ├── storage/
│   │   ├── interfaces.go              # All store interfaces (EventStore, etc.)
│   │   └── postgres/
│   │       ├── postgres.go            # Connection pool, migration runner
│   │       ├── event.go
│   │       ├── statusreport.go
│   │       ├── campaign.go
│   │       ├── correlation.go
│   │       ├── sector.go
│   │       ├── organization.go
│   │       ├── user.go
│   │       ├── audit.go
│   │       ├── policy.go
│   │       ├── nudge.go
│   │       └── formula.go
│   ├── service/
│   │   ├── event.go
│   │   ├── statusreport.go
│   │   ├── campaign.go
│   │   ├── correlation.go
│   │   ├── dashboard.go
│   │   ├── organization.go
│   │   ├── sector.go
│   │   ├── user.go
│   │   ├── audit.go
│   │   ├── attachment.go              # ClamAV scanning + file storage
│   │   └── nudge.go                   # Scheduler + escalation logic
│   ├── cedar/
│   │   ├── evaluator.go               # CedarEvaluator implementation
│   │   ├── resource.go                # Domain object → CedarResource mapping
│   │   ├── loader.go                  # Policy loading + cache invalidation
│   │   └── policies/                  # Embedded Cedar policy templates
│   │       ├── schema.cedar
│   │       ├── platform_root.cedar
│   │       ├── sector_root.cedar
│   │       ├── org_root.cedar
│   │       ├── org_admin.cedar
│   │       ├── content_admin.cedar
│   │       ├── contributor.cedar
│   │       ├── viewer.cedar
│   │       ├── liaison.cedar
│   │       └── tlp.cedar              # TLP-based conditions
│   ├── starlark/
│   │   ├── engine.go                  # FormulaEngine implementation
│   │   └── default_formula.star       # Default weighted-average formula
│   ├── markdown/
│   │   └── render.go                  # goldmark + bluemonday pipeline
│   ├── mail/
│   │   └── smtp.go                    # Mailer implementation
│   └── http/
│       ├── server.go                  # Server setup, graceful shutdown
│       ├── router.go                  # Route registration
│       ├── middleware/
│       │   ├── auth.go                # Bearer JWT validation → AuthContext from DB
│       │   ├── cedar_gate.go          # Tier 1 coarse-grained authz
│       │   ├── content_neg.go         # Accept header → renderer selection
│       │   └── logging.go             # Request/response structured logging
│       └── handlers/
│           ├── event.go
│           ├── statusreport.go
│           ├── campaign.go
│           ├── correlation.go
│           ├── dashboard.go
│           ├── sector.go
│           ├── organization.go
│           ├── user.go
│           ├── audit.go
│           ├── attachment.go
│           ├── auth.go                # App shell handler (keycloak-js bootstrap)
│           ├── health.go
│           └── federation.go          # 501 stubs
├── migrations/
│   ├── 001_fresnel_schema.sql
│   ├── 002_fresnel_iam_schema.sql
│   ├── 003_fresnel_audit_schema.sql
│   ├── 004_pgvector.sql
│   └── 005_seed_platform_config.sql
├── templates/                         # Go html/template files
│   ├── layouts/
│   │   └── base.html                  # App shell: keycloak-js bootstrap, nav, HTMX
│   ├── dashboard/
│   │   ├── index.html                 # Hierarchical tree
│   │   └── timeline.html              # Side panel fragment
│   ├── events/
│   │   ├── list.html
│   │   ├── detail.html
│   │   ├── form.html                  # Create/edit (shared)
│   │   └── updates.html               # Update log fragment
│   ├── status_reports/
│   │   ├── list.html
│   │   ├── detail.html
│   │   └── form.html
│   ├── campaigns/
│   │   ├── list.html
│   │   ├── detail.html
│   │   └── form.html
│   ├── admin/
│   │   ├── orgs.html
│   │   ├── sectors.html
│   │   ├── users.html
│   │   ├── roles.html
│   │   ├── policies.html
│   │   └── formula.html               # Starlark editor
│   ├── audit/
│   │   └── log.html
│   ├── partials/
│   │   ├── nav.html
│   │   ├── context_selector.html      # Org/sector context switcher
│   │   ├── tlp_badge.html
│   │   ├── impact_badge.html
│   │   ├── status_badge.html
│   │   └── pagination.html
│   └── errors/
│       ├── 403.html
│       ├── 404.html
│       └── 500.html
├── static/
│   ├── htmx.min.js                    # Vendored
│   ├── app.js                         # keycloak-js init + HTMX Bearer integration
│   ├── milkdown/                      # Vendored bundle
│   └── css/
│       └── fresnel.css                # Application styles
├── deploy/
│   ├── docker-compose.yml
│   ├── docker-compose.dev.yml         # Dev overrides (hot reload, debug ports)
│   ├── nginx/
│   │   ├── nginx.conf
│   │   ├── security-headers.conf
│   │   └── certs/                     # Self-signed for dev, mount real for prod
│   ├── keycloak/
│   │   └── fresnel-realm.json         # Realm import file
│   └── postgres/
│       └── init.sql                   # Schema creation (runs on first start)
├── scripts/
│   ├── seed-dev-data.sh               # Populate dev hierarchy + test users
│   └── backup.sh                      # pg_dump + keycloak export
├── docs/
│   ├── REQUIREMENTS.md
│   ├── ARCHITECTURE.md
│   └── IMPLEMENTATION_PLAN.md
├── go.mod
├── go.sum
├── Dockerfile                         # Multi-stage: build Go binary, copy to scratch/alpine
└── Makefile                           # build, test, lint, migrate, seed, compose-up
```

---

## 2. Development Environment

| Tool | Version | Purpose |
|---|---|---|
| Go | 1.23.x (latest stable) | Application language |
| Docker + Docker Compose | Latest stable | Local infrastructure |
| PostgreSQL | 16 (via pgvector image) | Database |
| Keycloak | 26.0 | Identity provider |
| ClamAV | 1.4 | Virus scanning |
| golangci-lint | Latest | Linter |
| go test | Standard | Testing |

**Local development flow:**

1. `make compose-up` — starts PostgreSQL, Keycloak, ClamAV, nginx.
2. `make migrate` — runs SQL migrations against local PostgreSQL.
3. `make seed` — populates dev hierarchy, test users, sample data.
4. `make run` — builds and runs the Go binary with hot reload (via `air` or manual restart).
5. Browser → `https://localhost` → redirected to Keycloak login → test user credentials.

---

## 3. Milestones

### M0: Project Bootstrap

**Goal**: `docker compose up` starts all infrastructure. Go binary compiles, connects to PostgreSQL and Keycloak, serves the health endpoint.

| ID | Task | Details |
|---|---|---|
| M0.1 | Initialize Go module | `go mod init`, add core dependencies (pgx, uuid, goldmark, bluemonday). Don't add cedar-go or starlark yet — resolve version compatibility first. |
| M0.2 | Create Dockerfile | Multi-stage build: Go build stage → minimal runtime image. |
| M0.3 | Create Docker Compose | PostgreSQL (pgvector), Keycloak, ClamAV, nginx, API server. Dev compose overlay with debug ports. |
| M0.4 | Configure Keycloak realm | Create `fresnel-realm.json`: realm name, OIDC **public** client (Authorization Code + PKCE), required TOTP, brute-force protection. No client secret. Export as JSON for idempotent import. |
| M0.5 | Write SQL migrations | All tables from Architecture Section 5.2 as migration files. Include schema creation (`CREATE SCHEMA`), pgvector extension, and seed `platform_config` row. |
| M0.6 | Implement migration runner | On API server startup, apply pending migrations (use `golang-migrate` or manual version tracking in a `schema_migrations` table). |
| M0.7 | Implement config loader | Environment-variable-based config struct: `DATABASE_URL`, `KEYCLOAK_ISSUER`, `KEYCLOAK_CLIENT_ID`, `KEYCLOAK_EXTERNAL_URL`, `CLAMAV_SOCKET`, `SMTP_HOST`, `LISTEN_ADDR`. No client secret (public client). No HMAC secret (no CSRF needed with Bearer auth). |
| M0.8 | Implement health endpoint | `GET /api/v1/health` — unauthenticated, returns DB connectivity and Keycloak reachability status. |
| M0.9 | Write nginx config | TLS termination (self-signed cert for dev), proxy_pass to API server, all security headers from Requirements Section 9.2. Rate limiting rules. |
| M0.10 | Create Makefile | Targets: `build`, `test`, `lint`, `compose-up`, `compose-down`, `migrate`, `seed`, `run`. |

**Acceptance**: `make compose-up && curl -k https://localhost/api/v1/health` returns `{"status":"ok"}`.

---

### M1: Authentication

**Goal**: Browser login flow works end-to-end with Authorization Code + PKCE. AuthContext is populated. Protected API endpoints return 401 for unauthenticated requests.

**Depends on**: M0

| ID | Task | Details |
|---|---|---|
| M1.1 | Implement app shell page | `GET /{path...}` catch-all — serves the bootstrap HTML page with `keycloak-js` configuration (Keycloak URL, realm, client ID injected via `<meta>` tags). This is the single unauthenticated HTML entry point. |
| M1.2 | Implement `app.js` (keycloak-js bootstrap) | Client-side JS that: (1) loads `keycloak-js` from the Keycloak instance, (2) initializes with `onLoad: 'login-required'` and `pkceMethod: 'S256'`, (3) attaches `Authorization: Bearer <token>` to all HTMX requests via `htmx:configRequest`, (4) refreshes the token periodically (every 30s, refresh if expiring within 60s), (5) handles 401 responses with token refresh or re-login, (6) loads initial content for the current URL path. |
| M1.3 | Implement Bearer token validation middleware | Reads `Authorization: Bearer <jwt>` header. Validates JWT signature against Keycloak JWKS (cached, 15-minute TTL). Checks expiry (60s grace). For `/api/` routes without a valid Bearer: 401 JSON. For non-API routes without Bearer: pass through (shell handler serves the bootstrap page). No cookies, no refresh, no token exchange on the server side. |
| M1.4 | Implement AuthContext builder | Extracts `sub` claim from validated JWT. Queries app DB for user record, org memberships, role assignments, root designations. Builds `AuthContext` struct, attaches to request context. The token provides identity; the database provides authorization data. This avoids Keycloak custom mappers and keeps Keycloak config minimal. The DB lookup is lightweight (three queries in a transaction) and the user record is small. |
| M1.5 | Create Keycloak test users | In `fresnel-realm.json` or via seed script: platform root, sector root, org root, org admin, contributor, viewer. Each with TOTP configured. |
| M1.6 | Implement content negotiation middleware | Inspect `Accept` header. Set renderer on context: `text/html` → template renderer, `application/json` → JSON encoder. Default to HTML for browser requests. |
| M1.7 | Implement request logging middleware | Structured JSON logs: timestamp, method, path, user_id (from AuthContext, if present), status code, latency, IP. |
| M1.8 | Implement base HTML shell template | `base.html`: static HTML shell with nav bar (user info populated by JS after auth), Keycloak config `<meta>` tags, HTMX + `app.js` script tags. No server-side auth rendering — this page is always served unauthenticated. |
| M1.9 | Wire router | Register middleware: logging → OIDC validator → content negotiation. Register routes: health (public), static files (public), API routes (authenticated), catch-all shell (public). No CSRF middleware (Bearer tokens eliminate CSRF). |

**Acceptance**: Navigate to `https://localhost/` → keycloak-js redirects to Keycloak → login with test credentials + TOTP → redirected back → keycloak-js exchanges code with PKCE → HTMX loads dashboard via Bearer token → user info displayed. `curl -H "Authorization: Bearer <token>" https://localhost/api/v1/dashboard` returns JSON. Token refresh works silently. Logout via keycloak-js clears session.

---

### M2: Authorization (Cedar) + Organizational Hierarchy

**Goal**: Cedar evaluates policies correctly. Sector/org/user CRUD works with proper access control. Audit logging captures all mutations.

**Depends on**: M1

| ID | Task | Details |
|---|---|---|
| M2.1 | Add cedar-go dependency | Resolve version, add to go.mod. Validate it compiles and runs a trivial policy evaluation. |
| M2.2 | Design Cedar entity schema | Define Cedar entity types (`Fresnel::User`, `Fresnel::Event`, `Fresnel::StatusReport`, `Fresnel::Campaign`, `Fresnel::Organization`, `Fresnel::Sector`), action types, and attribute shapes. Write as `.cedarschema` files. See Section 6 of this plan for the detailed design. |
| M2.3 | Write Cedar policy templates | One file per role (platform_root, sector_root, org_root, org_admin, content_admin, contributor, viewer, liaison). Plus TLP condition policies. Embed as Go embed files. See Section 6. |
| M2.4 | Implement CedarEvaluator | `IsPermitted(auth, action, resource)` and `FilterPermitted(auth, action, resources)`. Loads policy set from PolicyStore on startup, caches in memory, invalidates on change signal. |
| M2.5 | Implement Cedar Gate middleware (Tier 1) | Maps HTTP method + route → Cedar action + resource type. Evaluates via CedarEvaluator. Returns 403 on deny. |
| M2.6 | Implement domain-to-CedarResource mapping | Functions that convert each domain type (Event, StatusReport, etc.) into a `CedarResource` struct for Cedar evaluation. Includes TLP:RED recipient lookup from `tlp_red_recipients` table. |
| M2.7 | Implement PolicyStore | PostgreSQL-backed CRUD for `cedar_policies`, `role_assignments`, `root_designations`. |
| M2.8 | Implement AuditStore | PostgreSQL-backed append-only store. Insert only — the DB role for the app has no UPDATE/DELETE on `fresnel_audit`. |
| M2.9 | Implement audit middleware/helper | Service-layer helper that wraps mutations: captures before/after state, writes to AuditStore. Used by all service methods that create/update/delete. |
| M2.10 | Implement SectorStore + SectorService | CRUD for recursive sectors. Ancestry path management: on create, compute `ancestry_path` from parent. On move (if needed), recompute children paths. Depth validation (max 5). |
| M2.11 | Implement OrganizationStore + OrganizationService | CRUD for orgs. Includes `org_sector_memberships` management. Timezone field. |
| M2.12 | Implement UserStore + UserService | CRUD for users. Links to Keycloak `sub`. Primary org assignment. Multi-org membership via `user_org_memberships`. |
| M2.13 | Implement root designation management | Assign/reassign root for any scope. Validate: only self-reassignment or parent root can reassign. Audit as HIGH severity. |
| M2.14 | Implement role assignment management | Assign/revoke roles for users within scopes. Validate scope authority via Cedar. |
| M2.15 | Implement sector HTTP handlers | `GET /api/v1/sectors`, `GET /api/v1/sectors/{id}`, `GET /api/v1/sectors/{id}/children`. Dual-representation (HTML + JSON). |
| M2.16 | Implement organization HTTP handlers | `GET /api/v1/orgs`, `GET /api/v1/orgs/{id}`, org member management endpoints. |
| M2.17 | Implement user HTTP handlers | `GET /api/v1/users`, `GET /api/v1/users/{id}`, `GET /api/v1/users/me`, `PUT /api/v1/users/{id}`. |
| M2.18 | Implement audit HTTP handler | `GET /api/v1/audit` with scope-based filtering. Cedar-gated (only roots and admins). |
| M2.19 | Write Cedar policy test suite | Test matrix: each role × each action × each resource scope. Verify permit/deny matches the requirements role table. Test TLP conditions. Test root hierarchy inheritance. This is a critical quality gate. |
| M2.20 | Seed initial policies | On first startup (or via migration), render Cedar policy templates for the default hierarchy and insert into `cedar_policies`. |
| M2.21 | Implement admin UI templates | Basic HTML pages for sector management, org management, user management, role assignment. Functional, not polished. |

**Acceptance**: Platform root can create sectors (nested), create orgs within sectors, create users, assign roles. A contributor can only see endpoints allowed by their role. Audit log captures all mutations with correct severity. Cedar test suite passes with full coverage of the role × action × scope matrix.

---

### M3: Events (Core Domain)

**Goal**: Full event lifecycle works with authorization, revision history, updates, attachments, and TLP enforcement.

**Depends on**: M2

| ID | Task | Details |
|---|---|---|
| M3.1 | Implement domain types | `Event`, `EventRevision`, `EventUpdate`, `Attachment` structs with validation methods. Enum types for event status lifecycle (OPEN → INVESTIGATING → MITIGATING → RESOLVED → CLOSED) and event types (from Requirements 3.3.1). |
| M3.2 | Implement EventStore | Full CRUD with revision tracking. On update: insert new `event_revision` row, bump `revision_number`. Functional filters: sector_context, organization_id, status, impact, event_type, date range, text search on title. Pagination + sorting. |
| M3.3 | Implement EventUpdateStore | Append-only within an event. When update includes `impact_change` or `status_change`, also update the parent event's fields (in a transaction). |
| M3.4 | Implement EventService | CRUD with Cedar row-level filtering. Business rules: TLP on updates cannot be less restrictive than parent event. Status transitions validated. Sector context immutable after creation. Audit logging on all mutations. |
| M3.5 | Implement AttachmentStore | Metadata in PostgreSQL, file bytes on local filesystem. Storage path not web-accessible. |
| M3.6 | Implement AttachmentService | Upload flow: validate file type + size → write to temp → scan with ClamAV → if clean, move to permanent storage + insert metadata. If quarantined, audit log + notify. Enforce max 10 attachments per event. Download: serve through authenticated endpoint with Cedar check. |
| M3.7 | Implement ClamAV client | Connect to clamd via Unix socket. Scan file, parse response (OK/FOUND). Handle timeouts and connection errors gracefully (log error, reject upload with "scan unavailable" rather than allowing unscanned files). |
| M3.8 | Implement TLP:RED recipient management | When creating/editing a TLP:RED resource, require recipients list. Store in `tlp_red_recipients`. CedarResource mapping includes recipient user IDs in Attributes for policy evaluation. |
| M3.9 | Implement event HTTP handlers | All event endpoints from API spec. Dual-representation. Include sub-resource endpoints for updates, correlations (placeholder), relationships (placeholder), attachments. |
| M3.10 | Implement event HTML templates | List view (filterable, sortable, paginated), detail view (with update log, revision history), create/edit form (with sector context selector for multi-sector org users, TLP selector, impact selector, event type dropdown). |
| M3.11 | Integrate Markdown rendering | `markdown.Render(input string) template.HTML` — goldmark converts Markdown to HTML, bluemonday sanitizes. Used in all detail views for body fields. Test with adversarial input (script tags, event handlers, javascript: URLs, nested Markdown injection). |

**Acceptance**: Users can create events with all required fields. Events respect TLP visibility. Revision history tracks all edits. Event updates can change impact/status. Attachments upload, scan, and download correctly. A ClamAV-detected virus is quarantined. TLP:RED events are only visible to named recipients.

---

### M4: Status Reports, Campaigns, Correlations

**Goal**: All remaining domain objects work with full CRUD and authorization.

**Depends on**: M3

| ID | Task | Details |
|---|---|---|
| M4.1 | Implement StatusReportStore | CRUD with revision tracking (via `status_report_revisions`). Referenced events via `status_report_events` junction table. Filters: sector_context, scope_type, scope_ref, date range, assessed_status. |
| M4.2 | Implement StatusReportService | CRUD with Cedar row-level filtering. Business rules: scope validation (org-scoped reports require author in org, sector-scoped require sector root or content admin). Audit logging. |
| M4.3 | Implement status report HTTP handlers + templates | All endpoints. List, detail (with referenced events clickable), create/edit form (scope selector, period selector, assessed status picker). |
| M4.4 | Implement CampaignStore | CRUD for campaigns. Event linking via `campaign_events`. |
| M4.5 | Implement CampaignService | CRUD with Cedar filtering. Campaign visibility is independent of member event visibility. When listing campaign events, filter through Cedar — events the user can't see are counted as "N organizations have restricted content." |
| M4.6 | Implement campaign HTTP handlers + templates | All endpoints. Detail view shows events grouped by org/sector with restricted content indicator. |
| M4.7 | Implement CorrelationStore | CRUD for correlations. Canonical ordering (event_a_id < event_b_id). |
| M4.8 | Implement CorrelationService | CRUD with Cedar filtering. Visibility rule: user must have access to both linked events. Suggested correlations (type=SUGGESTED) are only visible to the event's org users until confirmed. |
| M4.9 | Implement EventRelationshipStore + Service | CRUD for directional relationships. Visibility: user must see both events (except "sanitized_version" where source is hidden). |
| M4.10 | Implement correlation/relationship HTTP handlers | Sub-resource endpoints under events. Add correlation from event detail page. |

**Acceptance**: Status reports can be created at org and sector scope. Revision history works. Campaigns link events across sectors. The "restricted content" indicator shows correctly. Correlations respect bidirectional visibility.

---

### M5: Dashboard + Starlark

**Goal**: The hierarchical dashboard renders the entire platform status tree with computed statuses, side panel timeline, and campaign section.

**Depends on**: M4

| ID | Task | Details |
|---|---|---|
| M5.1 | Add go-starlark dependency | Resolve version, verify compilation. |
| M5.2 | Implement FormulaEngine | Starlark execution with constraints: 100ms timeout, 1MB memory limit, no builtins beyond the children input. Parse formula, execute, extract result string. Validate function. Fallback to default formula on error. |
| M5.3 | Write default formula | Weighted average in Starlark. Map NORMAL=0, DEGRADED=1, IMPAIRED=2, CRITICAL=3, UNKNOWN=excluded. Threshold: <0.5→NORMAL, <1.5→DEGRADED, <2.5→IMPAIRED, ≥2.5→CRITICAL. |
| M5.4 | Implement FormulaStore | CRUD for `status_formulas`. Per-node (sector or platform). |
| M5.5 | Implement DashboardService | Build the hierarchical status tree: (1) load all sectors (recursive), (2) load all orgs within sectors, (3) for each leaf node (org), get latest status report's assessed_status, (4) for each parent node, evaluate formula over children, (5) cache result per-node with 60s TTL. Cedar-filter the tree: nodes the user can't see show as UNKNOWN/"restricted". |
| M5.6 | Implement status cache | In-memory cache keyed by node ID. TTL-based expiry (configurable, default 60s). Invalidation on: status report create/update, event status change, formula change. |
| M5.7 | Implement dashboard HTTP handler | `GET /api/v1/dashboard` returns the full tree (JSON or HTML). HTML version renders the collapsible/expandable hierarchy with status color coding. |
| M5.8 | Implement timeline HTTP handler | `GET /api/v1/dashboard/{node_type}/{id}/timeline` returns interleaved status reports + events for the selected node. Paginated, chronologically sorted. Cedar-filtered. |
| M5.9 | Implement dashboard HTML template | The primary view from Requirements 4.1.1. Collapsible tree (HTMX-driven expand/collapse). Status badges color-coded. Campaign section below. Click a node → HTMX loads timeline into side panel. Click a timeline entry → navigate to detail page. Auto-refresh every 60s via `hx-trigger`. |
| M5.10 | Implement formula management handler + template | `GET/PUT /api/v1/sectors/{id}/formula`. Admin UI: text editor for Starlark code, validate button (calls FormulaEngine.Validate), preview button (runs formula with current children, shows what status would be computed). |
| M5.11 | Implement campaign section on dashboard | List active campaigns below the hierarchy. Click → campaign detail view. |

**Acceptance**: Dashboard loads within 100ms. Status tree shows correct hierarchy with computed statuses. Clicking a node shows the timeline. Starlark formulas execute correctly. Custom formulas can be set and validated. Cache invalidation works (create an event, dashboard updates within 60s).

---

### M6: Nudge System

**Goal**: Email nudges are sent on schedule based on event impact. Escalation chain walks the hierarchy.

**Depends on**: M3 (events), M2 (org hierarchy)

| ID | Task | Details |
|---|---|---|
| M6.1 | Implement Mailer (SMTP) | `Send(ctx, to, subject, body)` via SMTP relay. TLS required. Timeout handling. Structured logging of send success/failure. |
| M6.2 | Implement NudgeStore | CRUD for `nudge_log` and `escalation_state`. Query: "has a nudge been sent for this event today?" Query: "what is the escalation level for this event?" |
| M6.3 | Implement NudgeScheduler | Goroutine with 15-minute tick. On each tick: (1) find all users whose EOB is within the current 15-minute window (user timezone → org timezone → platform default), (2) for each user, find open events where they're a contributor, (3) check nudge rules (impact × frequency), (4) check nudge_log for idempotency, (5) send email, (6) log to nudge_log + audit. |
| M6.4 | Implement escalation logic | If an event's last update is >1 business day old and no response from the current escalation level: advance `escalation_state.current_level`, send nudge to the next level (org root → parent sector root → ... → platform root). Escalation levels map to sector ancestry: level 0=contributors, 1=org root, 2+=sector roots walking up `sectors.ancestry_path`. |
| M6.5 | Implement weekly digest | Every Monday, aggregate all open events with impact > INFO into a per-user digest email. Respect org/sector scope. |
| M6.6 | Reset escalation on response | When an event update is created, reset `escalation_state.current_level` to 0 and update `last_response_at`. |
| M6.7 | Email templates | Plain-text email templates for: daily nudge, weekly nudge, escalation nudge, weekly digest. Include event title, impact, link to event detail page. |

**Acceptance**: Create a CRITICAL event, wait past EOB → nudge email sent. No duplicate nudges. Escalation advances after 1 business day with no response. Adding an event update resets escalation. Weekly digest arrives on schedule.

---

### M7: UI Polish + Markdown Editor

**Goal**: The UI is functional and presentable for stakeholder demos. Markdown editor works under CSP.

**Depends on**: M5 (dashboard exists), M3-M4 (all domain objects have basic templates)

| ID | Task | Details |
|---|---|---|
| M7.1 | Vendor HTMX | Download pinned version, place in `static/`. Verify works under CSP (no `unsafe-eval`). |
| M7.2 | Vendor Milkdown | Build or download pinned bundle. Place in `static/milkdown/`. Test under CSP: must work without `unsafe-eval`, must not require CDN resources. If Milkdown fails CSP validation, fall back to a simpler editor (EasyMDE, or plain textarea with Markdown preview). |
| M7.3 | Implement Markdown editor integration | Embed editor in event/status-report create/edit forms. Editor produces Markdown source, stored in DB. Preview button renders via server-side goldmark+bluemonday pipeline. |
| M7.4 | Style the application | CSS for: dashboard tree (status colors, indent levels, expand/collapse), forms (clean layout, TLP/impact/status selectors with color coding), detail pages, timeline, navigation. Responsive layout for various screen sizes (no mobile target, but should work on tablets). |
| M7.5 | Implement org context selector | For multi-org users: dropdown in nav bar. Sets `ActiveOrgContext` via cookie or request header. Affects which events the user creates (org context), which sector context options are shown, and the dashboard scope. |
| M7.6 | Implement sector context selector | For multi-sector org users creating events/reports: prominent selector with confirmation dialog (sector context is immutable after creation). |
| M7.7 | Implement TLP/Impact/Status badges | Consistent visual badges used across all views. TLP: colored labels (RED=red, AMBER=orange, GREEN=green, CLEAR=gray). Impact: colored dots matching the color scheme. Status: styled pills. |
| M7.8 | Implement revision history diff view | On event/report detail pages: expandable revision history. Each revision shows what changed (field-by-field diff). |
| M7.9 | Implement error pages | 403, 404, 500 error pages. Styled consistently with the application. |
| M7.10 | Adversarial Markdown testing | Feed the rendering pipeline with known XSS vectors: `<script>`, `<img onerror=...>`, `[link](javascript:...)`, HTML entities, nested markdown. Verify bluemonday strips all. |

**Acceptance**: The dashboard is visually clear and usable for a demo. Events/reports can be created with the Markdown editor. All badges render consistently. No XSS through the Markdown pipeline (verified by test).

---

### M8: Security Hardening + Deployment

**Goal**: nginx is fully hardened. ModSecurity + CRS active. Docker Compose is production-ready. Backup scripts work.

**Depends on**: M7

| ID | Task | Details |
|---|---|---|
| M8.1 | Configure ModSecurity + OWASP CRS | Install ModSecurity module in nginx. Add OWASP CRS rules. Test that normal operations work (HTMX POST/PUT requests, file uploads). Tune false positives. |
| M8.2 | Configure fail2ban | Monitor nginx access logs. Ban IPs with >10 failed auth attempts in 5 minutes. |
| M8.3 | Verify all security headers | Automated test: fetch every endpoint, verify all headers from Requirements 9.2 are present and correct. |
| M8.4 | Configure rate limiting | nginx `limit_req` zones: global (100 req/s per IP), login endpoint (5 req/min per IP), API (50 req/s per user). |
| M8.5 | Input validation audit | Review all handler input parsing. Verify: max field lengths, enum validation, UUID format validation, Markdown size limits. No SQL injection vectors (all queries parameterized via pgx). |
| M8.6 | Finalize Docker Compose | Production compose file: no debug ports, proper resource limits, restart policies, healthchecks for all services, log rotation. |
| M8.7 | Implement backup scripts | `backup.sh`: pg_dump to dated file, keycloak realm export, retention cleanup (30 days). Cron entry for daily execution. |
| M8.8 | Write deployment guide | Minimal ops documentation: how to deploy on a fresh vSphere VM (Ubuntu 24.04, LUKS, Docker, compose up), how to create the first platform root user, how to restore from backup. |
| M8.9 | Register federation + webhook stubs | All federation endpoints return 501 with `{"error": "not_implemented", "message": "Federation is planned for Phase 2"}`. Same for webhooks. Break-glass endpoint stub. |
| M8.10 | Final integration test pass | End-to-end: deploy from scratch on a clean environment, create hierarchy, create users with all roles, create events/reports/campaigns, verify dashboard, verify nudge email, verify audit log. |

**Acceptance**: Clean deployment on a fresh VM. All security headers present. ModSecurity blocks a test XSS attempt. Rate limiting rejects excessive requests. Backup + restore cycle works. Federation/webhook stubs return 501.

---

## 4. Critical Path

```
M0 (bootstrap) → M1 (auth) → M2 (cedar + hierarchy) → M3 (events) → M4 (reports/campaigns) → M5 (dashboard)
                                                                                                      ↓
                                                                                     M7 (UI polish) → M8 (hardening)
                                                        M3 ──→ M6 (nudge, parallel with M4/M5)
```

**The longest chain is**: M0 → M1 → M2 → M3 → M4 → M5 → M7 → M8

M6 (nudge system) can run in parallel with M4/M5 since it only depends on events (M3) and org hierarchy (M2).

M7 builds on all prior milestones but can start its CSS/styling work as soon as M5 delivers the dashboard. Milestone Editor integration (M7.2-M7.3) can start as soon as M3 delivers event forms.

---

## 5. Testing Strategy

| Layer | Tool | Scope |
|---|---|---|
| Domain types | `go test` | Validation methods, enum parsing, status transitions |
| Storage layer | `go test` + test PostgreSQL | Each store interface tested against real PostgreSQL (Docker). Functional queries, pagination, transactions. |
| Cedar policies | `go test` | **Critical.** Test matrix of every role × action × resource scope × TLP level. This is the primary correctness gate for authorization. |
| Service layer | `go test` + mocks | Business logic with mocked stores. Cedar evaluation with real policies against test scenarios. |
| HTTP handlers | `go test` + httptest | Request/response validation, content negotiation, error codes. |
| Integration | `go test` + full stack | End-to-end tests that start the server, authenticate via Keycloak, and exercise API endpoints. Fewer of these, focused on critical paths. |
| Markdown/XSS | `go test` | Dedicated test file with adversarial Markdown inputs. Verify sanitizer strips everything dangerous. |
| Starlark | `go test` | Formula execution with various child status combinations. Timeout enforcement. Malformed formula handling. |

**Test database**: Each storage test suite creates a fresh schema (or uses transactions that roll back) to ensure isolation. The test PostgreSQL instance runs in Docker alongside the development one, on a different port.

**Cedar test suite structure**: One test function per role. Each function tests a matrix of (action, resource_with_attributes) combinations and asserts permit/deny. Example:

```go
func TestContributorRole(t *testing.T) {
    tests := []struct {
        action   string
        resource CedarResource
        expect   bool
    }{
        {"create", eventInOwnOrg, true},
        {"create", eventInOtherOrg, false},
        {"edit", ownEvent, true},
        {"edit", otherUserEventSameOrg, false},
        {"view", eventTLPGreen, true},
        {"view", eventTLPRedNotRecipient, false},
        {"delete", anyEvent, false},
        {"manage_members", anyOrg, false},
        // ...
    }
    // ...
}
```

---

## 6. Cedar Policy Design

This section outlines the Cedar schema and policy templates that will be implemented in M2.

### 6.1 Entity Types

```
entity Fresnel::User {
    org_memberships: Set<Fresnel::Organization>,
    primary_org: Fresnel::Organization,
    roles: Set<Fresnel::RoleAssignment>,
    is_root: Bool,
    root_scope_type: String,    // "PLATFORM", "SECTOR", "ORG", or ""
    root_scope_id: String,      // UUID as string, or ""
};

entity Fresnel::Organization {
    sector: Fresnel::Sector,
};

entity Fresnel::Sector {
    parent: Fresnel::Sector,    // optional (nil for top-level)
    ancestry_path: String,
};

entity Fresnel::RoleAssignment {
    role: String,               // "PLATFORM_ROOT", "SECTOR_ROOT", "ORG_ROOT", etc.
    scope_type: String,
    scope_id: String,
};

entity Fresnel::Event {
    organization: Fresnel::Organization,
    sector_context: Fresnel::Sector,
    tlp: String,
    submitter: Fresnel::User,
};

entity Fresnel::StatusReport {
    organization: Fresnel::Organization,
    sector_context: Fresnel::Sector,
    scope_type: String,
    scope_ref: String,
    tlp: String,
    author: Fresnel::User,
};

entity Fresnel::Campaign {
    organization: Fresnel::Organization,
    tlp: String,
};
```

### 6.2 Policy Templates (Summary)

| Template | Logic |
|---|---|
| `platform_root` | Permit all actions on all resource types. |
| `sector_root` | Permit all actions on resources where `resource.sector_context.ancestry_path` starts with the root's sector ancestry path. |
| `org_root` | Permit all actions on resources where `resource.organization == root's org`. |
| `org_admin` | Permit all data-plane actions + `manage_members` within org. |
| `content_admin` | Permit `edit` on Event and StatusReport regardless of org. |
| `contributor` | Permit `create` Event within own org. Permit `edit` on own events (`resource.submitter == principal`). Permit `link` (correlations). Permit `view` subject to TLP. |
| `viewer` | Permit `view` subject to TLP and scope. |
| `liaison` | Permit `view` on resources in assigned organizations, subject to TLP. |
| `tlp_clear` | Permit `view` for any authenticated user when `resource.tlp == "CLEAR"`. |
| `tlp_green` | Permit `view` for any authenticated user when `resource.tlp == "GREEN"`. |
| `tlp_amber` | Permit `view` when user's org is the owning org or has explicit access grant. |
| `tlp_amber_strict` | Permit `view` when user's org is the owning org. |
| `tlp_red` | Permit `view` when user is in the named recipients list (checked via Attributes). |

### 6.3 Sector Root Ancestry Check

The key complexity is sector root permissions over descendant sectors/orgs. The ancestry_path materialized path enables this:

- Sector root for sector with `ancestry_path = '/gov/'` has authority over any resource where the resource's sector has an ancestry_path starting with `/gov/` (e.g., `/gov/federal/`, `/gov/state/`).
- This is expressed in Cedar as a string prefix check on the `ancestry_path` attribute.

---

## 7. Keycloak Configuration

The `fresnel-realm.json` export file configures:

| Setting | Value |
|---|---|
| Realm name | `fresnel` |
| OIDC client ID | `fresnel-app` |
| Client type | **Public** (no client secret) |
| PKCE | **Required** (`pkce.code.challenge.method: S256`) |
| Valid redirect URIs | `https://localhost/*`, `http://localhost:8080/*` (dev), configurable for prod |
| Web origins | `https://localhost`, `http://localhost:8080` (CORS for keycloak-js) |
| Access token TTL | 10 minutes |
| SSO session idle | 8 hours (effective session length) |
| SSO session max | 8 hours |
| Required actions | Configure OTP (TOTP) |
| Brute force protection | Enabled: lock after 5 failures, wait 30s, exponential backoff |
| Token claims | Standard: `sub`, `email`, `name`, `preferred_username`. No custom claims — app enriches from DB. |
| Direct access grants | Disabled (no password grant for the public client) |

**Dev test users** (created by seed script via Keycloak Admin API):

| Username | Role | Org |
|---|---|---|
| `platform-root` | Platform Root | — |
| `gov-sector-root` | Sector Root (Government) | — |
| `fed-sector-root` | Sector Root (Federal) | — |
| `orgA-root` | Org Root (Org A) | Org A |
| `orgA-admin` | Org Admin (Org A) | Org A |
| `orgA-contributor` | Contributor (Org A) | Org A |
| `orgA-viewer` | Viewer (Org A) | Org A |
| `orgB-root` | Org Root (Org B) | Org B |

Each user gets a predictable TOTP seed (for dev only) so automated tests can generate valid TOTP codes.

---

## 8. Dev Seed Data

The seed script creates a representative hierarchy for development and demos:

```
Platform
├── Sector: Government
│   ├── Subsector: Federal
│   │   ├── Org A: Department of Technology
│   │   └── Org B: National Security Agency
│   └── Subsector: State
│       └── Org C: State IT Authority
├── Sector: Finance
│   ├── Org D: Central Bank
│   └── Org E: Financial Regulatory Authority
└── Sector: Critical Infrastructure
    ├── Subsector: Energy
    │   └── Org F: National Grid Operator
    └── Subsector: Telecommunications
        └── Org G: Telecom Authority
```

Plus sample events (various types, impacts, TLPs, statuses), status reports, campaigns linking events across sectors, and correlations.

---

## 9. Open Implementation Decisions

These are choices to resolve during implementation, not blockers:

1. **Migration library**: `golang-migrate` vs. manual version tracking. Recommendation: manual — fewer dependencies, simple sequential SQL files, version tracked in a `schema_migrations` table.

2. **Structured logging library**: `log/slog` (stdlib, Go 1.21+) vs. `zerolog` vs. `zap`. Recommendation: `log/slog` — no dependency, sufficient for PoC.

3. **Router**: `net/http` (Go 1.22+ with method routing) vs. `chi` vs. `gorilla/mux`. Recommendation: `net/http` — Go 1.22 added `{pattern}` path parameters, sufficient for this API. Zero dependency.

4. **Template engine**: `html/template` (stdlib) vs. `templ`. Recommendation: `html/template` — no dependency. Embed templates via `embed.FS`.

5. **Hot reload for dev**: `air` (popular Go hot reloader) or manual `make run`. Not a dependency — dev tooling only.

6. **Markdown editor fallback**: If Milkdown fails CSP validation, the fallback is a plain `<textarea>` with a "Preview" button that renders server-side. Functional, not pretty, but acceptable for PoC.
