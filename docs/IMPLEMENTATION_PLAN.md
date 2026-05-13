# Fresnel — Implementation Plan Part 3

**Version:** 0.3  
**Date:** 2026-04-09  
**Supersedes:** `IMPLEMENTATION_PLAN_PART2.md` (milestones M1-fix through M8)

---

## 1. Current State

The application is running end-to-end in Docker Compose. A user can authenticate via Keycloak, see the dashboard, navigate the sidebar, and view list pages for events, status reports, campaigns, sectors, organizations, users, and audit entries. The Go binary compiles cleanly and all embedded templates parse without error.

### 1.1 What's Built (by Part 2 milestone)

| Milestone | Scope | Status |
|---|---|---|
| **M1-fix** | 8 Keycloak test users, full IAM seed, Makefile cleanup | **Done** |
| **M2** | Cedar authorizer (Go-native), domain types, storage interfaces + 15 Postgres stores, services (sector, org, user), admin handlers + templates, audit | **Done** (structure complete) |
| **M3** | Event domain, store, service, revisions, updates, attachments + ClamAV, markdown renderer, TLP enforcement, handlers + templates | **Done** (structure complete) |
| **M4** | Status reports, campaigns, correlations, event relationships, handlers + templates | **Done** (structure complete) |
| **M5** | Dashboard service (tree, weighted average, cache), dashboard template, formula stubs | **Done** (formulas are stubs per plan) |
| **M6** | Nudge scheduler, escalation, email (SMTP), digest, escalation reset | **Done** (runs on 15-min tick) |
| **M7** | CSS overhaul, dark theme, sidebar nav, badges, error pages | **Partial** (see gaps below) |
| **M8** | Deployment docs, security hardening guide, federation 501 stubs, nginx headers | **Done** (docs only, no actual prod deployment) |

### 1.2 File Inventory

| Area | Count | Notes |
|---|---|---|
| Go source files (`internal/`) | 74 | Across 14 packages |
| HTML templates | 32 | Dashboard, events, reports, campaigns, admin, audit, errors, partials |
| SQL migrations | 10 | Schema + IAM + audit + pgvector + seeds |
| Static assets | 4 + 1 CSS | htmx.min.js, keycloak.min.js, app.js, fresnel.css |
| Documentation | 9 files | Architecture, requirements, deployment, security, testing, plans |

### 1.3 What's Verified Working

These have been validated against a live Docker Compose stack:

- Docker Compose brings up postgres, keycloak, clamav, fresnel, nginx
- Postgres migrations run automatically on startup
- Keycloak realm imports with 8 test users
- OIDC login flow completes (Keycloak → app shell → keycloak-js → token)
- URL fragment cleanup after OIDC redirect
- Static files serve with correct MIME types through nginx
- Sidebar navigation renders and highlights active section
- `/api/v1/users/me` returns authenticated user JSON
- `/api/v1/dashboard` renders the sector/org status tree
- `/api/v1/events` returns event list (HTML and JSON)
- `/api/v1/status-reports`, `/api/v1/campaigns` list pages render
- `/api/v1/sectors`, `/api/v1/orgs`, `/api/v1/users` admin pages render
- `/api/v1/audit` renders audit entries
- `/api/v1/nav` returns sidebar HTML fragment
- Content negotiation (JSON vs HTML) based on Accept header
- HTMX-driven SPA navigation with `hx-push-url` history
- Nudge scheduler starts and attempts email delivery (SMTP-off warning in logs)
- ClamAV client initializes on startup
- TLS via nginx with self-signed cert
- Security headers (CSP, HSTS, X-Frame-Options, etc.)

---

## 2. Known Gaps and Issues

### 2.1 Template–Data Mismatches (Detail + Form Views)

The list views work, but **detail and form templates** still reference fields that don't exist on their Go data structs. These will produce 500 errors when a user clicks into a specific event, report, or campaign. The mismatches fall into categories:

| Category | Examples | Affected templates |
|---|---|---|
| Display-name fields | `.OrgName`, `.SectorName`, `.SubmitterName`, `.AuthorName`, `.ScopeName` | event detail, report detail/list, org list |
| Computed HTML | `.DescriptionHTML`, `.BodyHTML` | event detail, report detail, campaign detail |
| Permission flags | `.CanEdit`, `.CanDelete` | event detail, report detail, campaign detail |
| Related data | `.Attachments`, `.Correlations`, `.Revisions`, `.LinkedEvents` | event detail, report detail, campaign detail |
| Form options | `.EventTypes`, `.Sectors`, `.AvailableRecipients`, `.ScopeOptions` | event form, report form |
| Pagination | `.Pagination` (full struct with HasNext etc.) | removed from lists, but pagination partial still exists unused |

**Resolution approach:** Either enrich the handler data structs (add joins/lookups in the handler layer) or simplify the detail templates to use IDs. The former is better UX; the latter is faster to ship.

### 2.2 Cedar Authorization (Coarse Only)

The `CedarGate` middleware is a passthrough placeholder. Fine-grained authorization is implemented in the service layer (each service calls `authz.Authorize`), but:

- There's no evidence of actual authorization being enforced end-to-end in the running app — the `LoadAuthContext` path from Keycloak `sub` → Fresnel user → roles works, but whether the Cedar evaluator correctly restricts a `VIEWER` from creating events hasn't been validated.
- The Cedar evaluator is Go-native (role matrix), not using Cedar policy files. This is fine architecturally but means there's no policy corpus to audit.

### 2.3 No Automated Tests

Zero Go tests exist. The testing doc (`TESTING.md`) provides manual procedures only.

### 2.4 Keycloak Sub Linking

The seed migration uses placeholder `keycloak_sub` values (`'kc-sub-platform-root'` etc.), not real Keycloak user IDs. The `LoadAuthContext` function matches by email fallback, but this needs validation.

### 2.5 Starlark Formulas (Deferred)

The formula service is stubbed. `Get` and `Set` return errors. The dashboard uses a hardcoded weighted-average formula. The UI shows formulas as "coming soon" (disabled).

### 2.6 Frontend Limitations

- No client-side form validation
- No loading/error states beyond basic HTMX indicators
- No Markdown editor (just a plain `<textarea>`)
- Org context switcher calls a non-existent `/api/v1/users/me/org-context` PUT endpoint
- `highlightActiveNav` uses simple string matching (fragile for nested paths)

---

## 3. What Remains (Priority Order)

### P0 — Make Detail Views Work

Without this, clicking any item in a list view produces a 500. This blocks all testing beyond list browsing.

| Task | Effort | Approach |
|---|---|---|
| Enrich `EventDetailData` with attachments, revisions, correlations, display names | Medium | Add lookups in the event handler's `Get` method |
| Enrich `StatusReportDetailData` with linked events, revisions | Medium | Same pattern |
| Enrich `CampaignDetailData` with linked events | Small | Same pattern |
| Fix event form to populate `EventTypes` and `Sectors` options | Small | Add constants + sector list to form data |
| Fix status report form to populate scope options | Small | Same pattern |
| Add Markdown rendering to detail views (`.DescriptionHTML`) | Small | Call `markdown.Render()` in handler, add `safeHTML` funcMap entry |

**Estimated effort:** 1–2 days.

### P1 — Validate Authorization End-to-End

| Task | Effort |
|---|---|
| Log in as each of the 8 test users and verify scope restrictions | Manual, 2h |
| Fix any Cedar evaluator bugs found during validation | Variable |
| Write a basic Go test for the Cedar evaluator with role/scope matrix | 0.5 day |

### P2 — Automated Test Coverage

| Task | Effort |
|---|---|
| Unit tests for domain validation (`Validate()` methods) | 0.5 day |
| Unit tests for Cedar evaluator (role matrix, TLP rules) | 0.5 day |
| Integration tests for critical service flows (create event, update, status transitions) | 1 day |
| Integration tests for auth middleware (token validation, email fallback) | 0.5 day |

### P3 — Frontend Polish

| Task | Effort |
|---|---|
| Pagination (add proper `PaginationData` struct, wire in handlers) | 0.5 day |
| Filter bars on list views (wire `Filter` structs through handlers) | 1 day |
| Org context switcher (add `/api/v1/users/me/org-context` endpoint) | 0.5 day |
| Client-side form validation | 0.5 day |
| Markdown editor integration (e.g. EasyMDE or similar) | 0.5 day |
| Loading/error states | 0.5 day |

### P4 — Starlark Formulas

| Task | Effort |
|---|---|
| Integrate Starlark interpreter | 1 day |
| Formula validation + sandbox | 1 day |
| Formula management UI | 0.5 day |
| Test coverage for formula execution | 0.5 day |

### P5 — Production Readiness

| Task | Effort |
|---|---|
| Production Docker Compose (or Kubernetes manifests) | 1 day |
| Database backup strategy | 0.5 day |
| Keycloak production configuration (TOTP, password policy) | 0.5 day |
| ModSecurity / WAF rules | 0.5 day |
| Monitoring + alerting (Prometheus metrics endpoint) | 1 day |
| Rate limiting tuning | 0.5 day |

### P6 — Federation

| Task | Effort |
|---|---|
| Design federation protocol (STIX/TAXII or custom) | Research phase |
| Implement federation endpoints | Variable |
| Peer management UI | Variable |

---

## 4. Recommended Next Steps

1. **P0 first** — Fix detail views so the full read path works. This unblocks all manual testing.
2. **P1 in parallel** — Validate authorization with real logins while detail views are being fixed.
3. **P2 selectively** — Write tests for the Cedar evaluator and critical service flows. Don't aim for 100% coverage; focus on the authorization matrix and state transitions.
4. **Frontend decision** — Evaluate whether HTMX remains the right choice before investing heavily in P3. See `docs/FRONTEND_ANALYSIS.md` for a comparative analysis. If switching frameworks, do it before P3.

---

## 5. Architecture Diagram (Current)

```
Browser
  │
  ├── keycloak-js (OIDC PKCE) ──→ Keycloak (port 8081)
  │
  └── HTMX / fetch ──→ nginx (443, TLS)
                          │
                          └──→ fresnel (8080)
                                 │
                                 ├── Middleware: logging → audit ctx → OIDC → cedar gate → content neg
                                 │
                                 ├── Handlers (15 files)
                                 │     ├── HTML templates (32 files, embedded)
                                 │     └── JSON responses
                                 │
                                 ├── Service layer (13 services)
                                 │     ├── Cedar authorizer (Go-native role matrix)
                                 │     ├── Markdown renderer (goldmark + bluemonday)
                                 │     ├── ClamAV client (TCP)
                                 │     └── SMTP mailer
                                 │
                                 └── Postgres stores (15 stores)
                                       └── PostgreSQL + pgvector (port 5432)
```

---

## 6. Decision Log

| # | Decision | Rationale | Date |
|---|---|---|---|
| D1 | Go-native Cedar evaluator instead of `cedar-go` policy files | `cedar-go` lacks policy template support; Go role matrix is more debuggable | 2026-04-09 |
| D2 | Starlark formulas deferred | Not blocking any other milestone; can be added independently | 2026-04-09 |
| D3 | HTMX for frontend | Minimal JS, server-rendered HTML, good for admin/dashboard UIs | 2026-04-09 |
| D4 | ClamAV via TCP (not Unix socket) | Docker networking between containers requires TCP | 2026-04-09 |
| D5 | `directAccessGrantsEnabled: true` in dev realm | Required for CLI-based testing via `curl` | 2026-04-09 |
| D6 | Template data simplified for list views | Removed filter/pagination to unblock initial render; will be re-added in P3 | 2026-04-09 |
