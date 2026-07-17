#!/usr/bin/env bash

set -u

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

log() {
  printf '[deploy] %s\n' "$*"
}

fail() {
  printf '[deploy] ERROR: %s\n' "$*" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "missing command: $1"
}

random_secret() {
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex 32
    return
  fi
  if command -v python3 >/dev/null 2>&1; then
    python3 - <<'PY'
import secrets
print(secrets.token_hex(32))
PY
    return
  fi
  # last resort
  date +%s%N | sha256sum | awk '{print $1}'
}

ensure_env() {
  if [[ ! -f .env ]]; then
    if [[ -f .env.deploy.example ]]; then
      cp .env.deploy.example .env
      log "created .env from .env.deploy.example"
    else
      fail ".env and .env.deploy.example both missing"
    fi
  fi

  # fill defaults if empty placeholders remain
  if grep -q 'replace-with-a-strong-database-password' .env; then
    local pw
    pw="$(random_secret | cut -c1-24)"
    sed -i "s/replace-with-a-strong-database-password/${pw}/" .env
    log "generated POSTGRES_PASSWORD"
  fi
  if grep -q 'replace-with-a-random-secret-at-least-32-characters' .env; then
    local s1
    s1="$(random_secret)"
    sed -i "0,/replace-with-a-random-secret-at-least-32-characters/{s/replace-with-a-random-secret-at-least-32-characters/${s1}/}" .env
    log "generated JWT_ACCESS_SECRET"
  fi
  if grep -q 'replace-with-a-different-random-secret-at-least-32-characters' .env; then
    local s2
    s2="$(random_secret)"
    sed -i "s/replace-with-a-different-random-secret-at-least-32-characters/${s2}/" .env
    log "generated JWT_REFRESH_SECRET"
  fi
  if grep -q 'https://your-web-client.example' .env; then
    sed -i 's|https://your-web-client.example|*|' .env
    log "set CORS_ORIGIN=*"
  fi
}

compose() {
  if docker compose version >/dev/null 2>&1; then
    docker compose "$@"
  elif command -v docker-compose >/dev/null 2>&1; then
    docker-compose "$@"
  else
    fail "docker compose not found"
  fi
}

wait_health() {
  local url="${1:-http://127.0.0.1:3000/api/v1/health}"
  local tries="${2:-40}"
  local i
  for ((i = 1; i <= tries; i++)); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      log "health check ok: $url"
      return 0
    fi
    sleep 2
  done
  fail "health check failed: $url"
}

main() {
  need_cmd docker
  need_cmd curl

  ensure_env
  log "building and starting containers..."
  compose up -d --build

  local port
  port="$(grep -E '^API_PORT=' .env | tail -n1 | cut -d= -f2-)"
  port="${port:-3000}"

  wait_health "http://127.0.0.1:${port}/api/v1/health"
  log "deploy finished"
  log "API: http://127.0.0.1:${port}"
  log "services: api / postgres / redis / netease"
  compose ps
}

main "$@"