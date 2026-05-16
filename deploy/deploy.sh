#!/usr/bin/env bash
set -euo pipefail

APP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$APP_DIR"

# ── Configuration ──────────────────────────────────────────────────────────────
REGISTRY_ENV="/home/andree/docker/.env"
COMPOSE_FILE="/home/andree/docker/zora-core/docker-compose.yaml"
IMAGE="ghcr.io/andreejait/zora-core"
CONTAINER="zora-core"
TAG="${1:-latest}"

echo "[deploy] Deploying ${IMAGE}:${TAG}"

# ── Helpers ────────────────────────────────────────────────────────────────────
wait_for_container() {
  local container=$1
  local max_wait=${2:-60}
  local waited=0
  echo "[deploy] Waiting for ${container} to be running..."
  while [ $waited -lt $max_wait ]; do
    local status
    status=$(docker inspect -f '{{.State.Status}}' "${container}" 2>/dev/null || echo "missing")
    if [ "$status" = "running" ]; then
      echo "[deploy] ${container} is running."
      return 0
    fi
    echo "[deploy] ${container} status: ${status}, waiting..."
    sleep 2
    waited=$((waited + 2))
  done
  echo "[deploy] ERROR: ${container} did not become ready within ${max_wait}s."
  return 1
}

# ── Load registry credentials ─────────────────────────────────────────────────
if [ ! -f "$REGISTRY_ENV" ]; then
  echo "[deploy] ERROR: Registry env not found at $REGISTRY_ENV"
  exit 1
fi

set -a
source "$REGISTRY_ENV"
set +a

if [ -z "${REGISTRY_USER:-}" ] || [ -z "${REGISTRY_TOKEN:-}" ]; then
  echo "[deploy] ERROR: REGISTRY_USER or REGISTRY_TOKEN is empty in $REGISTRY_ENV"
  exit 1
fi

echo "[deploy] Logging in to GHCR as ${REGISTRY_USER}..."
echo "$REGISTRY_TOKEN" | docker login ghcr.io -u "$REGISTRY_USER" --password-stdin >/dev/null

# ── Pull latest image ──────────────────────────────────────────────────────────
echo "[deploy] Pulling ${IMAGE}:${TAG}..."
docker compose -f "$COMPOSE_FILE" pull

# ── Start services ────────────────────────────────────────────────────────────
echo "[deploy] Starting containers..."
docker compose -f "$COMPOSE_FILE" up -d --build

# ── Wait for app container to be ready ─────────────────────────────────────────
wait_for_container "$CONTAINER"

# ── Run migrations ────────────────────────────────────────────────────────────
echo "[deploy] Running migrations..."
docker compose -f "$COMPOSE_FILE" exec "$CONTAINER" /app/migrate up

# ── Cleanup ───────────────────────────────────────────────────────────────────
echo "[deploy] Cleaning up old images..."
docker image prune -f >/dev/null 2>&1 || true

echo "[deploy] Done. ${IMAGE}:${TAG} is live."
