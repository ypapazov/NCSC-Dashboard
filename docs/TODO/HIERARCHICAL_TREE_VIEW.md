# Hierarchical Tree View Redesign

**Goal**: Replace the current indented-list tree with a visual hierarchy that reads top-down, root → sectors → sub-sectors → organizations, connected by lines and showing dual status indicators at every node.

---

## 1. Proposed Layout

```
                    ┌──────────────┐
                    │   Platform   │
                    │ ● DEGRADED   │  ← computed from sectors
                    └──────┬───────┘
               ┌───────────┼───────────┐
               │           │           │
       ┌───────┴───┐ ┌────┴────┐ ┌────┴──────────┐
       │Government │ │Finance  │ │Crit. Infra.   │
       │ ○ DEGRADED│ │ ○ DEG.  │ │ ○ IMPAIRED    │  ← from status reports
       │ ● DEGRADED│ │ ● DEG.  │ │ ● IMPAIRED    │  ← computed from children
       └─────┬─────┘ └────┬────┘ └───┬───────┬───┘
          ┌──┴──┐      ┌──┴──┐   ┌───┴──┐ ┌──┴────┐
          │Fed. │      │ CB  │   │Energy│ │Telecom│
          │State│      │ FRA │   └──┬───┘ └──┬────┘
          └──┬──┘      └─────┘   ┌──┴──┐  ┌──┴───┐
          ┌──┴──┐                │NGO  │  │TA    │
          │DoT  │                └─────┘  └──────┘
          │NSA  │
          │SITA │
          └─────┘
```

**Key**: `○` = reported status (from status reports; GREEN if none), `●` = computed status (weighted average of children; leaf nodes show only `○`).

## 2. Status Icons

| Icon | Meaning |
|------|---------|
| `○ STATUS` | **Reported** — latest status report for this scope. Defaults to GREEN if no report exists. |
| `● STATUS ⚙` | **Computed** — mechanically derived from child statuses via formula (see §5). Only shown on non-leaf nodes. |

Leaf nodes (organizations without children) display only the reported status.

## 3. Click Behavior

| Click target | Action |
|---|---|
| Any node | Opens the side panel showing the status timeline for that scope (recursive: includes child reports). Highlights the selected node. |
| Platform root | Shows aggregated timeline across all sectors. |
| Expand/collapse toggle | Expands or collapses the sub-tree below that node (CSS transition). |
| Side panel `×` | Closes panel and clears selection. |

## 4. Implementation Approach

### Phase 1: CSS/HTML Layout (no JS framework change)

Use CSS Grid or Flexbox to lay out rows:

```
Row 0: Platform root (centered)
Row 1: Top-level sectors (evenly spaced, centered)
Row 2: Sub-sectors (grouped under parents)
Row 3: Organizations (grouped under parents)
```

Connecting lines drawn with:
- **Option A**: CSS `::before`/`::after` pseudo-elements with borders (simple, works well for 2–3 levels).
- **Option B**: Inline SVG `<line>` elements computed from element positions via a small JS helper on render (more flexible, required if tree depth > 3).
- **Option C**: A lightweight library like [Treant.js](https://fperucic.github.io/treant-js/) or [d3-hierarchy](https://d3js.org/d3-hierarchy) for layout calculation only, rendering to DOM elements that HTMX can still target.

**Recommendation**: Option A for MVP, Option C for long-term. The tree is small enough (< 50 nodes) that manual CSS layout is viable and keeps HTMX compatibility.

### Phase 2: Server-side Data

The `DashboardService.buildTree()` already returns the full tree with both `ReportedStatus` and `AssessedStatus` (computed). The templ template receives it as `[]*service.DashboardNode`. No backend changes needed beyond what's already done.

### Phase 3: Templ Template Structure

```go
templ HierarchicalTree(root *service.DashboardNode) {
    <div class="htree">
        <div class="htree-row htree-root">
            @htreeNode(root)
        </div>
        <div class="htree-connectors"> <!-- SVG or CSS lines --> </div>
        <div class="htree-row htree-sectors">
            for _, sector := range root.Children {
                @htreeNode(sector)
            }
        </div>
        // Dynamically render deeper rows based on tree depth
    </div>
}

templ htreeNode(node *service.DashboardNode) {
    <div class="htree-node" id={"htree-" + node.ID.String()}>
        <div class="htree-node-label">{ node.Name }</div>
        <div class="htree-node-status">
            if node.ReportedStatus != "" {
                // ○ badge
            }
            if len(node.Children) > 0 && node.AssessedStatus != "" {
                // ● ⚙ badge
            }
        </div>
    </div>
}
```

### Phase 4: CSS

```css
.htree { display: flex; flex-direction: column; align-items: center; gap: 2rem; }
.htree-row { display: flex; justify-content: center; gap: 1.5rem; flex-wrap: wrap; }
.htree-node {
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    padding: .5rem 1rem;
    text-align: center;
    cursor: pointer;
    min-width: 120px;
    transition: box-shadow .2s, border-color .2s;
}
.htree-node:hover { box-shadow: 0 2px 8px rgba(0,0,0,.1); }
.htree-node.selected { border-color: var(--accent); box-shadow: 0 0 0 2px var(--accent-bg); }
```

### Phase 5: Connecting Lines (CSS pseudo-element approach)

Each sector group gets a vertical connector from the row above:

```css
.htree-connectors {
    position: relative;
    height: 2rem;
}
/* Lines rendered with absolute-positioned thin divs or SVG */
```

For dynamic positioning, a small `<script>` calculates positions after render:

```js
function drawConnectors(parentRow, childRow, connectorContainer) {
    // For each parent node, find its children in the child row
    // Draw vertical + horizontal lines between center points
}
```

## 5. Formula System

The formula that computes a parent's status from its children is defined in `internal/service/dashboard.go`:

```go
func computeStatus(node *DashboardNode) domain.AssessedStatus {
    // Current: weighted numeric average of child statuses
    // NORMAL=0, DEGRADED=1, IMPAIRED=2, CRITICAL=3
    // avg < 0.5 → NORMAL, < 1.5 → DEGRADED, < 2.5 → IMPAIRED, else CRITICAL
}
```

**Design for future custom formulas**:

The `computeStatus` function is a single point of change. To make it configurable:

1. Define a `StatusFormula` interface:
   ```go
   type StatusFormula interface {
       Compute(childStatuses []domain.AssessedStatus) domain.AssessedStatus
   }
   ```

2. Implement the current logic as `WeightedAverageFormula`.

3. Future: implement `StarlarkFormula` that evaluates a user-defined Starlark script, or `MaxFormula` (worst-case child), `MajorityFormula`, etc.

4. The formula is configurable per-sector (stored in DB) or globally (config file). Default: `WeightedAverageFormula`.

## 6. Effort Estimate

| Phase | Effort | Dependencies |
|-------|--------|-------------|
| Phase 1: CSS/HTML layout | 2–3 days | None |
| Phase 2: Server data | Already done | — |
| Phase 3: Templ template | 1–2 days | Phase 1 |
| Phase 4: CSS styling | 1 day | Phase 1 |
| Phase 5: Connecting lines | 1–2 days | Phase 3 |
| Formula interface | 0.5 day | None |
| **Total** | **~6–8 days** | |

## 7. Alternatives Considered

| Approach | Pros | Cons |
|----------|------|------|
| **D3.js tree layout** | Automatic positioning, zoom/pan, flexible | Heavier dependency, harder HTMX integration |
| **Pure CSS indented tree** (current) | Simple, works with HTMX | Doesn't convey hierarchy visually |
| **CSS Grid manual layout** (proposed) | Lightweight, HTMX-friendly, good for small trees | Manual positioning for deep trees |
| **Canvas/WebGL** | Maximum visual flexibility | Overkill, accessibility issues |

## 8. Open Questions

1. Should the hierarchical view replace the indented tree, or be an additional view mode toggle?
2. Maximum expected tree depth? (Currently 3: sector → sub-sector → org)
3. Should nodes be collapsible in the visual layout, or always expanded?
4. Mobile/responsive behavior: stack vertically or horizontal scroll?
