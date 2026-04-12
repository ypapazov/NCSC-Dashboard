#!/usr/bin/env bash
# Fresnel restore script — restores a backup created by backup.sh.
#
# Usage: ./scripts/restore.sh BACKUP_DIR
#   BACKUP_DIR   Path to a backup directory (e.g. /data/backups/20260410-020000)
#
# This script:
#   1. Stops the Fresnel API (prevents writes during restore)
#   2. Decrypts backup files if GPG-encrypted
#   3. Restores the PostgreSQL dump
#   4. Restores attachment files
#   5. Restarts the stack
#
# See docs/OPERATIONS_GUIDE.md for full restore procedure.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_BASE="$REPO_ROOT/deploy/docker-compose.yml"
COMPOSE_PROD="$REPO_ROOT/deploy/docker-compose.prod.yml"
ATTACHMENT_DIR="${FRESNEL_ATTACHMENT_DIR:-/data/attachments}"

if mountpoint -q /data 2>/dev/null && [ -f "$COMPOSE_PROD" ]; then
  COMPOSE_CMD="docker compose -f $COMPOSE_BASE -f $COMPOSE_PROD"
else
  COMPOSE_CMD="docker compose -f $COMPOSE_BASE"
fi

if [ $# -lt 1 ]; then
  echo "Usage: $0 BACKUP_DIR" >&2
  echo "  Example: $0 /data/backups/20260410-020000" >&2
  exit 1
fi

BACKUP="$1"
if [ ! -d "$BACKUP" ]; then
  echo "ERROR: Backup directory not found: $BACKUP" >&2
  exit 1
fi

log() { echo "[$(date -Iseconds)] $*"; }

maybe_decrypt() {
  local path="$1"
  if [ -f "${path}.gpg" ]; then
    gpg --decrypt --output "$path" "${path}.gpg"
    log "  Decrypted: $path"
  elif [ ! -f "$path" ]; then
    log "  WARN: File not found: $path (and no .gpg variant)"
    return 1
  fi
}

log "Restoring from: $BACKUP"

# 1. Stop the Fresnel API to prevent writes
log "Stopping Fresnel API..."
$COMPOSE_CMD stop fresnel

# 2. Ensure Postgres is running
$COMPOSE_CMD up -d postgres
log "Waiting for Postgres to be ready..."
for i in $(seq 1 30); do
  if $COMPOSE_CMD exec -T postgres pg_isready -U fresnel > /dev/null 2>&1; then
    break
  fi
  sleep 1
done

# 3. Restore PostgreSQL
log "Restoring PostgreSQL..."
DUMP_FILE="$BACKUP/fresnel.dump"
maybe_decrypt "$DUMP_FILE"
if [ -f "$DUMP_FILE" ]; then
  $COMPOSE_CMD exec -T postgres \
    pg_restore -U fresnel -d fresnel --clean --if-exists < "$DUMP_FILE"
  log "  PostgreSQL restored."
else
  log "  ERROR: No database dump found. Aborting."
  exit 1
fi

# 4. Restore attachments
ATT_FILE="$BACKUP/attachments.tar.gz"
maybe_decrypt "$ATT_FILE"
if [ -f "$ATT_FILE" ]; then
  log "Restoring attachments..."
  mkdir -p "$ATTACHMENT_DIR"
  tar xzf "$ATT_FILE" -C "$ATTACHMENT_DIR"
  log "  Attachments restored."
else
  log "  No attachment archive found (may be empty — OK)."
fi

# 5. Restart the stack
log "Starting full stack..."
$COMPOSE_CMD up -d

log "Waiting for health check..."
for i in $(seq 1 30); do
  if curl -sf http://localhost:8080/api/v1/health > /dev/null 2>&1; then
    log "Health check passed. Restore complete."
    exit 0
  fi
  sleep 1
done

log "WARN: Health check did not pass within 30 seconds. Check logs."
$COMPOSE_CMD logs --tail 50 fresnel
exit 1
