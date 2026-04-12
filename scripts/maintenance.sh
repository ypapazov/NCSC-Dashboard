#!/usr/bin/env bash
# Toggle maintenance mode via nginx.
#
# Usage:
#   ./scripts/maintenance.sh on    — enable maintenance page, block all traffic
#   ./scripts/maintenance.sh off   — disable maintenance page, resume normal operation
#   ./scripts/maintenance.sh status — show current state
#
# Maintenance mode works by placing a flag file that nginx checks. When the
# flag exists, nginx returns 503 with a static maintenance page. The health
# endpoint remains accessible for monitoring.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_FILE="$REPO_ROOT/deploy/docker-compose.yml"
FLAG_CONTAINER_PATH="/etc/nginx/maintenance.flag"

nginx_exec() {
  docker compose -f "$COMPOSE_FILE" exec -T nginx "$@"
}

case "${1:-status}" in
  on)
    nginx_exec sh -c "touch $FLAG_CONTAINER_PATH"
    nginx_exec nginx -s reload
    echo "Maintenance mode ENABLED. Users see the maintenance page."
    ;;
  off)
    nginx_exec sh -c "rm -f $FLAG_CONTAINER_PATH"
    nginx_exec nginx -s reload
    echo "Maintenance mode DISABLED. Normal operation resumed."
    ;;
  status)
    if nginx_exec sh -c "test -f $FLAG_CONTAINER_PATH" 2>/dev/null; then
      echo "Maintenance mode is ON."
    else
      echo "Maintenance mode is OFF."
    fi
    ;;
  *)
    echo "Usage: $0 {on|off|status}" >&2
    exit 1
    ;;
esac
