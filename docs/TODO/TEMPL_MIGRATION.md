# Proposal: Migrate from html/template to templ

**Status:** Draft  
**Effort:** 4–5 days (or 5–6 days if combined with P0 detail-view fixes)  
**Prerequisite:** None (but ideally done alongside or before P0)  
**Risk:** Low–medium. The change is mechanical and incremental. Each template can be migrated independently.

---

## Problem

Go's `html/template` is stringly typed. Template expressions like `{{.Events}}`, `{{.Impact | lower}}`, and `{{template "pagination" .Pagination}}` are evaluated at runtime. A typo, a renamed struct field, or a missing funcMap entry produces a 500 error that is only discoverable by visiting the page.

This has been the dominant source of bugs in the project:

| Bug | Cause | Time to find |
|---|---|---|
| Server crash loop on startup | `fmtTime` function not in funcMap | Runtime |
| Dashboard 500 | `.Sectors` field doesn't exist on `DashboardData` | Runtime |
| Event list 500 | `.Items` used but field is `.Events` | Runtime |
| User detail 500 | `user_detail` template name doesn't exist | Runtime |
| Badge rendering crash | `lower` receives `domain.TLP` not `string` | Runtime |
| Status report list 500 | `.Filter` field doesn't exist on struct | Runtime |
| 7 handler template name mismatches | Handler says `"audit_list"`, template defines `"audit_log"` | Runtime |

Every single one of these would have been a **compile error** with templ.

---

## What is templ?

[templ](https://templ.guide/) is a Go HTML templating language that generates type-safe Go code. A `.templ` file defines components as Go functions with HTML bodies. The `templ generate` command compiles `.templ` files into `.go` files that are then compiled normally by `go build`.

```
┌──────────────┐     templ generate     ┌──────────────┐     go build     ┌─────────┐
│  *.templ     │ ──────────────────────► │  *_templ.go  │ ──────────────► │ binary  │
│  (you edit)  │                         │  (generated) │                 │         │
└──────────────┘                         └──────────────┘                 └─────────┘
```

Key properties:
- **Type-safe.** Component parameters are Go types. Field access is Go code. Mismatches are compile errors.
- **Component model.** Components are functions that accept typed arguments and return `templ.Component`. They compose via function calls, not string names.
- **Auto-escaping.** String interpolation is escaped by default. Raw HTML requires explicit `templ.Raw()`.
- **HTMX-native.** Outputs standard HTML with attributes. `hx-get`, `hx-target`, etc. work exactly as before.
- **No runtime overhead.** Generated code writes directly to `io.Writer`. No template parsing, no reflection, no funcMap lookup.

---

## Current State

```
templates/                    # 32 HTML files, 3,284 lines total
├── layouts/base.html         # App shell (30 lines)
├── dashboard/index.html      # Status tree (139 lines)
├── events/                   # list, detail, form, updates (4 files, ~600 lines)
├── status_reports/           # list, detail, form (3 files, ~290 lines)
├── campaigns/                # list, detail, form (3 files, ~240 lines)
├── admin/                    # users, orgs, sectors, roles, detail/form (10 files, ~500 lines)
├── audit/log.html            # Audit log (55 lines)
├── partials/                 # nav, badges, pagination (6 files, ~100 lines)
└── errors/                   # 403, 404, 500 (3 files, ~60 lines)

internal/httpserver/templates/fs.go    # ParseFS + funcMap (105 lines)
templates/embed.go                     # //go:embed directive
```

The handler layer uses this pattern everywhere:

```go
func (h *EventHandler) List(w http.ResponseWriter, r *http.Request) {
    // ... fetch data ...
    respond(w, r, h.tmpl, "event_list", http.StatusOK, EventListData{...})
}
```

Where `respond` does:
```go
if getRenderKind(r) == requestctx.RenderJSON {
    respondJSON(w, status, data)
    return
}
tmpl.ExecuteTemplate(w, templateName, data)
```

---

## Proposed Implementation

### Phase 1: Setup (0.5 day)

1. Install templ: `go install github.com/a-h/templ/cmd/templ@latest`
2. Add to Makefile:
   ```makefile
   generate:
   	templ generate

   build: generate
   	go build -o bin/fresnel ./cmd/fresnel
   ```
3. Add `templ generate` to Dockerfile build stage
4. Create `internal/views/` package for templ components
5. Create shared helper functions (replace funcMap):
   ```go
   // internal/views/helpers.go
   package views

   func FmtTime(t time.Time) string { ... }
   func FmtBytes(b int64) string { ... }
   func Lower(v any) string { ... }
   func FmtUser(id uuid.UUID) string { ... }
   ```

### Phase 2: Migrate Partials and Layout (0.5 day)

Start with the smallest, most-reused pieces:

**Before** (`partials/tlp_badge.html`):
```html
{{define "tlp_badge"}}
<span class="badge badge-tlp-{{. | lower}}">TLP:{{.}}</span>
{{end}}
```

**After** (`internal/views/badges.templ`):
```
package views

import "strings"
import "fmt"

templ TLPBadge(tlp fmt.Stringer) {
    <span class={ "badge badge-tlp-" + strings.ToLower(tlp.String()) }>
        TLP:{ tlp.String() }
    </span>
}

templ ImpactBadge(impact fmt.Stringer) {
    <span class={ "badge badge-impact-" + strings.ToLower(impact.String()) }>
        { impact.String() }
    </span>
}

templ StatusBadge(status fmt.Stringer) {
    <span class={ "badge badge-status-" + strings.ToLower(status.String()) }>
        { status.String() }
    </span>
}
```

Also: `Nav()`, `Pagination(...)`, `Shell(keycloakURL, realm, clientID string)`, error pages.

### Phase 3: Migrate List Views (1 day)

One template at a time. Each migration follows this pattern:

**Before** (`events/list.html` + `EventListData` struct + handler):
```html
{{define "event_list"}}
...
{{range .Events}}
<td>{{.Title}}</td>
<td><span class="badge badge-impact-{{.Impact | lower}}">{{.Impact}}</span></td>
{{end}}
...
{{end}}
```

**After** (`internal/views/events.templ`):
```
package views

import "fresnel/internal/domain"

templ EventList(events []*domain.Event, total int) {
    <div class="page-header">
        <h1>Events</h1>
        <button class="btn btn-primary"
                hx-get="/api/v1/events/new" hx-target="#app" hx-swap="innerHTML"
                hx-push-url="/events/new">
            + New Event
        </button>
    </div>
    <div id="event-results">
        if len(events) > 0 {
            <div class="card" style="overflow:hidden;">
                <table class="table">
                    <thead>
                        <tr>
                            <th class="table-header">Title</th>
                            <th class="table-header">Impact</th>
                            <th class="table-header">Status</th>
                            <th class="table-header">TLP</th>
                            <th class="table-header">Date</th>
                        </tr>
                    </thead>
                    <tbody>
                        for _, e := range events {
                            <tr class="table-row clickable"
                                hx-get={ "/api/v1/events/" + e.ID.String() }
                                hx-target="#app" hx-swap="innerHTML"
                                hx-push-url={ "/events/" + e.ID.String() }>
                                <td class="table-cell truncate">{ e.Title }</td>
                                <td class="table-cell">
                                    @ImpactBadge(e.Impact)
                                </td>
                                <td class="table-cell">
                                    @StatusBadge(e.Status)
                                </td>
                                <td class="table-cell">
                                    @TLPBadge(e.TLP)
                                </td>
                                <td class="table-cell text-sm text-muted">
                                    { FmtTime(e.CreatedAt) }
                                </td>
                            </tr>
                        }
                    </tbody>
                </table>
            </div>
            <p class="text-sm text-muted" style="margin-top:.5rem;">
                { fmt.Sprintf("%d event(s) total", total) }
            </p>
        } else {
            <div class="card">
                <div class="card-body" style="text-align:center;padding:3rem;">
                    <p class="text-muted">No events found.</p>
                </div>
            </div>
        }
    </div>
}
```

**Handler changes to:**
```go
func (h *EventHandler) List(w http.ResponseWriter, r *http.Request) {
    // ... fetch data ...
    if getRenderKind(r) == requestctx.RenderJSON {
        respondJSON(w, http.StatusOK, result)
        return
    }
    w.WriteHeader(http.StatusOK)
    views.EventList(result.Items, result.TotalCount).Render(r.Context(), w)
}
```

The handler no longer needs a `*template.Template` field at all.

### Phase 4: Migrate Detail + Form Views (1.5 days)

Same pattern. The detail views are more complex because they reference sub-objects (attachments, revisions, correlations). With templ, these become component composition:

```
templ EventDetail(event *domain.Event, attachments []*domain.Attachment, revisions []*domain.EventRevision) {
    // ... typed access to all fields ...
    for _, att := range attachments {
        <span class="attachment-size">{ FmtBytes(att.SizeBytes) }</span>
    }
}
```

If the handler hasn't fetched attachments yet (P0 gap), the templ component simply won't compile until the handler passes them. This forces you to fix P0 as part of the migration.

### Phase 5: Update Handlers + Router (0.5 day)

1. Remove `tmpl *template.Template` from all handler structs
2. Remove `tmpl` parameter from handler constructors
3. Remove `respond()` / `respondHTML()` helpers (replace with direct `component.Render()` calls)
4. Remove `apptemplates.Parse()` call from router
5. Delete `internal/httpserver/templates/fs.go`
6. Delete `templates/embed.go`
7. Delete all `templates/**/*.html` files

### Phase 6: Verify + Cleanup (0.5 day)

1. `templ generate && go build ./...` — must pass with zero errors
2. `go vet ./...`
3. Delete `templates/` directory entirely
4. Update Dockerfile: add `templ generate` before `go build`
5. Smoke test all pages in browser

---

## Migration Order (Per File)

Migrate in dependency order — partials first, then pages that use them:

| Order | File(s) | Lines | Depends on |
|---|---|---|---|
| 1 | Badge partials (5 files) | 15 | — |
| 2 | Pagination | 27 | — |
| 3 | Nav | 40 | — |
| 4 | Error pages (3 files) | 60 | — |
| 5 | Shell layout | 30 | — |
| 6 | Event list | 50 | Badges |
| 7 | Status report list | 50 | Badges |
| 8 | Campaign list | 50 | Badges |
| 9 | Audit log | 55 | — |
| 10 | Admin lists (users, orgs, sectors) | 150 | — |
| 11 | Dashboard | 139 | Badges |
| 12 | Event detail + form + updates | 600 | Badges, FmtTime, FmtBytes |
| 13 | Status report detail + form | 240 | Badges, FmtTime |
| 14 | Campaign detail + form | 240 | Badges, FmtTime |
| 15 | Admin detail + form (6 files) | 200 | — |

Each step can be tested independently. The old `html/template` rendering and templ components can coexist during migration — handlers can be switched one at a time.

---

## What Changes vs. What Doesn't

### Changes

| Component | Before | After |
|---|---|---|
| Template files | 32 `.html` files in `templates/` | ~15 `.templ` files in `internal/views/` |
| Template engine | `html/template` + funcMap + `ParseFS` | `templ generate` → Go code |
| Handler structs | Carry `*template.Template` | No template reference |
| `respond()` helper | Switches on render kind, calls `ExecuteTemplate` | Switches on render kind, calls `component.Render()` |
| Build command | `go build` | `templ generate && go build` |
| Error discovery | Runtime panic | Compile error |

### Doesn't Change

- Domain types, services, stores, middleware, router structure
- CSS, JS, static assets
- HTMX attribute patterns in the HTML output
- Content negotiation (JSON vs HTML)
- The visual appearance of every page

---

## Risks and Mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| templ has a bug or missing feature | Low (v0.3+, widely used) | We use basic HTML output only; no advanced features needed |
| Build pipeline complexity | Low | One extra command (`templ generate`); can be a Makefile target |
| Team unfamiliar with templ | Medium | Syntax is minimal (Go + HTML); the learning curve is hours, not days |
| Generated files clutter the repo | Low | Add `*_templ.go` to `.gitignore` and generate in CI, or commit them (both patterns are common) |
| Coexistence during migration | None | Old and new rendering paths can coexist; migrate one handler at a time |

---

## Acceptance Criteria

- [ ] All 32 templates converted to `.templ` files
- [ ] `templ generate && go build ./...` passes with zero errors
- [ ] `templates/` directory deleted
- [ ] `internal/httpserver/templates/fs.go` deleted
- [ ] No handler struct carries `*template.Template`
- [ ] All pages render identically in the browser (visual diff)
- [ ] Dockerfile updated with `templ generate` step
- [ ] Makefile `generate` target added

---

## Decision: When to Do This

**Option A — Before P0 (recommended).** Fixing detail views requires touching every detail/form template. Doing that work in `html/template` and then converting to templ later means touching every template twice. Migrating to templ first (or simultaneously) means each template is written once in its final form.

**Option B — After P0.** Fix detail views in `html/template` now, convert to templ later. Faster to get detail views working (2 days), but adds 4 days of conversion work later that partially redoes P0 changes.

**Option C — Never.** Accept the runtime-error risk and invest in a template linter test instead. Viable if the team is small and the template surface stabilizes. But the linter test is itself ~0.5 days of work and catches fewer issues than templ's compile-time checking.
