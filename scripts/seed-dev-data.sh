#!/usr/bin/env bash
set -euo pipefail
echo "Dev seed: migration 007 inserts sector/org/user (email dev@fresnel.local)." >&2
echo "Create the same user in Keycloak (Admin Console) with that email; OIDC links keycloak_sub on first login." >&2
echo "Optional: psql \"\$DATABASE_URL\" -f scripts/sql/ (custom SQL)." >&2
exit 0
