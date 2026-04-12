# Zero-Downtime Deploy Discipline

**Scope**: How to update Fresnel in production with minimal service interruption.
**Audience**: Anyone deploying Fresnel or writing database migrations.

---

## Acceptable downtime target

A few minutes of downtime per deploy is acceptable. The goal is not sub-second rolling deploys — it is to avoid situations where an upgrade breaks the running system or requires a long unplanned outage to recover from.

---

## Database migrations: expand-then-contract

The Fresnel API runs migrations on startup (`postgres.Migrate`). This means the new binary's migration runs while the old binary may still be serving (briefly) or will need to serve if you roll back.

**The rule: every migration must be backwards-compatible with the previous release's Go code.**

### What this means in practice

| Change type | Safe in one release? | Correct approach |
|---|---|---|
| Add a new table | Yes | New code uses it, old code ignores it |
| Add a nullable column | Yes | Old code ignores the column |
| Add a NOT NULL column with a default | Yes (Postgres backfills) | Old code ignores it |
| Rename a column | **No** | Release N: add new column + dual-write. Release N+1: read from new column only. Release N+2: drop old column |
| Drop a column | **No** | Release N: stop reading/writing the column in Go code. Release N+1: drop the column in migration |
| Change a column type | **No** | Same expand-then-contract as rename |
| Add a CHECK or FK constraint | Usually safe | Ensure existing data satisfies the constraint before adding it |

### The three-release pattern

When a migration is not backwards-compatible, split it across releases:

1. **Release N (expand)**: Add the new structure (column, table, index). Both old and new code work.
2. **Release N+1 (migrate)**: New code writes to both old and new structures, reads from the new one. Backfill existing data.
3. **Release N+2 (contract)**: Remove the old structure. Only new code exists.

For a PoC iterating over weeks, releases N and N+1 can often be the same deploy — just make sure the migration file runs before the new Go code starts serving, which it does by design (migrations run at startup before the HTTP server binds).

The critical rule is: **never drop or rename something in the same release that changes the Go code depending on it.** If you need to roll back the Go binary, the old code must still work against the migrated database.

---

## Deploy procedure (single-host Docker Compose)

### Application-only update (most common)

This is when you're updating the Fresnel API binary but not Postgres, Keycloak, or nginx.

```bash
# Pull or build the new image
docker compose build fresnel
# OR: docker compose pull fresnel (if using a registry)

# Restart only the Fresnel service
docker compose up -d --no-deps fresnel
```

What happens:
1. Compose stops the old `fresnel` container.
2. Starts the new one.
3. The new binary runs migrations (if any), then binds the HTTP port.
4. nginx sees the upstream come back and starts routing to it.

**Downtime**: Typically 2–10 seconds — the time between the old container stopping and the new one passing its health check. The Go binary starts fast (< 1s); the rest is container lifecycle overhead.

If a migration is slow (large data backfill), the downtime extends by the migration duration. For large data migrations, consider running them as a pre-deploy step:

```bash
# Run migrations without starting the server
docker compose run --rm fresnel /app/fresnel migrate

# Then restart the service (migrations are idempotent / already applied)
docker compose up -d --no-deps fresnel
```

### Full stack update

When upgrading Postgres, Keycloak, or nginx versions:

```bash
# Stop everything
docker compose down

# Pull new images
docker compose pull

# Start (Postgres first, then the rest via depends_on)
docker compose up -d
```

**Downtime**: 1–3 minutes depending on Keycloak startup time and whether Postgres needs recovery.

### Rollback

If the new version is broken:

```bash
# Tag or note the previous image before deploying
# Then:
docker compose up -d --no-deps fresnel  # with the old image tag
```

If a migration was applied that the old code can't handle, you have a problem — which is why the expand-then-contract discipline matters. Additive migrations (new columns, new tables) are always safe to leave in place when rolling back.

---

## Keycloak changes

Keycloak is slow to restart (~15–30 seconds). Over a period of weeks of active development, you should rarely need to restart it:

- **Adding users, updating client settings, changing realm config**: Use the Keycloak Admin REST API or Admin Console against the running instance. No restart needed.
- **Keycloak version upgrade**: Requires a restart. Schedule it with a deploy window.
- **Realm JSON re-import**: Only applies at first startup (`--import-realm`). For ongoing config, use the API.

The `start-dev` mode in the current Compose file imports the realm JSON only on first start. Subsequent restarts do not re-import (Keycloak detects the realm already exists). This means `fresnel-realm.json` changes require either: (a) the Admin API, (b) deleting the realm and restarting, or (c) switching to `kc.sh export/import` workflows.

For a PoC iterating over weeks, use the Admin Console for day-to-day changes and keep `fresnel-realm.json` as the canonical "clean start" definition.

---

## Summary

- Keep migrations additive. Never drop or rename in the same release as a code change.
- `docker compose up -d --no-deps fresnel` is your standard deploy. A few seconds of downtime.
- Full stack upgrades take a few minutes. Acceptable.
- Manage Keycloak via its Admin API, not restarts.
- If you need to run a slow migration, run it as a pre-deploy step with `fresnel migrate`.
