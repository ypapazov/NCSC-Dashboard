#!/usr/bin/env bash
# Fresnel deploy script — pulls latest code, backs up, rebuilds, restarts.
# Usage: ./scripts/deploy.sh [--skip-backup]
#
# Designed to run on the production host from the repo root (/opt/fresnel).
# See docs/OPERATIONS_GUIDE.md for full context.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_BASE="$REPO_ROOT/deploy/docker-compose.yml"
COMPOSE_PROD="$REPO_ROOT/deploy/docker-compose.prod.yml"
BACKUP_DIR="${FRESNEL_BACKUP_DIR:-/data/backups}"
SKIP_BACKUP=false

# Use production overrides if /data is a mount point (LUKS volume present).
if mountpoint -q /data 2>/dev/null && [ -f "$COMPOSE_PROD" ]; then
  COMPOSE_CMD="docker compose -f $COMPOSE_BASE -f $COMPOSE_PROD"
else
  COMPOSE_CMD="docker compose -f $COMPOSE_BASE"
fi

for arg in "$@"; do
  case "$arg" in
    --skip-backup) SKIP_BACKUP=true ;;
    *) echo "Unknown argument: $arg" >&2; exit 1 ;;
  esac
done

log() { echo "[$(date -Iseconds)] $*"; }

cd "$REPO_ROOT"

# 1. Pull latest code
log "Pulling latest code..."
git pull --ff-only

# 2. Pre-deploy backup (unless skipped)
if [ "$SKIP_BACKUP" = false ]; then
  log "Running pre-deploy backup..."
  "$SCRIPT_DIR/backup.sh"
else
  log "Skipping pre-deploy backup (--skip-backup)"
fi

# 3. Build the new Fresnel image
log "Building Fresnel image..."
$COMPOSE_CMD build fresnel

# 4. Run migrations before swapping (minimises downtime)
log "Running migrations..."
$COMPOSE_CMD run --rm fresnel /app/fresnel migrate

# 5. Restart the Fresnel API and nginx containers
log "Restarting Fresnel API and nginx..."
$COMPOSE_CMD up -d --no-deps fresnel nginx

# 6. Wait for health check
log "Waiting for health check..."
for i in $(seq 1 30); do
  if curl -sf http://localhost:80/api/v1/health > /dev/null 2>&1; then
    log "Health check passed."
    break
  fi
  if [ "$i" -eq 30 ]; then
    log "ERROR: Health check failed after 30 seconds."
    exit 1
  fi
  sleep 1
done

log "Deploy complete."
