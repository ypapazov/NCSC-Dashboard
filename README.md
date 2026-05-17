# Fresnel

A **cyber situational awareness platform** for tracking and sharing operational status across a hierarchical structure of sectors and organizations. Named after the Fresnel lens that lets a lighthouse cast light further and more efficiently.

Fresnel serves decision-makers — people in cyber governance and organizational impact assessment who need to understand *the situation*, not the technical details. It is not a threat intelligence platform, SIEM, or SOAR tool. It is the dashboard layer above all of those.

---

## Current State

The application runs end-to-end in Docker Compose. A user can authenticate via Keycloak (OIDC + PKCE), see the hierarchical dashboard, navigate between sections, and perform CRUD on events, status reports, campaigns, organizations, sectors, and users. Content negotiation serves both HTML (HTMX fragments) and JSON from the same endpoints.

| Area | Status |
|------|--------|
| Authentication (Keycloak OIDC + PKCE) | Working |
| Authorization (Go-native role matrix) | Working (Cedar policy files planned) |
| Dashboard (hierarchical status tree) | Working (weighted-average formula, no custom Starlark yet) |
| Events CRUD + revisions + updates | Working (detail views need enrichment) |
| Status Reports, Campaigns, Correlations | List views working; detail/form views partially broken |
| Nudge/escalation email scheduler | Running (15-min tick; needs SMTP configured) |
| ClamAV attachment scanning | Client initialized; upload flow present |
| Audit logging | All mutations logged |
| i18n | Bulgarian and English strings present |
| Automated tests | **None** |
| Frontend | HTMX + Go templ (migrated from html/template) |

### Known Gaps

- **Detail and form views** reference fields not yet populated by handlers (will 500 on click-through).
- **Authorization** is enforced but not validated end-to-end across all 8 test roles.
- **Starlark formulas** are stubbed; dashboard uses a hardcoded weighted-average.
- **No Markdown editor** — plain `<textarea>` only.
- **Org context switcher** calls a non-existent endpoint.
- Zero `*_test.go` files exist.

---

## Architecture

```
Browser
  ├── keycloak-js (OIDC PKCE) ──→ Keycloak (port 8081)
  └── HTMX / fetch ──→ nginx (443, TLS)
                          └──→ fresnel (8080)
                                 ├── Middleware: logging → OIDC → cedar gate → content neg
                                 ├── Handlers → templ components (HTML) or JSON
                                 ├── Service layer (Cedar authorizer, Markdown renderer, ClamAV, SMTP)
                                 └── Postgres stores → PostgreSQL 16 + pgvector
```

**Key design decisions**:
- **Single Go binary**, no microservices. Three-layer architecture: HTTP → Service → Storage.
- **Cedar-inspired authorization** with two tiers: coarse gate at HTTP boundary, row-level filtering in the service layer. Currently a Go role-matrix; migration to real Cedar policy files is planned.
- **HTMX + templ** for the frontend. Server-rendered HTML fragments with type-safe Go templates. Zero npm dependencies.
- **Keycloak** as a black-box identity provider. The server is a pure resource server (validates JWTs, no cookies, no sessions).
- **PostgreSQL** with three logical schemas: `fresnel` (domain), `fresnel_iam` (policies/roles), `fresnel_audit` (append-only).

---

## Quick Start (Development)

### Prerequisites

- Docker Engine + Docker Compose v2
- Go 1.23+ (for local development outside Docker)
- `templ` CLI: `go install github.com/a-h/templ/cmd/templ@latest`

### Run

```bash
# Generate TLS certs and start the stack
make compose-up

# Or manually:
make certs
cd deploy && docker compose up --build -d
```

Services started: PostgreSQL (pgvector), Keycloak, ClamAV, Fresnel API, nginx.

Migrations run automatically on Fresnel startup.

### Access

- **Application**: https://localhost (accept self-signed cert)
- **Keycloak admin**: http://localhost:8081 (admin / admin)

### Test Users

All passwords: `Fresnel_Test_1!`

| Username | Role |
|----------|------|
| `platform-root` | Platform Root (full access) |
| `gov-sector-root` | Government Sector Root |
| `fed-sector-root` | Federal Sector Root |
| `orgA-root` | Org A Root |
| `orgA-admin` | Org A Admin |
| `orgA-contributor` | Org A Contributor |
| `orgA-viewer` | Org A Viewer |
| `orgB-root` | Org B Root |

### Build Locally

```bash
make generate   # templ generate
make build      # go build
make run        # run with env vars
```

---

## Project Layout

```
fresnel/
├── cmd/fresnel/main.go          # Entry point
├── internal/
│   ├── authz/                   # Authorization (Cedar-style role matrix)
│   ├── config/                  # Environment-based configuration
│   ├── domain/                  # Pure domain types (AuthContext)
│   ├── httpserver/              # Server, router, middleware, handlers
│   ├── i18n/                    # Bulgarian + English translations
│   ├── keycloak/                # Keycloak admin API client (user provisioning)
│   ├── mail/                    # SMTP + SES mailer
│   ├── markdown/                # goldmark + bluemonday rendering
│   ├── oauth/                   # JWKS cache + JWT verification
│   ├── service/                 # Business logic (13 services)
│   ├── storage/postgres/        # PostgreSQL stores (15 stores)
│   └── views/                   # templ components (type-safe HTML)
├── migrations/                  # SQL migrations (001–010)
├── static/                      # CSS, vendored JS (HTMX, keycloak-js)
├── deploy/                      # Docker Compose, nginx, Keycloak realm
├── infra/aws/                   # Terraform for AWS deployment
├── scripts/                     # deploy.sh, backup.sh, restore.sh
└── docs/                        # Architecture, requirements, guides
```

---

## Deployment

### Development

`make compose-up` — single-command local stack.

### Production (Docker Compose on VM)

```bash
cd deploy
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

See [`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md) for the full guide, including TLS certificates, Keycloak configuration, SMTP setup, and LUKS encryption.

### AWS (Terraform)

```bash
cd infra/aws
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars
terraform init && terraform apply
```

Creates EC2 + ALB + SES. See [`docs/OPERATIONS_GUIDE.md`](docs/OPERATIONS_GUIDE.md).

### Production Checklist

- [ ] Replace all default passwords (Keycloak, PostgreSQL, realm users)
- [ ] Real TLS certificates (not self-signed)
- [ ] LUKS encryption on data volumes
- [ ] Configure SMTP (or SES) for nudge emails
- [ ] Set `FRESNEL_SKIP_DEV_SEEDS=true` to skip test data
- [ ] Set up daily backups (`scripts/backup.sh` via cron)
- [ ] Verify security headers (`curl -skI https://your-domain/api/v1/health`)

---

## Domain Model

**Hierarchical**:
- **Status Reports** — assessed operational state of an org or sector at a point in time
- **Events** — specific incidents or disruptions with a lifecycle (OPEN → INVESTIGATING → MITIGATING → RESOLVED → CLOSED)
- **Event Updates** — chronological progress log on an event

**Cross-cutting**:
- **Campaigns** — group related events across sectors and organizations
- **Correlations** — links between events indicating relationships

**Organizational**:
- **Sectors** — recursive hierarchy (up to 5 levels deep)
- **Organizations** — attach to sectors, can belong to multiple sectors
- **Root Users** — one per scope (platform/sector/org) with maximum authority

**Sharing**: TLP v2.0 (RED, AMBER+STRICT, AMBER, GREEN, CLEAR) enforced via authorization policies.

---

## Configuration

| Variable | Required | Description |
|----------|----------|-------------|
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `KEYCLOAK_ISSUER` | Yes | OIDC issuer URL (server-side) |
| `KEYCLOAK_CLIENT_ID` | Yes | OIDC client ID |
| `KEYCLOAK_EXTERNAL_URL` | No | Browser-facing Keycloak URL |
| `APP_PUBLIC_URL` | No | Public base URL for email links |
| `SMTP_HOST` | No | SMTP relay (empty = no email) |
| `SES_REGION` | No | AWS SES region (overrides SMTP) |
| `CLAMAV_SOCKET` | No | ClamAV daemon address |
| `ATTACHMENT_DIR` | No | File attachment storage path |

See `.env.example` for the complete list.

---

## Documentation

| Document | What it covers |
|----------|---------------|
| [`FUTURE_WORK.md`](FUTURE_WORK.md) | Planned but unimplemented initiatives (Cedar PDP, Starlark, AI, federation, etc.) |
| [`docs/REQUIREMENTS.md`](docs/REQUIREMENTS.md) | Product definition, domain model, authorization, API design |
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | System architecture, layered design, data schema |
| [`docs/OPERATIONS_GUIDE.md`](docs/OPERATIONS_GUIDE.md) | AWS provisioning, deploys, backups, maintenance, migration discipline |
| [`docs/HOSTING_REQUIREMENTS.md`](docs/HOSTING_REQUIREMENTS.md) | VM specs, network, firewall |
| [`docs/TESTING.md`](docs/TESTING.md) | Manual test procedures |
| [`docs/SECURITY_HARDENING.md`](docs/SECURITY_HARDENING.md) | WAF, CSP, fail2ban, DB security (advisory) |
| [`docs/SECTORS.md`](docs/SECTORS.md) | NIS2 sector list (Bulgarian) |
| [`docs/AWS_TO_VSPHERE_MIGRATION.md`](docs/AWS_TO_VSPHERE_MIGRATION.md) | Moving from AWS to on-prem vSphere |
| `docs/TODO/` | Proposals: Cedar real policies, UI redesign, hierarchical tree view |

---

## License

Fresnel is released under the **MIT License**. See [`LICENSE.md`](LICENSE.md) for the full text.

Copyright © 2026 Yavor Papazov.
