# Fresnel — UI Visualization Specification

**Version**: 0.1
**Status**: Supplement to Requirements Specification v0.1.1 and Architecture v0.1
**Scope**: Dashboard views, swimlane view, and correlation graph

---

## 1. Overview

Fresnel's primary interface serves two distinct decision-maker needs:

- **"How is everyone doing?"** — the hierarchical status view (tree dashboard)
- **"What's happening across the board?"** — the activity view (swimlane)

These are two views of the same underlying data, togglable on the main page. A third view — the correlation graph — visualizes relationships between events and is prioritized for Phase 2.

---

## 2. View 1: Hierarchical Dashboard (PoC — Tree View)

Defined in the main requirements specification, Section 4.1. This is the default view.

The tree shows global status at the top, sectors (recursive) below, organizations at the leaves. Each node is color-coded by assessed status. Selecting a node opens the side panel timeline. Campaigns appear as a horizontal section below the tree.

**Implementation**: Pure HTMX. Server-rendered HTML fragments. No client-side JS beyond the existing Markdown editor.

---

## 3. View 2: Swimlane View (PoC — Activity Lanes)

### 3.1 Concept

A vertical list of sectors and/or organizations, each rendered as a horizontal lane. Events within each lane are represented as cards, scrollable horizontally. The view prioritizes scanning recent activity across many organizational units simultaneously.

### 3.2 Layout

```
┌─ [Toggle: Tree | Lanes] ──── [Filters: Impact | TLP | Status | Type | Date Range] ─┐
│                                                                                       │
│ ┌─── Government ► Federal ──────────────────────────────────────────────────────── ► │
│ │ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐                  │
│ │ │ Event A  │ │ Event B  │ │ Event C  │ │ Event D  │ │ Event E  │  ← scroll →      │
│ │ │ ■ CRIT   │ │ ■ HIGH   │ │ ■ MED    │ │ ■ MED    │ │ ■ LOW    │                  │
│ │ │ TLP:AMB  │ │ TLP:GRN  │ │ TLP:GRN  │ │ TLP:AMB  │ │ TLP:CLR  │                  │
│ │ │ INVEST.  │ │ OPEN     │ │ MITIG.   │ │ RESOLVED │ │ OPEN     │                  │
│ │ │ 2h ago   │ │ 5h ago   │ │ 1d ago   │ │ 2d ago   │ │ 3d ago   │                  │
│ │ └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘                  │
│ └───────────────────────────────────────────────────────────────────────────────────  │
│                                                                                       │
│ ┌─── Government ► State ────────────────────────────────────────────────────────── ► │
│ │ ┌──────────┐ ┌──────────┐                                                          │
│ │ │ Event F  │ │ Event G  │                                                          │
│ │ │ ■ LOW    │ │ ■ INFO   │                                                          │
│ │ │ ...      │ │ ...      │                                                          │
│ │ └──────────┘ └──────────┘                                                          │
│ └───────────────────────────────────────────────────────────────────────────────────  │
│                                                                                       │
│ ┌─── Finance ───────────────────────────────────────────────────────────────────── ► │
│ │ ┌──────────┐ ┌──────────┐ ┌──────────┐                                             │
│ │ │ Event H  │ │ Event I  │ │ Event J  │                                             │
│ │ │ ...      │ │ ...      │ │ ...      │                                             │
│ │ └──────────┘ └──────────┘ └──────────┘                                             │
│ └───────────────────────────────────────────────────────────────────────────────────  │
│                                                                                       │
│ ═══ Campaigns ═══════════════════════════════════════════════════════════════════════  │
│ ┌─── Q4 Infrastructure Upgrade ─────────────────────────────────────────────────── ► │
│ │ ┌──────────┐ ┌──────────┐ ┌──────────┐                                             │
│ │ │ Event A  │ │ Event H  │ │ Event K  │  (events from multiple orgs/sectors)        │
│ │ └──────────┘ └──────────┘ └──────────┘                                             │
│ └───────────────────────────────────────────────────────────────────────────────────  │
└───────────────────────────────────────────────────────────────────────────────────────┘
```

### 3.3 Lane Structure

Each lane represents one organizational unit. The lane hierarchy mirrors the sector tree:

- Top-level sectors appear as lane group headers (not lanes themselves, unless they have directly attached orgs).
- Subsectors appear as indented lane group headers.
- Organizations appear as individual lanes with their events.
- A sector with no subsectors and direct orgs shows the orgs as lanes under the sector header.

Lanes are ordered top-to-bottom by: sector hierarchy order, then within a sector by assessed status severity (worst first), then alphabetically. This puts the orgs that need attention at the top.

Campaigns appear as a separate section below the org lanes (same as the tree view). Campaign lanes contain events from across orgs/sectors, providing the cross-cutting view.

### 3.4 Event Cards

Each card displays:

| Element | Description |
|---|---|
| Title | Event title, truncated to 2 lines |
| Impact badge | Color-coded: black/red/amber/yellow/green |
| TLP badge | Small label: RED/AMB+S/AMB/GRN/CLR |
| Status | Current lifecycle status |
| Timestamp | Relative time since last update ("2h ago", "3d ago") |
| Type icon | Small icon or abbreviation for event type category |

Cards are ordered within a lane by: most recently updated first (default), or configurable to sort by impact severity, creation date, or status.

Clicking a card navigates to the event detail page via standard HTMX navigation (`hx-get`, `hx-push-url`).

### 3.5 Lane Behavior

**Lane header** shows: org/sector name, assessed status badge, event count (visible / total if some are restricted).

**Restricted content**: If the user cannot see some events in a lane, the lane header shows "(+ N restricted)" and no placeholder cards appear — the restricted events are simply absent from the card row. The count tells the user they're not seeing everything.

**Empty lanes**: Orgs with no events in the current filter range show a single muted card: "No events." Lanes can be collapsed individually or hidden via a "hide empty lanes" toggle.

**Lazy loading**: Lanes load as the user scrolls vertically. Each lane fetches its events independently:
```
hx-get="/api/v1/events?organization={id}&sort=updated_at&limit=20"
hx-trigger="revealed"
```

**Horizontal scroll pagination**: Within a lane, if there are more events than fit on screen, scrolling right loads more:
```
<!-- Sentinel element at end of card row -->
<div hx-get="/api/v1/events?organization={id}&sort=updated_at&offset=20&limit=20"
     hx-trigger="intersect"
     hx-swap="beforebegin">
</div>
```

### 3.6 Filters

A shared filter bar at the top applies to all lanes simultaneously. Changing a filter reloads all visible lanes.

| Filter | Type | Options |
|---|---|---|
| Impact | Multi-select | Critical, High, Medium, Low, Info |
| TLP | Multi-select | RED, AMBER+STRICT, AMBER, GREEN, CLEAR |
| Status | Multi-select | Open, Investigating, Mitigating, Resolved, Closed |
| Event Type | Multi-select | All types from taxonomy |
| Date Range | Range picker | Events updated within the selected range |
| Sector | Tree selector | Filter to a specific sector subtree |

Filters are URL-encoded in query parameters so the view state is shareable/bookmarkable.

### 3.7 View Toggle

A toggle control at the top of the dashboard switches between Tree View and Swimlane View. The toggle preserves active filters where applicable. Both views share the same URL base (`/dashboard`) with a `view=tree` or `view=lanes` query parameter.

### 3.8 Implementation

**Pure HTMX + CSS.** No client-side JavaScript required.

- Lane layout: CSS `overflow-x: auto` on each lane's card container, `display: flex` with `flex-wrap: nowrap` for horizontal card flow.
- Vertical scroll: Standard page scroll. Lanes load lazily via `hx-trigger="revealed"`.
- Horizontal pagination: HTMX `intersect` trigger on a sentinel element.
- Filter changes: `hx-get` on the filter bar targeting the lanes container, replacing all lanes.
- Card click: `hx-get` with `hx-push-url` for navigation.
- No drag-and-drop. Events are not manually reorderable — they're sorted by data attributes.

---

## 4. View 3: Correlation Graph (Phase 2 — Priority)

### 4.1 Rationale for Priority

The correlation graph is the primary visualization for understanding how events relate to each other across organizations and sectors. For decision-makers, seeing that five orgs experienced related events in the same week — and understanding the connection pattern — is the core intelligence value of Fresnel. The tree and swimlane views show status and activity; the graph shows structure and relationships.

This is the highest-priority Phase 2 feature. It should be the first thing built after the PoC stabilizes.

### 4.2 Concept

An interactive node-link diagram where:
- **Nodes** are events (and optionally campaigns).
- **Edges** are correlations (manual, suggested, confirmed) and event relationships (sanitized_version, derived_from, etc.).
- Node size, color, and shape encode event attributes (impact, status, org, type).
- The graph is navigable: zoom, pan, drag nodes, click to open event detail.

### 4.3 Entry Points

The graph can be entered from multiple contexts:

| Entry Point | Initial Graph Scope |
|---|---|
| Event detail page → "View correlations" | The selected event + all directly correlated events (1-hop neighborhood) |
| Campaign detail page → "View graph" | All events in the campaign + their correlations |
| Dashboard → "Correlation explorer" | Full graph for the user's visible scope (filtered by access control) |
| Search results → "Graph view" | Selected events + their correlations |

### 4.4 Node Rendering

| Attribute | Visual Encoding |
|---|---|
| Event type category | Node shape (circle = security, square = disruption, diamond = operational, triangle = environmental, hexagon = advisory) |
| Impact | Node border color (black/red/amber/yellow/green) |
| Status | Node fill opacity (open/investigating = solid, mitigating = 75%, resolved = 50%, closed = 25%) |
| Organization | Node label includes org abbreviation |
| TLP | Not visually encoded on the node itself — restricted events simply don't appear (same as other views) |

### 4.5 Edge Rendering

| Attribute | Visual Encoding |
|---|---|
| Correlation type | Line style (MANUAL = solid, CONFIRMED = solid + thick, SUGGESTED = dashed) |
| Event relationship | Line style (dotted, with label text on hover) |
| Campaign membership | Not shown as edges — campaigns are shown as background regions/hulls enclosing their member events |

### 4.6 Interactions

| Action | Result |
|---|---|
| Click node | Side panel opens with event summary (same pattern as tree/swimlane) |
| Double-click node | Navigate to event detail page |
| Hover node | Tooltip with title, impact, status, org, timestamp |
| Hover edge | Tooltip with correlation label and type |
| Drag node | Repositions node (force-directed layout adjusts) |
| Scroll wheel | Zoom in/out |
| Click + drag background | Pan |
| Filter controls | Same filter bar as swimlane — impact, TLP, status, type, date range. Filtering hides nodes/edges that don't match. |
| Expand neighborhood | Right-click or button on a node → load 1 more hop of correlations |
| Collapse | Right-click → collapse node back to single node |

### 4.7 Access Control in the Graph

The graph is subject to the same Cedar authorization as every other view. Nodes the user cannot see are absent — not hidden behind a placeholder, simply not present. Edges to invisible nodes are also absent.

This means different users see structurally different graphs for the same scope. A sector root sees the full correlation web. An org viewer sees only their org's events and correlations to other events they can access. This is the correct behavior — the graph shows what you're permitted to know.

**Campaign hulls**: If a campaign contains events the user can't see, the hull still renders around the visible events. The campaign label shows "(+ N restricted)" consistent with other views.

### 4.8 Performance Considerations

Large graphs (hundreds of nodes) become unreadable and slow. Mitigations:

- **Default scope limit**: Graph entry points load a limited neighborhood (1-2 hops, max 100 nodes). User can expand manually.
- **Clustering**: When a sector or org has many events, they can be collapsed into a single summary node showing count and worst-case impact. Clicking expands.
- **Server-side graph computation**: The API returns a pre-computed node/edge list, not raw data for the client to assemble. The server applies access control, computes layout hints (optional), and returns a clean graph payload.

### 4.9 Implementation

**Requires client-side JavaScript.** HTMX cannot render interactive graph visualizations.

**Recommended library**: Cytoscape.js.

| Consideration | Cytoscape.js | D3.js | vis-network |
|---|---|---|---|
| Purpose-built for graphs | Yes | General visualization | Yes |
| Layout algorithms | Extensive built-in (force-directed, hierarchical, circle, grid) | Manual implementation | Built-in but fewer options |
| Interaction support | Click, hover, drag, zoom, pan — all built-in | Manual implementation | Built-in |
| Styling | CSS-like stylesheet syntax | SVG/Canvas manipulation | Configuration object |
| Bundle size | ~300KB min | ~90KB min | ~200KB min |
| Learning curve | Moderate | High | Low |
| Extensibility | Plugin system | Unlimited | Limited |

Cytoscape.js is the best fit: purpose-built, comprehensive interaction model, declarative styling, and good performance up to ~1000 nodes. D3 offers more control but requires significantly more implementation effort for the same result. vis-network is simpler but less flexible for the access-control-driven dynamic filtering Fresnel needs.

**Integration pattern:**
```
┌────────────────────────────────────────────────────────────────┐
│ /graph page (HTMX shell)                                       │
│                                                                │
│  ┌─── Filter Bar (HTMX) ────────────────────────────────────┐ │
│  │ [Impact ▼] [Status ▼] [Type ▼] [Date Range]              │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                │
│  ┌─── Graph Container (Cytoscape.js) ───────────────────────┐ │
│  │                                                           │ │
│  │  Fetches JSON from:                                       │ │
│  │  GET /api/v1/events/{id}/correlations?format=graph        │ │
│  │  or                                                       │ │
│  │  GET /api/v1/campaigns/{id}/graph                         │ │
│  │                                                           │ │
│  │  Renders interactive node-link diagram                    │ │
│  │                                                           │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                │
│  ┌─── Side Panel (HTMX) ────────────────────────────────────┐ │
│  │ Event summary loaded on node click                        │ │
│  │ hx-get="/api/v1/events/{id}?partial=summary"             │ │
│  └───────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────┘
```

The page is an HTMX shell. The filter bar and side panel are HTMX partials. The graph container is a Cytoscape.js instance that fetches JSON from the API. Filter changes trigger the HTMX filter bar to emit a custom event that the Cytoscape component listens to, refetching and re-rendering the graph with updated parameters. Node clicks trigger HTMX loads into the side panel. This keeps the HTMX-first architecture while using JS only where HTML cannot go.

### 4.10 API Requirements

Two new endpoints (or extensions to existing ones) needed for graph rendering:

```
GET /api/v1/events/{id}/correlations?format=graph&depth=2
  → Returns: { nodes: [...], edges: [...] }
  → Nodes include: id, title, impact, status, type, org, timestamps
  → Edges include: id, source, target, label, correlation_type
  → Access-control filtered server-side

GET /api/v1/campaigns/{id}/graph
  → Returns same structure, scoped to campaign events + their correlations

GET /api/v1/graph/explore?sector={id}&impact=CRITICAL,HIGH&limit=100
  → Full scope graph explorer with filters
```

These return JSON only (no HTML representation). The `format=graph` parameter on the correlations endpoint distinguishes between the list view (default, returns a flat list) and the graph view (returns node/edge structure).

### 4.11 Vendoring

Cytoscape.js must be vendored (downloaded and served from the Fresnel nginx container), not loaded from a CDN. Sovereign deployment means no external runtime dependencies. Pin to a specific version in the build process.

---

## 5. View Relationships and Navigation

### 5.1 View Switching

The main dashboard supports toggling between Tree and Swimlane views. The Correlation Graph is a separate page, accessible from event detail pages, campaign pages, and a top-level navigation entry.

```
Main Dashboard ──┬── Tree View (default)
                 └── Swimlane View (toggle)

Event Detail ────┬── Event content, updates, revisions
                 └── "View correlations" → Graph View (Phase 2)

Campaign Detail ─┬── Campaign content, linked events
                 └── "View graph" → Graph View (Phase 2)

Top Nav ─────────┬── Dashboard
                 ├── Events (list/search)
                 ├── Campaigns
                 ├── Correlation Explorer → Graph View (Phase 2)
                 └── Admin (IAM, hierarchy, formulas)
```

### 5.2 Shared Patterns

All three views share:
- **Side panel**: Clicking a node/card/tree-item opens the same side panel component with the same content.
- **Filter bar**: Same filter dimensions, same URL parameter encoding. Switching views preserves active filters.
- **Access control**: Same Cedar evaluation. Different users see different data in all views.
- **Restricted content indicators**: Same pattern ("N organizations have restricted content") across all views.

### 5.3 Consistent Card Component

The event card used in the swimlane view should be a reusable HTMX partial that also appears in: timeline side panel entries, campaign event lists, search results, and graph side panel summaries. One template, many contexts.

---

## 6. Implementation Phases

| Phase | View | Technology | Dependency |
|---|---|---|---|
| PoC | Tree Dashboard | HTMX | Core API, Cedar, Starlark |
| PoC | Swimlane View | HTMX + CSS | Same as tree — different template, same data |
| Phase 2 (Priority) | Correlation Graph | Cytoscape.js + HTMX shell | Graph API endpoints, vendored Cytoscape.js |

The swimlane adds no new architectural components — it's a rendering concern. The graph is the first feature that introduces a meaningful client-side JS dependency beyond the Markdown editor. Cytoscape.js is vendored like all other frontend assets.

---

*This specification supplements the main Requirements and Architecture documents. It does not modify them — it adds UI detail that was underspecified in those documents.*