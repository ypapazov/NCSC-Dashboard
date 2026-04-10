# Proposal: Replace Go Role Matrix with Real Cedar Policies

**Status:** Draft  
**Effort:** 3–4 days  
**Prerequisite:** None (can be done independently)  
**Risk:** Low — the `Authorizer` interface is already clean; this replaces one implementation with another behind the same interface.

---

## Problem

The authorization system is named "Cedar" but does not use Cedar. The `cedar-go` library is in `go.mod` but no source file imports it. The entire authorization logic is a hand-written Go `switch` statement in `internal/authz/cedar.go` (218 lines).

This has three consequences:

1. **No externalized policies.** The permission model is embedded in Go code. A security auditor must read Go to understand who can do what. This defeats Cedar's core value proposition: policies as a separate, auditable artifact.
2. **No policy evolution without recompilation.** Adding a new role, changing TLP visibility rules, or adjusting scope semantics requires a code change, a rebuild, and a redeployment.
3. **No Cedar tooling.** Cedar has a CLI for validation, a test harness for policy evaluation, and a formal model for reasoning about policy coverage. None of this is usable with a Go switch statement.

---

## Current Implementation

```
internal/authz/
├── authorizer.go   # Authorizer interface, Action constants, Resource struct, FilterAuthorized[T]
├── cedar.go        # CedarAuthorizer: Go switch-based role matrix (218 lines)
└── resource.go     # Domain-to-Resource converters (EventResource, SectorResource, etc.)
```

The `Authorizer` interface is clean:

```go
type Authorizer interface {
    Authorize(ctx context.Context, auth *domain.AuthContext, action Action, res *Resource) bool
}
```

Every service calls this interface. No service depends on `CedarAuthorizer` directly. This means the implementation can be swapped without touching any caller.

### Role Matrix (Current)

| Role | Scope | view | create | edit | delete | manage_members | manage_roles | link | view_audit |
|---|---|---|---|---|---|---|---|---|---|
| PLATFORM_ROOT | * | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| SECTOR_ROOT | sector subtree | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| ORG_ROOT | own org | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| ORG_ADMIN | own org | ✓ | ✓ | ✓ | ✓ | ✓ | ✗ | ✓ | ✓ |
| CONTENT_ADMIN | Event/StatusReport only | ✓ | ✗ | ✓ | ✗ | ✗ | ✗ | ✗ | ✗ |
| CONTRIBUTOR | scoped + TLP | ✓ | Event only | own only | ✗ | ✗ | ✗ | ✓ | ✗ |
| VIEWER | scoped + TLP | ✓ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| LIAISON | scoped + TLP | ✓ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |

TLP overlay (applied to CONTRIBUTOR, VIEWER, LIAISON):
- CLEAR/GREEN: visible to all
- AMBER: visible to org members + sector roots above
- AMBER:STRICT: visible to org members only
- RED: visible to explicit recipient list only

---

## Proposed Implementation

### Step 1: Write Cedar Policy Files

Create `policies/` directory with `.cedar` files:

```
policies/
├── platform_root.cedar
├── sector_root.cedar
├── org_root.cedar
├── org_admin.cedar
├── content_admin.cedar
├── contributor.cedar
├── viewer.cedar
├── liaison.cedar
└── tlp.cedar           # TLP forbid rules (deny overrides)
```

Example (`contributor.cedar`):

```cedar
permit (
    principal is Fresnel::User,
    action == Fresnel::Action::"view",
    resource
)
when {
    principal.roles.contains(Fresnel::Role::"CONTRIBUTOR") &&
    resource.scope in principal.scope &&
    resource.tlp_allows(principal)
};

permit (
    principal is Fresnel::User,
    action == Fresnel::Action::"create",
    resource is Fresnel::Event
)
when {
    principal.roles.contains(Fresnel::Role::"CONTRIBUTOR") &&
    resource.scope in principal.scope
};

permit (
    principal is Fresnel::User,
    action == Fresnel::Action::"edit",
    resource is Fresnel::Event
)
when {
    principal.roles.contains(Fresnel::Role::"CONTRIBUTOR") &&
    resource.submitter == principal &&
    resource.scope in principal.scope
};
```

### Step 2: Define Cedar Schema

Create `policies/schema.cedarschema`:

```
namespace Fresnel {
    entity User = {
        roles: Set<Role>,
        scope: Set<Scope>,
        org_memberships: Set<Organization>,
    };
    entity Event = {
        scope: Scope,
        org: Organization,
        submitter: User,
        tlp: String,
    };
    // ... etc
}
```

### Step 3: Replace CedarAuthorizer

Create a new `internal/authz/cedar_policy.go` that:

1. Embeds the `.cedar` files via `//go:embed policies/*.cedar`
2. On startup, parses all policies into a `cedar.PolicySet`
3. On each `Authorize()` call, converts the Go `AuthContext` + `Resource` into Cedar entities and evaluates against the policy set
4. Returns `true` if the policy set produces `Allow` and no `Deny`

The `Authorizer` interface stays the same. The constructor changes:

```go
// Before
az := authz.NewCedarAuthorizer(sectorAncestryFunc)

// After
az, err := authz.NewPolicyCedarAuthorizer(sectorAncestryFunc)
```

### Step 4: Validate Equivalence

Write a table-driven Go test that exercises every cell in the role matrix above with both the old `CedarAuthorizer` and the new `PolicyCedarAuthorizer`, asserting identical results. This is the safety net.

### Step 5: Remove Old Implementation

Once tests pass, delete `cedar.go` (the Go switch-based implementation). Keep `authorizer.go` and `resource.go` unchanged.

---

## Open Questions

1. **cedar-go maturity.** As of v1.6.0, `cedar-go` supports the core Cedar language but not policy templates or partial evaluation. The policies above use basic `permit`/`forbid` with `when` clauses, which are fully supported. Verify with a spike before committing.

2. **Sector ancestry in Cedar.** The current Go code does `strings.HasPrefix(res.SectorAncestry, rolePath)` for hierarchy checks. In Cedar, this would use the `like` operator or entity parent relationships. Need to determine which cedar-go supports.

3. **Performance.** The Go switch evaluates in nanoseconds. The Cedar policy evaluator adds overhead per evaluation (entity construction, policy iteration). For Fresnel's scale (tens of users), this is irrelevant. But if `FilterAuthorized` is called on lists of 1000+ items, benchmark it.

4. **Policy hot-reload.** The proposal embeds policies at compile time. A future enhancement could load policies from the database or filesystem, enabling runtime policy updates without redeploy. This is explicitly deferred.

---

## Files Changed

| File | Action |
|---|---|
| `policies/*.cedar` (8-9 files) | New |
| `policies/schema.cedarschema` | New |
| `policies/embed.go` | New (`//go:embed`) |
| `internal/authz/cedar_policy.go` | New (replaces `cedar.go`) |
| `internal/authz/cedar.go` | Delete |
| `internal/authz/cedar_test.go` | New (equivalence tests) |
| `cmd/fresnel/main.go` | Change constructor call (1 line) |
| `go.mod` | `cedar-go` moves from `indirect` to direct |

No changes to: services, handlers, middleware, templates, domain types, storage.

---

## Acceptance Criteria

- [ ] All `.cedar` files validate with `cedar validate --schema schema.cedarschema`
- [ ] Equivalence test passes for all 8 roles × 8 actions × 6 resource types × 5 TLP levels
- [ ] `go build ./...` passes
- [ ] Manual smoke test: log in as each of the 8 test users and verify the same access as before
- [ ] The Go switch-based `cedar.go` is deleted
- [ ] `cedar-go` is a direct (not indirect) dependency
