# Fresnel — Frontend Technology Analysis

**Date:** 2026-04-09  
**Context:** Fresnel is an internal government platform for cyber incident management. The UI consists of dashboards, hierarchical trees, CRUD forms, real-time status displays, and admin panels. The user base is small (tens to low hundreds of concurrent users, not thousands). The backend is Go.

---

## Candidates

| # | Technology | Architecture |
|---|---|---|
| **A** | **HTMX** + Go `html/template` | Server-rendered HTML fragments, minimal JS |
| **B** | **React / Next.js** (static export) | Client-side SPA, JSON API, static build served by Go/nginx |
| **C** | **SvelteKit** (static adapter) | Compiled SPA, JSON API, static build served by Go/nginx |

---

## 1. Expressiveness and UI Richness

### HTMX

HTMX excels at making server-rendered HTML feel dynamic: swapping page fragments, infinite scroll, lazy-loading tabs. For Fresnel's core use cases (tables, forms, tree views, badge-heavy dashboards), it is fully sufficient.

Where it struggles:
- **Complex client-side state.** A drag-and-drop event timeline, a real-time collaboration editor, or a multi-step form wizard with client-side validation and preview are painful to build. You end up writing vanilla JS alongside HTMX, which defeats the purpose.
- **Offline / optimistic updates.** HTMX assumes the server is always available. Every interaction is a round-trip.
- **Rich interactivity.** Autocomplete with debounce, inline editing, animated transitions, modals with nested forms — all possible but require hand-rolling JS or pulling in libraries like Alpine.js.
- **Component composition.** Go's `html/template` has `{{template}}` and `{{block}}` but no props, slots, or conditional composition. Complex UIs lead to deeply nested templates that are hard to reason about.

**Floor:** Very high for CRUD/admin tools. **Ceiling:** Moderate. You hit a wall once the UI needs significant client-side logic.

### React / Next.js

React is the industry-standard SPA framework. With Next.js static export (`output: 'export'`), you get a fully client-side SPA that can be served as static files from the Go backend or nginx.

Strengths:
- **Unlimited UI complexity.** Any UI you can imagine can be built: drag-and-drop, real-time collaboration, rich text editors, data grids with sorting/filtering/grouping, chart dashboards.
- **Component model.** Reusable, composable components with props, children, context, hooks. The Fresnel badge system (`TLP:RED`, `CRITICAL`, `NORMAL`) maps naturally to React components.
- **Ecosystem.** Thousands of production-quality libraries: TanStack Table, React Hook Form, Radix UI, Recharts, Lexical (editor), React DnD.
- **TypeScript.** End-to-end type safety from API response to rendered component.

Where it struggles:
- **Bundle size.** A React + Next.js static build starts at ~80-120 KB gzipped for the framework alone, before application code. For a government intranet, this is likely irrelevant (fast network, powerful machines).
- **Build complexity.** Requires Node.js toolchain, npm/yarn, webpack/turbopack, potentially separate CI pipeline.
- **Two runtimes.** You now maintain a Go backend and a Node.js build step. The Go `embed.FS` for static files still works, but the development workflow splits.

**Floor:** High (but with significant setup overhead). **Ceiling:** Unlimited.

### SvelteKit

SvelteKit compiles components into imperative vanilla JS at build time — no virtual DOM, no runtime framework. With the static adapter, it produces a pre-rendered SPA.

Strengths:
- **Small bundles.** A comparable SvelteKit app is typically 30-50% smaller than React equivalent. The compiler eliminates framework overhead.
- **Simpler mental model.** Reactive state is a language feature (`$:` reactive declarations, `$state` runes in Svelte 5), not a library pattern (`useState`, `useEffect`, `useMemo`). Less boilerplate.
- **Component model.** Similar to React in capability (props, slots, context, stores) but with less ceremony.
- **Built-in transitions/animations.** First-class `transition:`, `animate:`, `in:/out:` directives.
- **TypeScript support.** Full TypeScript with generated types.

Where it struggles:
- **Smaller ecosystem.** Fewer component libraries than React, though growing rapidly. You'll find quality equivalents for most needs, but niche requirements may require more custom code.
- **Enterprise adoption.** Less common in large government/enterprise projects compared to React. Fewer developers will be familiar with it.
- **Build complexity.** Same as React — requires Node.js toolchain, Vite, separate build step.

**Floor:** High. **Ceiling:** Very high (practically unlimited, same as React).

### Verdict: Expressiveness

| Criterion | HTMX | React/Next | SvelteKit |
|---|---|---|---|
| Simple CRUD / tables | Excellent | Good (overkill) | Good |
| Complex dashboards | Good | Excellent | Excellent |
| Rich text / editors | Poor (needs JS libs) | Excellent | Excellent |
| Drag-and-drop | Poor | Excellent | Very good |
| Real-time updates | Moderate (SSE/polling) | Excellent (WebSocket) | Excellent |
| Multi-step wizards | Moderate | Excellent | Excellent |
| **Ceiling** | **Moderate** | **Unlimited** | **Very high** |

---

## 2. Developer Experience

### HTMX

- **Learning curve:** Minimal. HTML + a few `hx-*` attributes. Any developer can be productive in hours.
- **Tooling:** No build step. No transpiler. No bundler. Edit HTML, restart Go server (or use `air` for hot-reload). The feedback loop is extremely fast.
- **Debugging:** Browser DevTools network tab shows HTML fragments. Easy to inspect. But template errors surface at runtime (as we've experienced), not at compile time.
- **Refactoring:** Risky. Template field names are strings; renaming a struct field doesn't produce a compile error — it produces a runtime panic. This is a major DX weakness for Go templates.
- **Testing:** Go's `httptest` can test HTML responses, but verifying template correctness requires either runtime tests or manual inspection. No static analysis catches `{{.NonexistentField}}`.

### React / Next.js

- **Learning curve:** Steeper. JSX, hooks, state management patterns, build tooling. A junior developer needs weeks to be productive.
- **Tooling:** Rich. ESLint, Prettier, TypeScript compiler, React DevTools, Storybook for component development. Hot Module Replacement (HMR) provides sub-second feedback.
- **Debugging:** React DevTools shows component tree, props, state. Excellent. But debugging hydration issues or stale closure bugs in hooks can be challenging.
- **Refactoring:** Excellent. TypeScript catches missing props, wrong types, and renamed fields at compile time. This is React's killer advantage for maintainability.
- **Testing:** Jest/Vitest + React Testing Library. Well-established patterns. Component tests are straightforward.

### SvelteKit

- **Learning curve:** Lower than React, higher than HTMX. Svelte's template syntax is HTML-like, which helps. The reactivity model is intuitive.
- **Tooling:** Vite-based. Fast HMR. The Svelte compiler produces helpful warnings. Svelte DevTools exist but are less mature than React's.
- **Debugging:** Simpler than React (no virtual DOM indirection). The compiled output is readable JS.
- **Refactoring:** Good with TypeScript. Not quite as mature as React's type story (especially for complex generic components), but very solid for typical use cases.
- **Testing:** Vitest + `@testing-library/svelte`. Less ecosystem than React but sufficient.

### Verdict: Developer Experience

| Criterion | HTMX | React/Next | SvelteKit |
|---|---|---|---|
| Time to first page | Minutes | Hours | Hours |
| Ongoing velocity | High (simple), decreasing (complex) | Moderate (consistent) | High (consistent) |
| Type safety | None (templates are strings) | Excellent (TypeScript) | Very good (TypeScript) |
| Refactoring confidence | Low | High | High |
| Tooling maturity | Minimal (by design) | Best in class | Good |
| Build complexity | None | Moderate | Moderate |

---

## 3. Security

### HTMX

**Strengths:**
- **No XSS surface from client-side rendering.** All HTML is server-generated. The Go template engine auto-escapes by default. Combined with `bluemonday` for user-supplied Markdown, XSS risk is minimal.
- **No exposed API surface.** The server returns HTML, not raw data. An attacker who intercepts a response gets rendered HTML, not a structured JSON payload they can easily parse.
- **CSP-friendly.** No inline scripts needed (HTMX reads from HTML attributes). `script-src 'self'` works cleanly.

**Weaknesses:**
- **CSRF.** HTMX makes real HTTP requests (POST, PUT, DELETE) from the browser. Without CSRF tokens, these are vulnerable. HTMX has `hx-headers` for adding tokens, but it's manual wiring.
- **Attribute injection.** If user data ends up in `hx-*` attributes without escaping, an attacker could inject `hx-get` or `hx-post` targets. Go's template escaping handles this within `{{}}` blocks, but manual HTML construction is risky.
- **History injection.** `hx-push-url` can be abused if the URL comes from user input.

### React / Next.js (Static Export)

**Strengths:**
- **No server-side template injection.** The server serves only JSON. HTML is constructed client-side. This eliminates SSTI as an attack vector.
- **JSX auto-escaping.** React escapes all interpolated values by default. `dangerouslySetInnerHTML` is opt-in and obviously named.
- **Mature security tooling.** `npm audit`, Snyk, Socket.dev scan dependencies. ESLint security plugins catch common patterns.

**Weaknesses:**
- **XSS via `dangerouslySetInnerHTML`.** Rendering Markdown/rich text requires this. Must use a sanitizer (DOMPurify).
- **Dependency supply chain.** A React + Next.js project pulls in hundreds of npm packages. Each is an attack surface. Supply chain attacks (event-stream, ua-parser-js, colors.js) are a real and recurring threat.
- **Exposed API surface.** The JSON API is fully exposed and documented by the client-side code. An attacker can reverse-engineer every endpoint and parameter from the JS bundle.
- **CSP complexity.** Next.js often requires `unsafe-eval` or nonce-based CSP for its hydration. Static export avoids some of this, but third-party libraries may still require CSP exceptions.
- **Secret leakage.** Client-side code can accidentally include environment variables, API keys, or internal URLs. Build-time environment injection (`NEXT_PUBLIC_*`) is an easy footgun.

### SvelteKit (Static Adapter)

**Strengths:**
- **Same as React** for auto-escaping (`{@html ...}` is explicit opt-in).
- **Smaller bundle = smaller attack surface.** Less framework code to exploit.
- **Fewer dependencies.** SvelteKit's dependency tree is significantly smaller than Next.js. Less supply chain risk.
- **CSP-friendly.** SvelteKit's static output doesn't require `unsafe-eval`.

**Weaknesses:**
- **Same as React** for exposed API surface and dependency supply chain (though the surface is smaller).
- **`{@html}` is the equivalent of `dangerouslySetInnerHTML`.** Same risk for Markdown rendering.

### Verdict: Security

| Criterion | HTMX | React/Next | SvelteKit |
|---|---|---|---|
| XSS default protection | Excellent | Very good | Very good |
| Supply chain risk | Minimal (0 npm deps) | High (hundreds of deps) | Moderate (fewer deps) |
| API surface exposure | Low (HTML responses) | High (full JSON API visible) | High (full JSON API visible) |
| CSP compatibility | Excellent | Moderate (may need exceptions) | Good |
| CSRF risk | Moderate (needs manual tokens) | Low (JSON API + Bearer auth) | Low (JSON API + Bearer auth) |
| Dependency audit burden | None | High | Moderate |

**For a government security platform,** the supply chain risk of npm is a genuine concern, not a theoretical one. NCSC itself has published guidance on [supply chain security](https://www.ncsc.gov.uk/collection/supply-chain-security). The HTMX approach eliminates this entire category of risk.

---

## 4. Performance

| Criterion | HTMX | React/Next | SvelteKit |
|---|---|---|---|
| Initial page load | Fast (small JS) | Slower (80-120KB+ framework) | Fast (30-50KB compiled) |
| Navigation speed | Server round-trip per click | Instant (client-side routing) | Instant (client-side routing) |
| Perceived performance | Good with indicators | Excellent with optimistic UI | Excellent |
| Server load per request | Higher (renders HTML) | Lower (returns JSON) | Lower (returns JSON) |
| Works on slow connections | Best (minimal JS, progressive) | Worst (needs full bundle) | Good (small bundle) |

For Fresnel's user base (government analysts on reliable networks), performance differences are unlikely to be perceptible. The server is on the same network as the users.

---

## 5. Maintenance and Longevity

| Criterion | HTMX | React/Next | SvelteKit |
|---|---|---|---|
| Framework churn | Minimal (stable, simple API) | High (React 18→19, Next 13→14→15, App Router migration) | Moderate (Svelte 4→5 runes transition) |
| Go backend coupling | Tight (templates in Go) | Loose (JSON API only) | Loose (JSON API only) |
| Team skill requirements | Go developers only | Go + TypeScript/React developers | Go + TypeScript/Svelte developers |
| Hiring pool | Any backend developer | Large | Growing |
| Long-term support | Indefinite (HTML is HTML) | React is backed by Meta | Svelte is community-driven |

---

## 6. Fit for Fresnel Specifically

Fresnel has some specific characteristics that influence the choice:

1. **Government context.** Security posture matters more than cutting-edge UI. Supply chain risk is a first-order concern.
2. **Small team.** Likely 1-3 developers. Maintaining two toolchains (Go + Node.js) doubles the operational surface.
3. **Admin-heavy UI.** The application is mostly tables, forms, status trees, and audit logs — not a consumer-facing product that needs animations and delight.
4. **Real-time is limited.** The dashboard auto-refreshes every 60 seconds. There's no chat, no collaborative editing, no live cursors.
5. **Extensibility.** Starlark formulas and federation will add complexity later, but these are backend concerns, not frontend.

### Where HTMX Falls Short for Fresnel

The pain points we've already experienced are real:
- **Template–struct mismatches** cause runtime errors, not compile errors. This has been the #1 source of bugs so far.
- **No component abstraction.** Badges, filter bars, and pagination are copy-pasted across templates rather than composed from reusable units with typed props.
- **Testing templates is hard.** There's no static analysis for Go templates. Every field reference is a potential runtime bomb.

### The Case for Staying with HTMX

Despite those pain points:
- **Zero npm dependencies** eliminates supply chain risk entirely.
- **One language, one build** keeps operational complexity minimal.
- **The ceiling hasn't been reached.** Fresnel doesn't need drag-and-drop, real-time collaboration, or offline mode. Everything in the current requirements can be built with HTMX.
- **Switching cost is high.** Rewriting 32 templates as React/Svelte components, building a build pipeline, and restructuring the API for pure JSON is weeks of work that doesn't add features.

### The Case for Switching

If the project's scope grows to include:
- Rich Markdown editor with live preview
- Drag-and-drop event correlation builder
- Real-time multi-user dashboards
- Complex multi-step form wizards with client-side validation
- Offline capability for field analysts

Then the HTMX approach will become increasingly painful, and SvelteKit would be the recommended alternative (smaller bundle than React, simpler mental model, fewer dependencies, good CSP story).

---

## 7. Recommendation

**Stay with HTMX for now.** The switching cost doesn't justify the marginal UX improvement for Fresnel's current requirements. Instead, invest in mitigating HTMX's weaknesses:

1. **Add a template linter.** Write a Go test that parses all templates with the actual funcMap and a mock data struct to catch field mismatches at `go test` time, not at runtime.
2. **Consider [templ](https://templ.guide/)** as a replacement for `html/template`. `templ` is a Go templating language that generates type-safe Go code from `.templ` files. It catches field mismatches at compile time, supports components with typed props, and integrates with HTMX idiomatically. This gives you HTMX's architecture benefits with React-like type safety. This is the highest-leverage improvement available.
3. **Add CSRF tokens** to all mutating HTMX requests.
4. **Use Alpine.js** (3 KB) alongside HTMX for the cases that need client-side interactivity (autocomplete, inline editing, conditional form fields).

**Revisit this decision** if the requirements expand to include real-time collaboration, offline support, or consumer-grade UI polish. At that point, SvelteKit (not React) would be the recommended migration path due to its smaller bundle, simpler model, and lower supply chain risk.
