#!/usr/bin/env bash
# 宝塔一键 docker run 部署
# 用法：
# curl -fsSL https://raw.githubusercontent.com/Ruoxi-TH/hyacine-server/master/scripts/baota-run.sh | bash

set -u

REPO_URL="${REPO_URL:-https://github.com/Ruoxi-TH/hyacine-server.git}"
INSTALL_DIR="${INSTALL_DIR:-/www/wwwroot/hyacine-server}"
API_PORT="${API_PORT:-3000}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-hyacine_$(date +%s | tail -c 8)}"
JWT_ACCESS_SECRET="${JWT_ACCESS_SECRET:-$(openssl rand -hex 32 2>/dev/null || echo hyacine_access_secret_change_me_32bytes)}"
JWT_REFRESH_SECRET="${JWT_REFRESH_SECRET:-$(openssl rand -hex 32 2>/dev/null || echo hyacine_refresh_secret_change_me_32bytes)}"
NETWORK="hyacine-net"
DATA_DIR="/www/wwwroot/hyacine-data"

log(){ printf '[hyacine] %s\n' "$*"; }
fail(){ printf '[hyacine] ERROR: %s\n' "$*" >&2; exit 1; }

need_docker(){
  command -v docker >/dev/null 2>&1 || fail "请先在宝塔安装 Docker 管理器"
  docker info >/dev/null 2>&1 || fail "Docker 未启动"
}

prepare_repo(){
  mkdir -p "$(dirname "$INSTALL_DIR")" "$DATA_DIR/postgres" "$DATA_DIR/redis"
  if [[ -d "$INSTALL_DIR/.git" ]]; then
    git -C "$INSTALL_DIR" pull --ff-only || true
  else
    command -v git >/dev/null 2>&1 || fail "缺少 git"
    rm -rf "$INSTALL_DIR"
    git clone "$REPO_URL" "$INSTALL_DIR"
  fi
  cd "$INSTALL_DIR" || fail "无法进入 $INSTALL_DIR"
}

ensure_network(){
  docker network inspect "$NETWORK" >/dev/null 2>&1 || docker network create "$NETWORK"
}

stop_old(){
  for c in hyacine-api hyacine-netease hyacine-postgres hyacine-redis; do
    docker rm -f "$c" >/dev/null 2>&1 || true
  done
}

run_postgres(){
  docker run -d \
    --name hyacine-postgres \
    --network "$NETWORK" \
    --restart unless-stopped \
    -e POSTGRES_DB=hyacine \
    -e POSTGRES_USER=hyacine \
    -e POSTGRES_PASSWORD="$POSTGRES_PASSWORD" \
    -p 5432:5432 \
    -v "$DATA_DIR/postgres:/var/lib/postgresql/data" \
    postgres:16-alpine
}

run_redis(){
  docker run -d \
    --name hyacine-redis \
    --network "$NETWORK" \
    --restart unless-stopped \
    -p 6379:6379 \
    -v "$DATA_DIR/redis:/data" \
    redis:7-alpine \
    redis-server --appendonly yes
}

run_netease(){
  docker run -d \
    --name hyacine-netease \
    --network "$NETWORK" \
    --restart unless-stopped \
    -e PORT=3000 \
    binaryify/netease_cloud_music_api:latest
}

build_and_run_api(){
  log "building api image..."
  docker build -t hyacine-server:latest "$INSTALL_DIR"

  docker run -d \
    --name hyacine-api \
    --network "$NETWORK" \
    --restart unless-stopped \
    -e NODE_ENV=production \
    -e PORT=3000 \
    -e DATABASE_URL="postgresql://hyacine:${POSTGRES_PASSWORD}@hyacine-postgres:5432/hyacine?schema=public" \
    -e REDIS_URL="redis://hyacine-redis:6379" \
    -e NETEASE_API_BASE="http://hyacine-netease:3000" \
    -e CORS_ORIGIN="*" \
    -e JWT_ACCESS_SECRET="$JWT_ACCESS_SECRET" \
    -e JWT_REFRESH_SECRET="$JWT_REFRESH_SECRET" \
    -e JWT_ACCESS_TTL=15m \
    -e JWT_REFRESH_TTL=30d \
    -p "${API_PORT}:3000" \
    hyacine-server:latest
}

wait_health(){
  local i
  for ((i=1;i<=60;i++)); do
    if curl -fsS "http://127.0.0.1:${API_PORT}/api/v1/health" >/dev/null 2>&1; then
      log "health ok"
      return 0
    fi
    sleep 2
  done
  log "health timeout，看日志: docker logs -f hyacine-api"
  return 1
}

main(){
  need_docker
  prepare_repo
  ensure_network
  stop_old

  log "start postgres"
  run_postgres
  sleep 3

  log "start redis"
  run_redis

  log "start netease"
  run_netease

  log "start api"
  build_and_run_api

  wait_health || true

  local ip
  ip="$(hostname -I 2>/dev/null | awk '{print $1}')"
  ip="${ip:-服务器IP}"

  cat > "$INSTALL_DIR/.env.runtime" <<EOF
POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
JWT_ACCESS_SECRET=${JWT_ACCESS_SECRET}
JWT_REFRESH_SECRET=${JWT_REFRESH_SECRET}
API_PORT=${API_PORT}
EOF
  chmod 600 "$INSTALL_DIR/.env.runtime" || true

  log "done"
  docker ps --filter name=hyacine-
  log "API: http://${ip}:${API_PORT}"
  log "health: http://${ip}:${API_PORT}/api/v1/health"
  log "手机端填: http://${ip}:${API_PORT}"
  log "密码已保存: ${INSTALL_DIR}/.env.runtime"
}

main "$@"
