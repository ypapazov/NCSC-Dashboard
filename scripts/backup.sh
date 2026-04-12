#!/usr/bin/env bash
# Fresnel backup script — PostgreSQL dump + attachment snapshot + Keycloak realm.
# Outputs are GPG-encrypted if FRESNEL_BACKUP_GPG_RECIPIENT is set.
#
# Usage: ./scripts/backup.sh [--label TAG]
#   --label TAG   Optional label appended to the backup directory name.
#
# Environment:
#   FRESNEL_BACKUP_DIR             Base directory for backups (default: /data/backups)
#   FRESNEL_BACKUP_GPG_RECIPIENT   GPG recipient for encryption (default: unset = no encryption)
#   FRESNEL_BACKUP_RETAIN_DAYS     Days to retain old backups (default: 30)
#   FRESNEL_ATTACHMENT_DIR         Path to attachment files (default: /data/attachments)
#   DATABASE_URL                   Postgres connection string (read from .env if not set)
#
# See docs/OPERATIONS_GUIDE.md for setup and cron configuration.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_BASE="$REPO_ROOT/deploy/docker-compose.yml"
COMPOSE_PROD="$REPO_ROOT/deploy/docker-compose.prod.yml"

if mountpoint -q /data 2>/dev/null && [ -f "$COMPOSE_PROD" ]; then
  COMPOSE_CMD="docker compose -f $COMPOSE_BASE -f $COMPOSE_PROD"
else
  COMPOSE_CMD="docker compose -f $COMPOSE_BASE"
fi

BACKUP_DIR="${FRESNEL_BACKUP_DIR:-/data/backups}"
ATTACHMENT_DIR="${FRESNEL_ATTACHMENT_DIR:-/data/attachments}"
GPG_RECIPIENT="${FRESNEL_BACKUP_GPG_RECIPIENT:-}"
RETAIN_DAYS="${FRESNEL_BACKUP_RETAIN_DAYS:-30}"
LABEL=""

for arg in "$@"; do
  case "$arg" in
    --label) shift; LABEL="-$1" ;;
    --label=*) LABEL="-${arg#*=}" ;;
  esac
done

TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
DEST="$BACKUP_DIR/$TIMESTAMP$LABEL"
mkdir -p "$DEST"

log() { echo "[$(date -Iseconds)] $*"; }

maybe_encrypt() {
  local src="$1"
  if [ -n "$GPG_RECIPIENT" ]; then
    gpg --encrypt --recipient "$GPG_RECIPIENT" --output "${src}.gpg" "$src"
    rm -f "$src"
    log "  Encrypted: ${src}.gpg"
  fi
}

# --- PostgreSQL dump ---
log "Dumping PostgreSQL..."
$COMPOSE_CMD exec -T postgres \
  pg_dump -U fresnel -Fc fresnel > "$DEST/fresnel.dump"
maybe_encrypt "$DEST/fresnel.dump"

# --- Keycloak realm export ---
log "Exporting Keycloak realm..."
# Try the Admin REST API first; fall back to copying the import JSON.
KC_URL="${KEYCLOAK_INTERNAL_URL:-http://localhost:8081}"
KC_ADMIN="${KC_BOOTSTRAP_ADMIN_USERNAME:-admin}"
KC_PASS="${KC_BOOTSTRAP_ADMIN_PASSWORD:-admin}"

TOKEN=$(curl -sf -X POST "$KC_URL/realms/master/protocol/openid-connect/token" \
  -d "client_id=admin-cli" \
  -d "username=$KC_ADMIN" \
  -d "password=$KC_PASS" \
  -d "grant_type=password" 2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin).get('access_token',''))" 2>/dev/null || echo "")

if [ -n "$TOKEN" ]; then
  curl -sf -H "Authorization: Bearer $TOKEN" "$KC_URL/admin/realms/fresnel" \
    > "$DEST/keycloak-realm.json"
  log "  Realm exported via REST API."
else
  log "  WARN: Could not get Keycloak admin token. Copying import JSON as fallback."
  cp "$REPO_ROOT/deploy/keycloak/fresnel-realm.json" "$DEST/keycloak-realm.json"
fi
maybe_encrypt "$DEST/keycloak-realm.json"

# --- Attachment files ---
if [ -d "$ATTACHMENT_DIR" ] && [ "$(ls -A "$ATTACHMENT_DIR" 2>/dev/null)" ]; then
  log "Archiving attachments..."
  tar czf "$DEST/attachments.tar.gz" -C "$ATTACHMENT_DIR" .
  maybe_encrypt "$DEST/attachments.tar.gz"
else
  log "  No attachments to archive."
fi

# --- Cleanup old backups ---
if [ "$RETAIN_DAYS" -gt 0 ]; then
  log "Pruning backups older than $RETAIN_DAYS days..."
  find "$BACKUP_DIR" -maxdepth 1 -mindepth 1 -type d -mtime "+$RETAIN_DAYS" -exec rm -rf {} + 2>/dev/null || true
fi

log "Backup complete: $DEST"
ls -lh "$DEST"
