# Fresnel deployment guide

This document describes how to deploy the Fresnel API and its dependencies on a typical Linux server (Ubuntu 24.04 or equivalent) using Docker Compose, TLS in front of the app, and PostgreSQL.

## Prerequisites

- **Host OS**: Ubuntu 24.04 LTS (or another recent Linux with systemd and a supported kernel).
- **Docker Engine** and **Docker Compose plugin** (v2), installed per [Docker’s official instructions](https://docs.docker.com/engine/install/ubuntu/).
- **TLS certificates** for the hostname users and browsers will use (see below). Self-signed certificates are acceptable for lab use only.
- **Network**: Outbound access to pull container images; optionally outbound SMTP (port 25, 587, or 465) for email nudges.
- **Resources**: At least 2 vCPU, 4 GiB RAM, and sufficient disk for PostgreSQL data, ClamAV virus definitions, and attachment storage.

Optional but recommended:

- A dedicated **SMTP relay** on the local network for nudge and notification mail.
- **Off-host backups** with encryption at rest.

---

## Step-by-step deployment

### 1. Clone the repository

```bash
git clone <your-fresnel-repo-url> fresnel
cd fresnel
```

All paths below are relative to the repository root unless noted.

### 2. Configure environment

Compose and the Fresnel binary read settings from environment variables. For Docker Compose, set them in a `.env` file next to `deploy/docker-compose.yml`, or export them in your shell before `docker compose up`.

See [Environment variables reference](#environment-variables-reference) for the full list.

Minimum for a working stack (adjust hosts and secrets):

- `DATABASE_URL` — PostgreSQL DSN (Compose wires this to the bundled Postgres service).
- `KEYCLOAK_ISSUER` — Issuer URL **as seen by the Fresnel container** (e.g. `http://keycloak:8080/realms/fresnel`).
- `KEYCLOAK_CLIENT_ID` — OIDC client id (default in realm export: `fresnel-app`).
- `KEYCLOAK_EXTERNAL_URL` — Realm URL **as seen by the browser** (e.g. `https://auth.example.com/realms/fresnel`), used for token `iss` alignment and client configuration.

### 3. TLS certificates

The sample **nginx** service terminates HTTPS and proxies to Fresnel on port 8080.

1. Create a directory `deploy/nginx/certs/` on the host.
2. Place PEM files there:
   - `server.crt` — leaf certificate (full chain recommended concatenated in the PEM if your nginx setup expects it).
   - `server.key` — private key (restrict permissions, e.g. `chmod 600`).

**Development / lab**: generate a self-signed certificate:

```bash
mkdir -p deploy/nginx/certs
openssl req -x509 -newkey rsa:4096 -keyout deploy/nginx/certs/server.key \
  -out deploy/nginx/certs/server.crt -days 365 -nodes \
  -subj "/CN=localhost"
```

**Production**: use certificates from your CA or ACME (Let’s Encrypt). Update `server_name` in `deploy/nginx/nginx.conf` to match the certificate CN/SAN.

### 4. Keycloak realm and users

1. Ensure `deploy/keycloak/fresnel-realm.json` is mounted into the Keycloak container (already configured in `deploy/docker-compose.yml` with `--import-realm`).
2. Start Keycloak once and confirm the **fresnel** realm appears in the admin UI (bootstrap admin uses `KC_BOOTSTRAP_ADMIN_*` in Compose — that account is for the **master** realm admin console, not Fresnel application users).
3. **Production**: change every password from the JSON import; prefer creating users via Admin Console or API rather than committing secrets to git.
4. Align **email addresses** between Keycloak users and rows in `fresnel.users` — Fresnel links OIDC identities to application users by email on first login. Migration `008_m1_platform_admin.sql` seeds `admin@fresnel.local` for the platform administrator; the bundled realm defines a matching user (`platform-root` in `fresnel-realm.json`).

If realm import does not create users for your Keycloak version, create them manually and assign the same emails as in your database seed or admin procedures.

### 5. Docker Compose

From the `deploy/` directory:

```bash
cd deploy
docker compose build
docker compose up -d
```

Services:

- **postgres** — PostgreSQL 16 with pgvector image.
- **keycloak** — Identity provider (dev mode in the sample Compose file).
- **clamav** — Virus scanning daemon (optional socket wiring depends on your Fresnel `CLAMAV_SOCKET` setting).
- **fresnel** — Go API (runs migrations on startup unless you use the separate migrate command).
- **nginx** — TLS reverse proxy to Fresnel.

### 6. Run migrations

The application applies SQL migrations from `migrations/` automatically when the **fresnel** container starts (`postgres.Migrate` in `cmd/fresnel/main.go`).

To run migrations **only** (e.g. from CI or an admin job):

```bash
docker compose run --rm fresnel /app/fresnel migrate
```

Ensure `DATABASE_URL` is set for that one-off container (Compose passes through the same env as the long-running service).

Migration files (apply in lexical order by the migrator):

| File | Purpose (summary) |
|------|---------------------|
| `001_fresnel_schema.sql` | Core Fresnel schema (events, orgs, nudges, formulas, …) |
| `002_fresnel_iam_schema.sql` | IAM tables (Cedar-related role data) |
| `003_fresnel_audit_schema.sql` | Append-only audit log schema |
| `004a_pgvector_ext.sql` / `004b_pgvector_table.sql` | pgvector extension and embedding storage |
| `005_seed_platform_config.sql` | Platform configuration seed |
| `006_m1_email_unique.sql` | Email uniqueness constraint |
| `007_m1_dev_seed.sql` | Development seed data |
| `008_m1_platform_admin.sql` | Platform admin user row for OIDC linking |
| `009_full_dev_seed.sql` | Extended dev seed (if used in your environment) |

### 7. Create the first platform root user

After Keycloak and PostgreSQL are up:

1. Ensure a **Keycloak** user exists with the same **email** as the platform row you want (see `008_m1_platform_admin.sql` for the dev email `admin@fresnel.local`).
2. In Fresnel IAM tables, designate **platform root** for that user (via your admin API/UI or SQL against `fresnel_iam` per your operational runbook). The exact SQL or API call depends on how you manage root designations in your environment; the seed migrations may already assign org memberships for the platform admin user.

### 8. Verify the health endpoint

Through nginx (HTTPS):

```bash
curl -sk https://localhost/api/v1/health
```

Directly to the API container (if port 8080 is published):

```bash
curl -s http://localhost:8080/api/v1/health
```

A healthy deployment returns JSON indicating database and Keycloak issuer reachability (see `internal/httpserver/handlers/health.go` for the current shape).

---

## Production considerations

- **Replace dev passwords** everywhere: PostgreSQL, Keycloak bootstrap admin, imported realm users, and any default SMTP credentials.
- **Use real TLS certificates** and disable weak protocols; the sample nginx config enables TLS 1.2+.
- **Configure SMTP** (`SMTP_HOST`, `SMTP_PORT`, `SMTP_FROM`) so the nudge scheduler can send mail. Without `SMTP_HOST`, the mailer logs warnings and skips delivery.
- **ClamAV**: schedule or automate `freshclam` / image updates so virus definitions stay current; monitor daemon health if attachments are scanned.
- **Log rotation**: configure Docker logging drivers or logrotate for container logs; retain nginx access/error logs according to policy.
- **Backups**:
  - **PostgreSQL**: regular `pg_dump` / `pg_dumpall` or volume snapshots; test restores.
  - **Keycloak**: realm export from Admin Console or `kc.sh export` depending on deployment mode; store exports encrypted.
- **Monitoring**: poll `GET /api/v1/health` from your monitor; alert on non-200 or degraded JSON fields; track Postgres and Keycloak metrics separately.

---

## Environment variables reference

| Variable | Required | Description |
|----------|----------|-------------|
| `LISTEN_ADDR` | No | Address the Go server binds to (default `:8080`). |
| `DATABASE_URL` | Yes | PostgreSQL connection string (e.g. `postgres://user:pass@host:5432/fresnel?sslmode=require`). |
| `KEYCLOAK_ISSUER` | Yes | OIDC issuer URL used server-side (JWKS, validation). |
| `KEYCLOAK_CLIENT_ID` | Yes | Expected OAuth2/OIDC client id (`aud` / client checks). |
| `KEYCLOAK_EXTERNAL_URL` | No | Browser-facing realm URL; helps when internal and external issuer strings differ. |
| `APP_PUBLIC_URL` | No | Public base URL for links in emails and similar (default `https://localhost`). |
| `SMTP_HOST` | No | SMTP relay hostname; empty disables sending (logs only). |
| `SMTP_PORT` | No | SMTP port (default `587`). |
| `SMTP_FROM` | No | RFC 5322 From address (default in binary: `noreply@fresnel.local` if unset). |
| `CLAMAV_SOCKET` | No | Path to ClamAV daemon socket; empty may disable scanning depending on attachment code paths. |
| `ATTACHMENT_DIR` | No | Writable directory for uploaded attachment bytes. |
| `DASHBOARD_CACHE_TTL_SEC` | No | Dashboard computation cache TTL in seconds. |

`KEYCLOAK_TOKEN_URL` may be required in some setups for token refresh when the issuer seen by the browser differs from the URL the Fresnel container must use; see `deploy/keycloak/README.md`.

---

## Troubleshooting

| Symptom | Likely cause | What to check |
|---------|----------------|---------------|
| `401` / `403` on API | Token issuer or audience mismatch | `KEYCLOAK_ISSUER`, `KEYCLOAK_EXTERNAL_URL`, client id; JWT `iss` vs allowed issuers in config. |
| Database connection errors | Wrong DSN or Postgres not ready | `DATABASE_URL`, `docker compose ps`, Postgres logs. |
| Health check reports Keycloak down | JWKS URL unreachable from Fresnel container | Network from `fresnel` service to `KEYCLOAK_ISSUER` host; TLS/proxy. |
| No nudge emails | SMTP not configured or blocked | `SMTP_HOST`, firewall egress, relay logs; application logs for “email not sent”. |
| TLS handshake errors in browser | Cert/CN/SAN mismatch or self-signed trust | Certificate files under `deploy/nginx/certs/`, `server_name` in nginx. |
| Realm not imported | Keycloak volume or import flags | Keycloak logs; confirm `fresnel-realm.json` mount and `--import-realm`. |
| User exists in Keycloak but not in Fresnel | Missing DB row or email mismatch | `fresnel.users.email` vs Keycloak user email. |

For development-only issues with token refresh across Docker and the host browser, see `deploy/keycloak/README.md`.
