# Fresnel — Implementation Plan, Part 2

**Version**: 0.2
**Date**: 2025-04-09
**Supersedes**: IMPLEMENTATION_PLAN.md (milestones M0–M1)
**Prerequisites**: Working M0 + M1 baseline (see Section 1)

---

## 1. Current State — What Has Been Built

Milestones M0 (Bootstrap) and M1 (Authentication) from the original plan are substantially complete. This section records what exists in the codebase so the remaining milestones can be planned against reality rather than the original spec.

### 1.1 M0: Project Bootstrap — COMPLETE

| ID | Task | Status | Notes |
|---|---|---|---|
| M0.1 | Go module | ✅ | `go 1.23.0`. Dependencies: `pgx/v5`, `google/uuid`, `go-jose/v4`. Cedar-go and starlark not yet added (as planned). |
| M0.2 | Dockerfile | ✅ | Multi-stage: `golang:1.23-alpine` → `alpine:3.21`. Non-root UID 65532, port 8080. |
| M0.3 | Docker Compose | ✅ | `deploy/docker-compose.yml`: PostgreSQL (pgvector/pgvector:pg16), Keycloak 26.0, ClamAV, nginx, Fresnel API. Dev overlay exists (`docker-compose.dev.yml`) with port overrides. |
| M0.4 | Keycloak realm | ✅ | `fresnel-realm.json`: public client `fresnel-app`, PKCE S256, brute-force protection, 10-min access token, 8h SSO session. One user (`platform-admin`). |
| M0.5 | SQL migrations | ✅ | 8 migration files (001–008) cover the **full** application schema: `fresnel` (sectors, orgs, users, events, status_reports, campaigns, correlations, relationships, attachments, TLP recipients, nudge/escalation, formulas), `fresnel_iam` (cedar_policies, role_assignments, root_designations), `fresnel_audit` (audit_entries), pgvector extension + event_embeddings. Dev seed data creates a single sector/org/user. |
| M0.6 | Migration runner | ✅ | Embedded SQL via `embed.FS`. Tracked in `public.schema_migrations`. Runs on startup and via `./cmd/fresnel migrate`. |
| M0.7 | Config loader | ✅ | Env-based: `DATABASE_URL`, `KEYCLOAK_ISSUER`, `KEYCLOAK_CLIENT_ID`, `KEYCLOAK_EXTERNAL_URL`, `APP_PUBLIC_URL`, `CLAMAV_SOCKET`, `SMTP_HOST/PORT`, `ATTACHMENT_DIR`, `DASHBOARD_CACHE_TTL_SEC`. |
| M0.8 | Health endpoint | ✅ | `GET /api/v1/health` — pings DB and Keycloak OIDC discovery. |
| M0.9 | Nginx config | ✅ | TLS termination (self-signed for dev), proxy_pass to Fresnel, security headers (CSP, HSTS, X-Frame-Options, etc.), `limit_req` rate limiting. |
| M0.10 | Makefile | ✅ | Targets: `build`, `test`, `lint`, `certs`, `compose-up`, `compose-down`, `migrate`, `seed`, `run`. |

### 1.2 M1: Authentication — COMPLETE (with noted gaps)

| ID | Task | Status | Notes |
|---|---|---|---|
| M1.1 | App shell page | ✅ | `GET /{path...}` catch-all serves `base.html` with Keycloak `<meta>` tags. |
| M1.2 | `app.js` | ✅ | keycloak-js PKCE init, Bearer token on HTMX requests, 30s refresh interval, 401 retry, initial HTMX content load, user info + logout. `keycloak.min.js` vendored locally (not loaded from Keycloak instance). |
| M1.3 | Bearer token validation | ✅ | `middleware.OIDC`: JWKS-cached JWT validation, issuer allowlist (internal + external URLs), expiry check. API routes → 401 JSON; non-API routes pass through. |
| M1.4 | AuthContext builder | ✅ | `postgres.LoadAuthContext`: resolves by `keycloak_sub`, falls back to email linking on first login. Loads org memberships, IAM roles, root designations in a single transaction. |
| M1.5 | Keycloak test users | ⚠️ | Only `platform-admin` in `fresnel-realm.json`. Dev seed SQL has a `dev@fresnel.local` user and a `platform-admin` user, but **no root_designations or role_assignments** are seeded. The full set of 8 test users from the plan was not created. |
| M1.6 | Content negotiation | ✅ | `Accept` header → `RenderKind` (HTML or JSON) on context. |
| M1.7 | Request logging | ✅ | Structured JSON: method, path, status, latency, user_id when present. |
| M1.8 | Base HTML shell | ✅ | `layouts/base.html`: nav with user-info slot, `#app` main, keycloak meta, CSS + JS script tags. |
| M1.9 | Wire router | ✅ | Middleware chain: logging → OIDC → content negotiation → mux. Routes: health (public), static (public), dashboard (authenticated), catch-all shell (public). |

### 1.3 Deviations from Original Plan

1. **No TOTP enforcement.** The realm JSON does not set `requiredActions: ["CONFIGURE_TOTP"]`. TOTP should be added when test users are created, or deferred to M8 hardening.
2. **Database schema is complete.** The original plan had schema in M0.5 covering only the initial tables. In practice, all tables for the full domain (events, status reports, campaigns, correlations, nudge, formulas, etc.) were created up front. This is a positive deviation — no additional migration work is needed for M2–M5.
3. **Only 2 Fresnel users seeded.** The plan called for 8 test users across all roles. Only a dev user and platform-admin exist (with no IAM records).
4. **Makefile `run` exports `HMAC_SECRET`** but `config.go` does not reference it. This is a leftover — there is no CSRF/HMAC layer (by design, since we use Bearer tokens).
5. **Dashboard handler returns a stub.** `GET /api/v1/dashboard` renders user info but no hierarchy or status data.
6. **No test files exist.** Zero `*_test.go` files in the project.

### 1.4 Existing Package Layout

```
cmd/fresnel/main.go              # Entry point: config, migrate, serve
internal/
  config/config.go                # Env config struct
  domain/auth.go                  # AuthContext, ScopeEntry, RoleAssignment
  httpserver/
    server.go                     # http.Server wrapper
    router.go                     # Route registration + middleware chain
    handlers/
      auth.go                     # Shell handler
      dashboard.go                # Dashboard stub handler
      health.go                   # Health check
      page.go                     # Template data structs
    middleware/
      auth.go                     # OIDC Bearer validation + AuthContext
      content_neg.go              # Accept header content negotiation
      logging.go                  # Request logging
    requestctx/
      ctx.go                      # Context getters/setters
      keys.go                     # Context keys, RenderKind enum
    templates/
      fs.go                       # Template parsing
  oauth/
    jwks.go                       # JWKS cache
    token.go                      # JWT verification
  storage/postgres/
    postgres.go                   # Pool + migration runner
    auth.go                       # LoadAuthContext
```

**Not yet created (from original plan):**
- `internal/domain/` — event.go, statusreport.go, campaign.go, correlation.go, organization.go, sector.go, user.go, audit.go, enums.go
- `internal/storage/interfaces.go`
- `internal/storage/postgres/` — event.go, statusreport.go, campaign.go, correlation.go, sector.go, organization.go, user.go, audit.go, policy.go, nudge.go, formula.go
- `internal/service/` — all service files
- `internal/cedar/` — evaluator, resource mapper, loader, policies
- `internal/starlark/` — formula engine
- `internal/markdown/` — goldmark+bluemonday pipeline
- `internal/mail/` — SMTP client
- `internal/http/handlers/` — all domain handlers
- `templates/` — all domain templates beyond base shell + dashboard stub
- Tests — everything

---

## 2. Remaining Milestones

The original plan's M2–M8 structure is preserved with revisions based on what we've learned. The critical path is unchanged:

```
M2 (authz + hierarchy) → M3 (events) → M4 (reports/campaigns) → M5 (dashboard)
                                                                       ↓
                                                        M7 (UI polish) → M8 (hardening)
                                   M3 ──→ M6 (nudge, parallel with M4/M5)
```

A new **M1.5-fix** mini-milestone addresses gaps from M1 before proceeding to M2.

---

### M1-fix: Authentication Gaps

**Goal**: Resolve M1 gaps so that M2 has a solid foundation to test against.

| ID | Task | Details |
|---|---|---|
| M1F.1 | Seed full test user set in Keycloak | Add all 8 users from the original plan (platform-root, gov-sector-root, fed-sector-root, orgA-root, orgA-admin, orgA-contributor, orgA-viewer, orgB-root) to `fresnel-realm.json` or via a seed script using the Keycloak Admin API. Each with a known password. |
| M1F.2 | Seed matching Fresnel users + IAM records | Write a migration or seed script that inserts: the dev hierarchy (Government → Federal → Org A + Org B, etc.), matching `fresnel.users` rows, `user_org_memberships`, `fresnel_iam.role_assignments`, and `fresnel_iam.root_designations` for each test user. This is the hierarchy from Section 8 of the original plan. |
| M1F.3 | Verify end-to-end login for each test user | Manual or scripted: each Keycloak user logs in → keycloak-js completes PKCE → HTMX loads dashboard → AuthContext shows correct roles/root flags. Document any issues. |
| M1F.4 | Clean up Makefile | Remove `HMAC_SECRET` from `make run`. Add `KEYCLOAK_CLIENT_ID` default (`fresnel-app`). |

**Acceptance**: All 8 test users can log in. `AuthContext` correctly reflects their roles and root scope. The dashboard stub shows user identity, org context, and root status.

---

### M2: Authorization (Cedar) + Organizational Hierarchy

**Goal**: Cedar evaluates policies correctly. Sector/org/user CRUD works with proper access control. Audit logging captures all mutations.

**Depends on**: M1-fix

**Schema note**: All IAM tables (`cedar_policies`, `role_assignments`, `root_designations`) already exist in migration 002. All domain tables already exist. No new migrations needed unless Cedar requires schema changes.

| ID | Task | Details |
|---|---|---|
| M2.1 | Add cedar-go dependency | `go get github.com/cedar-policy/cedar-go`. Validate it compiles and runs a trivial policy evaluation. If cedar-go is not mature enough, evaluate alternative: hand-written Go evaluator that interprets a subset of Cedar semantics sufficient for our role matrix (see fallback note below). |
| M2.2 | Define Cedar entity schema | Write `.cedarschema` for entity types: `Fresnel::User`, `Fresnel::Event`, `Fresnel::StatusReport`, `Fresnel::Campaign`, `Fresnel::Organization`, `Fresnel::Sector`, `Fresnel::RoleAssignment`. Per Section 6 of original plan. |
| M2.3 | Write Cedar policy templates | One file per role + TLP conditions. Embed via `embed.FS` in `internal/cedar/policies/`. |
| M2.4 | Implement CedarEvaluator | `IsPermitted(auth, action, resource) bool` and `FilterPermitted(auth, action, resources) []Resource`. Loads policies from DB on startup, caches in memory, re-reads on signal. |
| M2.5 | Implement Cedar Gate middleware | Maps HTTP method + route pattern → Cedar action + resource type. Calls CedarEvaluator. Returns 403 on deny. Wire into the middleware chain after OIDC. |
| M2.6 | Implement domain→CedarResource mapping | Convert domain types to Cedar entities for evaluation. Includes TLP:RED recipient lookup. |
| M2.7 | Implement domain types for hierarchy | `internal/domain/`: `sector.go`, `organization.go`, `user.go`. Validation methods. |
| M2.8 | Implement storage interfaces | `internal/storage/interfaces.go`: `SectorStore`, `OrganizationStore`, `UserStore`, `PolicyStore`, `AuditStore`. |
| M2.9 | Implement SectorStore (Postgres) | CRUD for sectors. Ancestry path management: compute `ancestry_path` from parent on create. Depth validation (max 5). Children listing. |
| M2.10 | Implement OrganizationStore (Postgres) | CRUD for organizations. Sector membership. Timezone field. |
| M2.11 | Implement UserStore (Postgres) | CRUD for users. Primary org, multi-org membership. Keycloak sub linking. |
| M2.12 | Implement PolicyStore (Postgres) | CRUD for `cedar_policies`, `role_assignments`, `root_designations`. |
| M2.13 | Implement AuditStore (Postgres) | Append-only insert into `fresnel_audit.audit_entries`. The DB role should have no UPDATE/DELETE on this table. |
| M2.14 | Implement service layer for hierarchy | `internal/service/`: `sector.go`, `organization.go`, `user.go`, `audit.go`. Each wraps store operations with Cedar checks and audit logging. |
| M2.15 | Implement root designation management | Assign/reassign root for scopes. Validate: only self-reassignment or parent root can reassign. Audit as HIGH severity. |
| M2.16 | Implement role assignment management | Assign/revoke roles within scopes. Validate scope authority via Cedar. |
| M2.17 | Implement sector HTTP handlers | `GET /api/v1/sectors`, `GET /api/v1/sectors/{id}`, `GET /api/v1/sectors/{id}/children`. Dual-representation (HTML fragment + JSON). |
| M2.18 | Implement organization HTTP handlers | `GET /api/v1/orgs`, `GET /api/v1/orgs/{id}`, org member management endpoints. |
| M2.19 | Implement user HTTP handlers | `GET /api/v1/users`, `GET /api/v1/users/{id}`, `GET /api/v1/users/me`, `PUT /api/v1/users/{id}`. |
| M2.20 | Implement audit HTTP handler | `GET /api/v1/audit` with scope-based filtering. Cedar-gated. |
| M2.21 | Implement admin UI templates | Basic HTML templates for sector management, org management, user management, role assignment. Minimal styling — functional for testing. |
| M2.22 | Write Cedar policy test suite | Test matrix: each role × each action × each resource scope. Verify permit/deny matches the requirements role table. Test TLP conditions. **This is a critical quality gate.** |
| M2.23 | Write storage layer tests | Test each store against real Postgres (Docker). Pagination, transactions, edge cases. |
| M2.24 | Seed initial Cedar policies | Migration or startup logic: render Cedar policy templates for the default hierarchy and insert into `cedar_policies`. |

**Cedar-go fallback**: If `cedar-go` proves too immature (API instability, missing features for string prefix checks, etc.), implement a simplified Go-native evaluator. The policy logic is well-defined — 8 roles with clear rules. A Go evaluator that reads policy definitions from the same DB table and evaluates them in Go would be acceptable for PoC, with Cedar proper as a future upgrade. The important thing is that the `IsPermitted` / `FilterPermitted` interface exists so callers don't know or care about the implementation.

**Acceptance**: Platform root can CRUD sectors (nested), orgs, users, and assign roles. A contributor logged in sees only what their role allows. Audit log captures all mutations. Cedar test suite passes with full coverage.

---

### M3: Events (Core Domain)

**Goal**: Full event lifecycle with authorization, revision history, updates, attachments, and TLP enforcement.

**Depends on**: M2

| ID | Task | Details |
|---|---|---|
| M3.1 | Implement domain types | `internal/domain/event.go`: `Event`, `EventRevision`, `EventUpdate`, `Attachment`. `enums.go`: TLP, Impact, Status, EventType enums with validation. Status lifecycle: OPEN → INVESTIGATING → MITIGATING → RESOLVED → CLOSED. |
| M3.2 | Implement EventStore | Full CRUD with revision tracking. On update: insert `event_revision`, bump revision number. Filters: sector_context, organization_id, status, impact, event_type, date range, text search. Pagination + sorting. |
| M3.3 | Implement EventUpdateStore | Append-only within an event. Impact/status changes propagate to parent event (in a transaction). |
| M3.4 | Implement EventService | CRUD with Cedar row-level filtering. Business rules: TLP on updates ≥ parent event TLP. Status transitions validated. Sector context immutable after creation. Audit logging. |
| M3.5 | Implement AttachmentStore | Metadata in Postgres, bytes on filesystem (`ATTACHMENT_DIR`). |
| M3.6 | Implement AttachmentService | Upload: validate type + size → temp write → ClamAV scan → if clean, move to permanent + insert metadata. If quarantined: audit + reject. Max 10 per event. Download: authenticated + Cedar-checked. |
| M3.7 | Implement ClamAV client | `internal/clamav/client.go`: Unix socket to clamd. Scan, parse OK/FOUND. Handle timeouts gracefully — reject upload on scan failure (don't allow unscanned files). |
| M3.8 | Implement Markdown rendering pipeline | `internal/markdown/render.go`: goldmark → HTML, bluemonday sanitization. `Render(input) template.HTML`. Test with adversarial XSS vectors. |
| M3.9 | Implement TLP:RED recipient management | When creating/editing TLP:RED resources, require recipients. Store in `tlp_red_recipients`. CedarResource includes recipients for policy evaluation. |
| M3.10 | Implement event HTTP handlers | All event CRUD endpoints. Sub-resources: updates, attachments. Dual-representation. |
| M3.11 | Implement event HTML templates | List (filterable, sortable, paginated), detail (update log, revision history), create/edit form (sector context selector, TLP/impact/status dropdowns). |
| M3.12 | Write event tests | Domain validation tests, store tests (real Postgres), service tests (mocked stores), handler tests (httptest). Markdown/XSS test suite. |

**Acceptance**: Events can be created, edited, updated. TLP visibility enforced. Revision history tracks changes. Attachments upload, scan, download. ClamAV quarantine works. TLP:RED only visible to named recipients.

---

### M4: Status Reports, Campaigns, Correlations

**Goal**: All remaining domain objects with full CRUD and authorization.

**Depends on**: M3

| ID | Task | Details |
|---|---|---|
| M4.1 | Implement StatusReport domain + store + service | CRUD with revision tracking. Scope validation (org-scoped vs sector-scoped). Cedar row-level filtering. Audit. |
| M4.2 | Implement status report HTTP handlers + templates | All endpoints. List, detail (referenced events clickable), create/edit form. |
| M4.3 | Implement Campaign domain + store + service | CRUD for campaigns + event linking. Cedar filtering. Restricted-content indicator for events the user can't see. |
| M4.4 | Implement campaign HTTP handlers + templates | All endpoints. Detail shows events grouped by org/sector. |
| M4.5 | Implement Correlation store + service | CRUD with canonical ordering. Visibility: user must access both linked events. Suggested correlations only visible to the event's org until confirmed. |
| M4.6 | Implement EventRelationship store + service | Directional relationships. Visibility: see both events (except sanitized_version). |
| M4.7 | Implement correlation/relationship handlers | Sub-resource endpoints under events. |
| M4.8 | Write M4 tests | Store, service, handler tests for status reports, campaigns, correlations. |

**Acceptance**: Status reports at org and sector scope with revision history. Campaigns link events cross-sector with restricted-content indicator. Correlations respect bidirectional visibility.

---

### M5: Dashboard + Starlark

**Goal**: Hierarchical dashboard with computed statuses, timeline side panel, campaign section.

**Depends on**: M4

| ID | Task | Details |
|---|---|---|
| M5.1 | Add go-starlark dependency | `go get go.starlark.net`. Verify compilation. |
| M5.2 | Implement FormulaEngine | `internal/starlark/engine.go`: Starlark execution with 100ms timeout, 1MB memory. Parse, execute, validate. Fallback to default formula on error. |
| M5.3 | Write default formula | Weighted average: NORMAL=0, DEGRADED=1, IMPAIRED=2, CRITICAL=3. Threshold: <0.5→NORMAL, <1.5→DEGRADED, <2.5→IMPAIRED, ≥2.5→CRITICAL. |
| M5.4 | Implement FormulaStore | CRUD for `status_formulas`. Per-node (sector or platform). |
| M5.5 | Implement DashboardService | Build hierarchical status tree. For each org: latest status report's assessed_status. For each parent: evaluate Starlark formula. Cedar-filter: restricted nodes show as UNKNOWN. Cache with configurable TTL (default 60s). |
| M5.6 | Implement status cache | In-memory, TTL-based. Invalidation on: status report create/update, event status change, formula change. |
| M5.7 | Replace dashboard stub handler | `GET /api/v1/dashboard` returns full hierarchy tree (JSON + HTML). |
| M5.8 | Implement timeline handler | `GET /api/v1/dashboard/{node_type}/{id}/timeline`: interleaved reports + events, paginated, Cedar-filtered. |
| M5.9 | Implement dashboard HTML template | Collapsible tree (HTMX expand/collapse), status badges, campaign section, click → timeline in side panel, 60s auto-refresh. |
| M5.10 | Implement formula management | `GET/PUT /api/v1/sectors/{id}/formula`. Admin UI: Starlark editor, validate + preview. |
| M5.11 | Write M5 tests | Starlark engine tests (various inputs, timeout, malformed). Dashboard service tests. Cache invalidation tests. |

**Acceptance**: Dashboard loads <100ms. Status tree with computed statuses. Timeline on click. Custom Starlark formulas. Cache invalidation works within TTL.

---

### M6: Nudge System

**Goal**: Email nudges on schedule based on event impact. Escalation walks the hierarchy.

**Depends on**: M3 + M2 (can run in parallel with M4/M5)

| ID | Task | Details |
|---|---|---|
| M6.1 | Implement Mailer | `internal/mail/smtp.go`: `Send(ctx, to, subject, body)`. TLS required. Timeout handling. Structured logging. |
| M6.2 | Implement NudgeStore | CRUD for `nudge_log` + `escalation_state`. Idempotency queries. |
| M6.3 | Implement NudgeScheduler | Goroutine, 15-min tick. EOB calculation (user tz → org tz → platform default). Impact × frequency rules. Idempotent sends. |
| M6.4 | Implement escalation logic | Advance `escalation_state.current_level` when no response for >1 business day. Walk sector ancestry: contributors → org root → sector roots → platform root. |
| M6.5 | Implement weekly digest | Monday aggregation of open events with impact > INFO. Per-user, scope-filtered. |
| M6.6 | Implement escalation reset | Event update → reset current_level to 0, update last_response_at. |
| M6.7 | Email templates | Plain-text: daily nudge, weekly nudge, escalation, digest. Event title, impact, link. |
| M6.8 | Write M6 tests | Scheduler logic tests (time mocking). Escalation level advancement. Duplicate prevention. |

**Acceptance**: CRITICAL event → nudge at EOB. No duplicates. Escalation advances after 1 business day. Event update resets escalation. Weekly digest on schedule.

---

### M7: UI Polish + Markdown Editor

**Goal**: Functional, presentable UI for stakeholder demos. Markdown editor works under CSP.

**Depends on**: M5 (dashboard), M3–M4 (domain templates)

| ID | Task | Details |
|---|---|---|
| M7.1 | Evaluate Markdown editor for CSP | Test Milkdown under CSP (no `unsafe-eval`). If it fails, fall back to plain textarea + server-side preview. This decision should be made early — if Milkdown works, vendor it; if not, skip it. |
| M7.2 | Integrate Markdown editor in forms | Event/status-report create/edit: editor produces Markdown source, stored in DB. Preview renders via goldmark+bluemonday. |
| M7.3 | Style the application | `static/css/fresnel.css` overhaul: dashboard tree (status colors, indent, expand/collapse), forms, detail pages, timeline, nav. The existing dark theme is a starting point. Responsive for desktop/tablet. |
| M7.4 | Implement org context selector | Multi-org users: nav bar dropdown. Sets `X-Fresnel-Org` header via JS. Affects event creation org context and dashboard scope. |
| M7.5 | Implement sector context selector | For multi-sector org users: prominent selector with confirmation (immutable after creation). |
| M7.6 | Implement TLP/Impact/Status badges | Consistent visual badges: TLP colors (RED/AMBER/GREEN/CLEAR), impact dots, status pills. Used everywhere. |
| M7.7 | Implement revision history diff view | Expandable revision history on detail pages. Field-by-field diff. |
| M7.8 | Error pages | 403, 404, 500 — styled consistently. |
| M7.9 | Adversarial Markdown testing | Feed XSS vectors through the pipeline. Verify bluemonday strips everything. |

**Acceptance**: Dashboard is visually clear and demo-ready. Markdown editor or fallback works. Badges consistent. No XSS.

---

### M8: Security Hardening + Deployment

**Goal**: Production-ready nginx, ModSecurity, backup, deployment guide.

**Depends on**: M7

| ID | Task | Details |
|---|---|---|
| M8.1 | Configure ModSecurity + OWASP CRS | Install in nginx. Test HTMX POST/PUT + file uploads. Tune false positives. |
| M8.2 | Configure fail2ban | Monitor nginx logs. Ban after 10 failed auth attempts in 5 min. |
| M8.3 | Verify all security headers | Automated test: fetch every endpoint, verify headers from Requirements 9.2. |
| M8.4 | Rate limiting review | Verify nginx `limit_req` zones: global, login, API. Adjust thresholds based on actual usage patterns. |
| M8.5 | Input validation audit | Review all handler input parsing. Max lengths, enum validation, UUID format, Markdown size limits. Parameterized queries (pgx). |
| M8.6 | TOTP enforcement | Add required TOTP to Keycloak realm. Update test users. |
| M8.7 | Finalize Docker Compose for production | No debug ports, resource limits, restart policies, healthchecks, log rotation. |
| M8.8 | Backup scripts | `backup.sh`: pg_dump, Keycloak realm export, 30-day retention. Cron entry. |
| M8.9 | Deployment guide | Fresh VM (Ubuntu 24.04, LUKS, Docker): compose up, first platform root, restore from backup. |
| M8.10 | Federation/webhook stubs | All federation endpoints return 501 `{"error":"not_implemented","message":"Federation is planned for Phase 2"}`. Break-glass stub. |
| M8.11 | Final integration test | Clean deployment, full hierarchy, all roles, events/reports/campaigns, dashboard, nudge, audit. |

**Acceptance**: Clean deploy on fresh VM. Security headers present. ModSecurity blocks test XSS. Rate limiting works. Backup + restore cycle. Federation stubs return 501.

---

## 3. Revised Project Structure

Updates to the original plan's project structure, reflecting reality and remaining work.

```
fresnel/
├── cmd/fresnel/main.go                     # ✅ Exists
├── internal/
│   ├── config/config.go                    # ✅ Exists
│   ├── domain/
│   │   ├── auth.go                         # ✅ Exists
│   │   ├── event.go                        # M3.1
│   │   ├── statusreport.go                 # M4.1
│   │   ├── campaign.go                     # M4.3
│   │   ├── correlation.go                  # M4.5
│   │   ├── organization.go                 # M2.7
│   │   ├── sector.go                       # M2.7
│   │   ├── user.go                         # M2.7
│   │   ├── audit.go                        # M2.13
│   │   └── enums.go                        # M3.1
│   ├── storage/
│   │   ├── interfaces.go                   # M2.8
│   │   └── postgres/
│   │       ├── postgres.go                 # ✅ Exists (pool + migrator)
│   │       ├── auth.go                     # ✅ Exists (LoadAuthContext)
│   │       ├── sector.go                   # M2.9
│   │       ├── organization.go             # M2.10
│   │       ├── user.go                     # M2.11
│   │       ├── policy.go                   # M2.12
│   │       ├── audit.go                    # M2.13
│   │       ├── event.go                    # M3.2
│   │       ├── event_update.go             # M3.3
│   │       ├── attachment.go               # M3.5
│   │       ├── statusreport.go             # M4.1
│   │       ├── campaign.go                 # M4.3
│   │       ├── correlation.go              # M4.5
│   │       ├── nudge.go                    # M6.2
│   │       └── formula.go                  # M5.4
│   ├── service/
│   │   ├── sector.go                       # M2.14
│   │   ├── organization.go                 # M2.14
│   │   ├── user.go                         # M2.14
│   │   ├── audit.go                        # M2.14
│   │   ├── event.go                        # M3.4
│   │   ├── attachment.go                   # M3.6
│   │   ├── statusreport.go                 # M4.1
│   │   ├── campaign.go                     # M4.3
│   │   ├── correlation.go                  # M4.5
│   │   ├── dashboard.go                    # M5.5
│   │   └── nudge.go                        # M6.3
│   ├── cedar/
│   │   ├── evaluator.go                    # M2.4
│   │   ├── resource.go                     # M2.6
│   │   ├── loader.go                       # M2.4
│   │   └── policies/                       # M2.3
│   │       └── *.cedar                     # Embedded policy templates
│   ├── starlark/
│   │   ├── engine.go                       # M5.2
│   │   └── default_formula.star            # M5.3
│   ├── clamav/
│   │   └── client.go                       # M3.7
│   ├── markdown/
│   │   └── render.go                       # M3.8
│   ├── mail/
│   │   └── smtp.go                         # M6.1
│   ├── oauth/
│   │   ├── jwks.go                         # ✅ Exists
│   │   └── token.go                        # ✅ Exists
│   └── httpserver/
│       ├── server.go                       # ✅ Exists
│       ├── router.go                       # ✅ Exists (will grow)
│       ├── middleware/
│       │   ├── auth.go                     # ✅ Exists
│       │   ├── cedar_gate.go               # M2.5
│       │   ├── content_neg.go              # ✅ Exists
│       │   └── logging.go                  # ✅ Exists
│       ├── handlers/
│       │   ├── auth.go                     # ✅ Exists (shell)
│       │   ├── health.go                   # ✅ Exists
│       │   ├── dashboard.go                # ✅ Exists (stub → M5.7)
│       │   ├── page.go                     # ✅ Exists
│       │   ├── event.go                    # M3.10
│       │   ├── statusreport.go             # M4.2
│       │   ├── campaign.go                 # M4.4
│       │   ├── sector.go                   # M2.17
│       │   ├── organization.go             # M2.18
│       │   ├── user.go                     # M2.19
│       │   ├── audit.go                    # M2.20
│       │   ├── attachment.go               # M3.10
│       │   └── federation.go               # M8.10 (stubs)
│       ├── requestctx/
│       │   ├── ctx.go                      # ✅ Exists
│       │   └── keys.go                     # ✅ Exists
│       └── templates/
│           └── fs.go                       # ✅ Exists (will grow)
├── migrations/                             # ✅ 001–008 exist (schema complete)
├── templates/                              # ✅ base.html + dashboard stub exist
│   ├── layouts/base.html                   # ✅
│   ├── dashboard/index.html                # ✅ (stub → M5.9)
│   ├── events/                             # M3.11
│   ├── status_reports/                     # M4.2
│   ├── campaigns/                          # M4.4
│   ├── admin/                              # M2.21
│   ├── audit/                              # M2.20
│   ├── partials/                           # M7.6
│   └── errors/                             # M7.8
├── static/
│   ├── app.js                              # ✅ Exists
│   ├── htmx.min.js                         # ✅ Vendored 2.0.4
│   ├── keycloak.min.js                     # ✅ Vendored
│   ├── css/fresnel.css                     # ✅ Exists (minimal → M7.3)
│   └── embed.go                            # ✅ Exists
├── deploy/                                 # ✅ Complete for dev
├── docs/                                   # ✅
├── go.mod                                  # ✅
├── Dockerfile                              # ✅
└── Makefile                                # ✅
```

---

## 4. Testing Strategy (Unchanged, but Noting Current State)

**Current test count: 0.** No `*_test.go` files exist. Testing should be introduced with M2 and maintained throughout.

| Layer | Tool | When |
|---|---|---|
| Domain types | `go test` | Start in M2.7, M3.1 |
| Storage layer | `go test` + Docker Postgres | Start in M2.9 |
| Cedar policies | `go test` | M2.22 (critical gate) |
| Service layer | `go test` + mocks | Start in M2.14 |
| HTTP handlers | `go test` + httptest | Start in M2.17 |
| Integration | `go test` + full stack | M8.11 |
| Markdown/XSS | `go test` | M3.8 |
| Starlark | `go test` | M5.2 |

Test database approach: each test suite uses a fresh schema or transaction rollback. The Docker Compose Postgres instance runs alongside dev.

---

## 5. Recommended Build Order (Per-Milestone Work Breakdown)

For each milestone, the recommended order within the milestone:

**M2**: domain types (M2.7) → interfaces (M2.8) → stores (M2.9–M2.13) → store tests (M2.23) → Cedar evaluator (M2.1–M2.6) → Cedar tests (M2.22) → services (M2.14–M2.16) → handlers (M2.17–M2.20) → admin templates (M2.21) → seed policies (M2.24)

**M3**: domain types + enums (M3.1) → Markdown pipeline (M3.8) → stores (M3.2–M3.5) → ClamAV client (M3.7) → services (M3.4, M3.6) → TLP:RED (M3.9) → handlers (M3.10) → templates (M3.11) → tests (M3.12)

**M4**: StatusReport (M4.1–M4.2) → Campaign (M4.3–M4.4) → Correlation + Relationship (M4.5–M4.7) → tests (M4.8)

**M5**: Starlark engine (M5.1–M5.3) → FormulaStore (M5.4) → DashboardService + cache (M5.5–M5.6) → replace dashboard handler (M5.7) → timeline (M5.8) → templates (M5.9–M5.10) → tests (M5.11)

**M6** (parallel with M4/M5): Mailer (M6.1) → NudgeStore (M6.2) → Scheduler (M6.3) → Escalation (M6.4) → Reset (M6.6) → Digest (M6.5) → Email templates (M6.7) → tests (M6.8)

---

## 6. Open Decisions (Updated)

| # | Decision | Original Recommendation | Current Status |
|---|---|---|---|
| 1 | Migration library | Manual | ✅ Implemented: manual with `schema_migrations` |
| 2 | Structured logging | `log/slog` | ✅ Implemented: `slog` with JSON handler |
| 3 | Router | `net/http` (Go 1.22+) | ✅ Implemented: `http.ServeMux` with path params |
| 4 | Template engine | `html/template` | ✅ Implemented: `html/template` with `embed.FS` |
| 5 | Hot reload | `air` or manual | Not yet decided — manual `make run` works for now |
| 6 | Markdown editor | Milkdown or fallback | Not yet evaluated — decision needed in M7.1 |
| 7 | **Cedar-go maturity** | Use cedar-go | **NEW**: Evaluate cedar-go vs. hand-written Go evaluator. See M2.1 fallback note. |
| 8 | **Test DB strategy** | Fresh schema per suite | **NEW**: Decide between per-suite schema creation vs. transaction rollback. Transaction rollback is faster but less isolated. |
