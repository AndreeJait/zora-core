#!/usr/bin/env bash
set -euo pipefail

APP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$APP_DIR"

# ── Configuration ──────────────────────────────────────────────────────────────
REGISTRY_ENV="/home/andree/docker/.env"
COMPOSE_FILE="/home/andree/docker/zora-core/docker-compose.yaml"
IMAGE="ghcr.io/andreejait/zora-core"
TAG="${1:-latest}"

echo "[deploy] Deploying ${IMAGE}:${TAG}"

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

# ── Wait for app to be ready ──────────────────────────────────────────────────
echo "[deploy] Waiting for containers to start..."
sleep 5

# ── Run migrations ────────────────────────────────────────────────────────────
echo "[deploy] Running migrations..."
docker compose -f "$COMPOSE_FILE" exec zora-core /app/migrate up

# ── Cleanup ───────────────────────────────────────────────────────────────────
echo "[deploy] Cleaning up old images..."
docker image prune -f >/dev/null 2>&1 || true

echo "[deploy] Done. ${IMAGE}:${TAG} is live."
