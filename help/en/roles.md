# Roles & Permissions

Fresnel uses a role-based access control system. Each user has one or more roles, and each role is scoped to a specific level of the platform hierarchy.

## Available roles

| Role | Scope | Description |
|------|-------|-------------|
| **Platform Root** | Platform | Full platform administrator. Can manage all sectors, organizations, users, and content. Bypasses TLP. |
| **Sector Root** | Sector | Administers a specific sector and all organizations within it. Bypasses TLP for resources in their sector. |
| **Org Root** | Organization | Full control over a single organization. Bypasses TLP for resources in their organization. |
| **Org Admin** | Organization | Manages members and content within their organization. Cannot assign roles. Bypasses TLP within their org. |
| **Content Admin** | Platform | Can view and edit events and status reports across all organizations. Cannot manage users or roles. |
| **Contributor** | Organization | Can create new events and edit their own events within their organization. Subject to TLP. |
| **Viewer** | Organization or Sector | Read-only access to events and reports. Subject to TLP restrictions. |
| **Liaison** | Organization | Cross-organization coordination role. Can view events in assigned organizations. Subject to TLP. |

## Scope types

Each role is assigned at a specific scope:

- **Platform** — Applies to the entire platform (Platform Root, Content Admin)
- **Sector** — Applies to a specific sector and its sub-sectors (Sector Root; Viewer may also use this)
- **Organization** — Applies to a single organization (Org Root, Org Admin, Contributor, Viewer, Liaison)

When assigning a role, the scope type is determined automatically by the role you select. For sector and org-scoped roles, you must also select the specific sector or organization.

## Permission matrix

| Action | Platform Root | Sector Root | Org Root | Org Admin | Content Admin | Contributor | Viewer | Liaison |
|--------|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| View events | All | Sector | Org | Org | All | Org (TLP) | Org/Sector (TLP) | Org (TLP) |
| Create events | All | Sector | Org | Org | — | Org | — | — |
| Edit events | All | Sector | Org | Org | All | Own only | — | — |
| Delete events | All | Sector | Org | — | — | — | — | — |
| Manage members | All | Sector | Org | Org | — | — | — | — |
| Manage roles | All | Sector | Org | — | — | — | — | — |
| View audit log | All | Sector | Org | — | — | — | — | — |

## Root designations

The three "Root" roles (Platform Root, Sector Root, Org Root) carry a special **root designation** that provides override capabilities. When you assign a Root role through the admin UI, the root designation is created automatically.

## Multiple roles

A user can hold multiple roles simultaneously. For example, a user might be a Contributor in their primary organization and a Liaison in another. The system evaluates all roles — if any role grants access, the action is allowed.
