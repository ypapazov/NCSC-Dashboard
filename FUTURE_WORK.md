# Fresnel — Future Work

Planned initiatives that are architecturally prepared but not yet implemented. Each item has schema, interface boundaries, or stub endpoints in the codebase — they are designed to be a short distance from implementation.

---

## Cedar Policy Engine (Authorization)

**Current state**: Authorization uses a Go-native role matrix (`internal/authz/cedar.go`) — a `switch` statement encoding the role x action x TLP permission table. The `Authorizer` interface is clean and all services call it; no service depends on the implementation directly.

**Target**: Replace the Go switch-matrix with real [Cedar](https://www.cedarpolicy.com/) policy files evaluated by `cedar-go`. Policies become separate, auditable `.cedar` artifacts. Policy changes no longer require recompilation.

**What exists**:
- `Authorizer` interface with `Authorize()` and `FilterAuthorized[T]()` — callers won't change.
- `fresnel_iam.cedar_policies` table in PostgreSQL (created by migration 002, currently empty).
- `fresnel_iam.role_assignments` and `root_designations` tables (populated and used).
- Detailed proposal: [`docs/TODO/CEDAR_REAL_POLICIES.md`](docs/TODO/CEDAR_REAL_POLICIES.md).

**Effort estimate**: 3–4 days.

---

## Starlark Formula Engine (Dashboard Aggregation)

**Current state**: Dashboard status is computed by a hardcoded weighted-average function in `internal/service/dashboard.go`. The `FormulaService` exists but `Set()` and `Validate()` return errors ("not available in this PoC build").

**Target**: Embed the [Starlark](https://github.com/google/starlark-go) interpreter so sector/platform administrators can write custom aggregation formulas (e.g., worst-case, majority, weighted with custom weights). Formulas are sandboxed (no I/O, 100ms timeout, 1MB memory limit).

**What exists**:
- `status_formulas` table in PostgreSQL (migration 001).
- `FormulaStore` with full CRUD.
- `FormulaService` with stubbed `Set`/`Validate`.
- Dashboard service has a single `computeStatus()` call site — swap point for the engine.
- UI shows formulas as "coming soon" (disabled controls).

**Effort estimate**: 2–3 days (engine + validation + management UI).

---

## AI Correlation Module (Ollama + pgvector)

**Current state**: No AI pipeline exists in the running application.

**Target**: Two agent types:
1. **Agent 1 (Privileged Analyst)** — system-level read access to all events. Generates embedding vectors, detects cross-organizational correlations, produces suggested correlation links that analysts confirm. Runs on a schedule or event trigger.
2. **Agent 2 (User-Scoped Assistant)** — delegated user credentials. Assists with drafting, searching, summarizing within the user's access scope.

**What exists**:
- `event_embeddings` table with `vector(768)` column and IVFFLAT index (migration 004).
- `pgvector` extension installed in PostgreSQL.
- `correlations` table with `correlation_type` supporting `SUGGESTED` (for AI-generated links).
- `SystemAgent` principal type referenced in requirements (not yet in Cedar schema).
- `static/cytoscape.min.js` vendored for correlation graph visualization.
- Docker Compose can include an Ollama container under an `ai` profile.

**Effort estimate**: Research phase for model selection; 1–2 weeks for embedding pipeline + correlation logic.

---

## Federation

**Current state**: All federation endpoints return `501 Not Implemented`.

**Target**: Allow organizations to run their own Fresnel instances and participate in the broader platform ecosystem. Federated instances publish events/reports to the hub selectively, with org-controlled sanitization.

**What exists**:
- `source_instance` field on all domain objects (default: `"local"`).
- All resource IDs are globally unique UUIDs.
- Federation API endpoints registered in the router (`/api/v1/federation/*`), returning 501.
- `handlers/federation.go` — stub implementation.
- Protocol requirements documented in `REQUIREMENTS.md` §2.5 (mTLS, signed payloads, async request-response).

**Effort estimate**: Significant — protocol design + implementation + multi-instance testing.

---

## Sovereign Mode

**Current state**: All participating organizations accept hierarchical visibility (root users see everything in their scope).

**Target**: An organization activates sovereign mode to block data-plane access from all external principals, including parent roots. External users see "restricted" status on the dashboard instead of actual data.

**What exists**:
- Cedar authorization architecture supports `forbid` policies that override `permit`.
- Dashboard rendering already handles `UNKNOWN`/"restricted" display for nodes the user can't see.
- TLP enforcement infrastructure provides the per-object access control foundation.

**Effort estimate**: Primarily a Cedar policy change (once real Cedar is implemented) + UI for conflict surfacing.

---

## Break-Glass Procedure

**Current state**: API endpoint `/api/v1/orgs/{id}/break-glass` is registered and returns 501.

**Target**: Allow a parent root to override sovereign mode when an org is non-responsive (no reply within configurable window, default 48h). Includes cooling-off period, time-limited access, mandatory audit trail, and abuse prevention.

**What exists**:
- Audit entry severity `CRITICAL` and action type `BREAK_GLASS` defined.
- `Mailer` interface (same as nudge system) available for break-glass notifications.
- Detailed procedure documented in `REQUIREMENTS.md` §2.6.

**Effort estimate**: 2–3 days.

---

## Swimlane View

**Current state**: Dashboard shows only the hierarchical tree view.

**Target**: A second dashboard view showing sectors/organizations as horizontal lanes with event cards scrolling horizontally. Prioritizes scanning recent activity across many orgs simultaneously.

**What exists**:
- Detailed specification in [`docs/TODO/UI_UPGRADE.md`](docs/TODO/UI_UPGRADE.md).
- Same backend data as the tree view — purely a rendering concern.
- Filter bar and side panel patterns shared with the tree view.

**Effort estimate**: 3–4 days (HTMX + CSS, no new JS).

---

## Correlation Graph (Cytoscape.js)

**Current state**: `static/cytoscape.min.js` is vendored. Graph page and basic handler exist. No dedicated graph API endpoints.

**Target**: Interactive node-link diagram for event correlations. Nodes are events (shape = type, color = impact, opacity = status). Edges are correlations and relationships. Zoom, pan, drag, filter, expand neighborhood.

**What exists**:
- Cytoscape.js vendored in static assets.
- Graph templ component and route registered.
- Specification in [`docs/TODO/UI_UPGRADE.md`](docs/TODO/UI_UPGRADE.md) §4.
- Requires new API endpoints: `GET /api/v1/events/{id}/correlations?format=graph`, `GET /api/v1/campaigns/{id}/graph`, `GET /api/v1/graph/explore`.

**Effort estimate**: 3–5 days.

---

## Hierarchical Tree Redesign

**Current state**: Dashboard tree is an indented list with status badges.

**Target**: Visual top-down hierarchy with connecting lines, dual status indicators (reported vs. computed), and proper visual grouping.

**What exists**:
- Specification in [`docs/TODO/HIERARCHICAL_TREE_VIEW.md`](docs/TODO/HIERARCHICAL_TREE_VIEW.md).
- Backend `DashboardService.buildTree()` already returns both `ReportedStatus` and `AssessedStatus`.

**Effort estimate**: 6–8 days.

---

## Infrastructure Hardening

**Current state**: nginx has TLS termination, security headers, and basic rate limiting. No WAF or brute-force protection beyond Keycloak's built-in lockout.

**Target**:
- **ModSecurity + OWASP CRS** on nginx (WAF).
- **fail2ban** monitoring nginx logs for sustained 401/403/404 patterns.
- **TOTP enforcement** in Keycloak for all users (currently optional).
- **Programmatic firewall integration** (one-way IP blackholing from app to nftables).

**What exists**:
- Advisory documentation in [`docs/SECURITY_HARDENING.md`](docs/SECURITY_HARDENING.md).
- nginx config structure ready for ModSecurity module inclusion.
- Keycloak realm supports required actions for OTP.

**Effort estimate**: 2–3 days for ModSecurity + fail2ban; TOTP is a Keycloak config change.

---

## Automated Test Suite

**Current state**: Zero `*_test.go` files. `TESTING.md` provides manual procedures only.

**Target**:
- **Cedar/authz tests**: Role x action x resource scope x TLP matrix (critical quality gate).
- **Domain validation tests**: Enum parsing, status transitions, TLP rules.
- **Storage tests**: Against real PostgreSQL (Docker), pagination, transactions.
- **Service tests**: Business logic with mocked stores.
- **Handler tests**: HTTP request/response via `httptest`.
- **Markdown/XSS tests**: Adversarial input through the rendering pipeline.

**Effort estimate**: 3–5 days for meaningful coverage of the authorization matrix and critical paths.
