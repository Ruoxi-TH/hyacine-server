#!/usr/bin/env bash
# 宝塔一键 docker run 部署 v3
# curl -fsSL https://raw.githubusercontent.com/Ruoxi-TH/hyacine-server/master/scripts/baota-run.sh | bash

set -u

SCRIPT_VERSION="2026-07-17-v3"
REPO_URL="${REPO_URL:-https://github.com/Ruoxi-TH/hyacine-server.git}"
INSTALL_DIR="${INSTALL_DIR:-/www/wwwroot/hyacine-server}"
API_PORT="${API_PORT:-3000}"
NETWORK="hyacine-net"
DATA_DIR="/www/wwwroot/hyacine-data"

# 镜像源候选（自动回退）
MIRRORS_DEFAULT="docker.1ms.run docker.m.daocloud.io dockerproxy.com"
MIRRORS="${MIRRORS:-$MIRRORS_DEFAULT}"

log(){ printf '[hyacine] %s\n' "$*"; }
fail(){ printf '[hyacine] ERROR: %s\n' "$*" >&2; exit 1; }

random_secret(){
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex 32
  else
    date +%s%N | sha256sum | awk '{print $1}'
  fi
}

POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-hyacine_$(date +%s | tail -c 8)}"
JWT_ACCESS_SECRET="${JWT_ACCESS_SECRET:-$(random_secret)}"
JWT_REFRESH_SECRET="${JWT_REFRESH_SECRET:-$(random_secret)}"

need_docker(){
  command -v docker >/dev/null 2>&1 || fail "请先在宝塔安装 Docker 管理器"
  docker info >/dev/null 2>&1 || fail "Docker 未启动"
}

prepare_repo(){
  mkdir -p "$(dirname "$INSTALL_DIR")" "$DATA_DIR/postgres" "$DATA_DIR/redis"
  if [[ -d "$INSTALL_DIR/.git" ]]; then
    log "update repo: $INSTALL_DIR"
    git -C "$INSTALL_DIR" fetch --all --prune || true
    git -C "$INSTALL_DIR" reset --hard origin/master || git -C "$INSTALL_DIR" pull --ff-only || true
  else
    command -v git >/dev/null 2>&1 || fail "缺少 git"
    rm -rf "$INSTALL_DIR"
    log "clone $REPO_URL -> $INSTALL_DIR"
    git clone "$REPO_URL" "$INSTALL_DIR"
  fi
  cd "$INSTALL_DIR" || fail "无法进入 $INSTALL_DIR"
}

# curl 可能是旧缓存：拉完代码后强制用仓库里最新脚本重跑
maybe_reexec(){
  if [[ "${HYACINE_REEXEC:-0}" == "1" ]]; then
    return
  fi
  prepare_repo
  if [[ -f "$INSTALL_DIR/scripts/baota-run.sh" ]]; then
    log "re-exec local script v3 from repo"
    export HYACINE_REEXEC=1
    export POSTGRES_PASSWORD JWT_ACCESS_SECRET JWT_REFRESH_SECRET API_PORT INSTALL_DIR REPO_URL
    exec bash "$INSTALL_DIR/scripts/baota-run.sh"
  fi
}

ensure_network(){
  docker network inspect "$NETWORK" >/dev/null 2>&1 || docker network create "$NETWORK"
}

stop_old(){
  for c in hyacine-api hyacine-netease hyacine-postgres hyacine-redis; do
    docker rm -f "$c" >/dev/null 2>&1 || true
  done
}

# 尝试多个镜像源拉取
pull_with_mirrors(){
  local path="$1"   # 例如 library/postgres:16-alpine 或 binaryify/netease_cloud_music_api:latest
  local out_var="$2"
  local m img
  for m in $MIRRORS; do
    img="${m}/${path}"
    log "try pull $img"
    if docker pull "$img" >/tmp/hyacine-pull.log 2>&1; then
      eval "$out_var=\"$img\""
      log "ok: $img"
      return 0
    fi
    log "fail: $img"
  done
  # 最后试官方
  img="${path#library/}"
  if [[ "$path" == library/* ]]; then
    img="${path#library/}"
  else
    img="$path"
  fi
  log "try pull docker.io/$img"
  if docker pull "$img" >/tmp/hyacine-pull.log 2>&1; then
    eval "$out_var=\"$img\""
    log "ok: $img"
    return 0
  fi
  cat /tmp/hyacine-pull.log || true
  fail "所有镜像源都拉失败: $path"
}

run_postgres(){
  # 关键：不映射主机 5432，避免 address already in use
  docker run -d \
    --name hyacine-postgres \
    --network "$NETWORK" \
    --restart unless-stopped \
    -e POSTGRES_DB=hyacine \
    -e POSTGRES_USER=hyacine \
    -e POSTGRES_PASSWORD="$POSTGRES_PASSWORD" \
    -v "$DATA_DIR/postgres:/var/lib/postgresql/data" \
    "$IMG_POSTGRES" \
    || fail "postgres 启动失败"
}

run_redis(){
  # 不映射主机 6379
  docker run -d \
    --name hyacine-redis \
    --network "$NETWORK" \
    --restart unless-stopped \
    -v "$DATA_DIR/redis:/data" \
    "$IMG_REDIS" \
    redis-server --appendonly yes \
    || fail "redis 启动失败"
}

run_netease(){
  docker run -d \
    --name hyacine-netease \
    --network "$NETWORK" \
    --restart unless-stopped \
    -e PORT=3000 \
    "$IMG_NETEASE" \
    || fail "netease 启动失败"
}

build_and_run_api(){
  log "building api with NODE_IMAGE=$IMG_NODE"
  docker build \
    --build-arg "NODE_IMAGE=$IMG_NODE" \
    -t hyacine-server:latest \
    "$INSTALL_DIR" \
    || fail "api 镜像构建失败"

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
    hyacine-server:latest \
    || fail "api 启动失败"
}

wait_health(){
  local i
  for ((i=1;i<=90;i++)); do
    if curl -fsS "http://127.0.0.1:${API_PORT}/api/v1/health" >/dev/null 2>&1; then
      log "health ok"
      return 0
    fi
    sleep 2
  done
  log "health timeout"
  docker logs --tail 100 hyacine-api || true
  return 1
}

main(){
  log "script $SCRIPT_VERSION"
  need_docker
  maybe_reexec
  # reexec 后从这里继续
  prepare_repo
  ensure_network
  stop_old

  pull_with_mirrors "library/postgres:16-alpine" IMG_POSTGRES
  pull_with_mirrors "library/redis:7-alpine" IMG_REDIS
  pull_with_mirrors "binaryify/netease_cloud_music_api:latest" IMG_NETEASE
  pull_with_mirrors "library/node:20-alpine" IMG_NODE

  log "start postgres (no host 5432)"
  run_postgres
  sleep 5

  log "start redis (no host 6379)"
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
IMG_POSTGRES=${IMG_POSTGRES}
IMG_REDIS=${IMG_REDIS}
IMG_NETEASE=${IMG_NETEASE}
IMG_NODE=${IMG_NODE}
EOF
  chmod 600 "$INSTALL_DIR/.env.runtime" || true

  log "done"
  docker ps --filter name=hyacine- || true
  log "API: http://${ip}:${API_PORT}"
  log "health: http://${ip}:${API_PORT}/api/v1/health"
  log "手机端填: http://${ip}:${API_PORT}"
  log "密码文件: ${INSTALL_DIR}/.env.runtime"
}

main "$@"