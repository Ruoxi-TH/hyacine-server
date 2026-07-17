#!/usr/bin/env bash
# 宝塔一键 docker run 部署（国内镜像兼容）
# 用法：
# curl -fsSL https://raw.githubusercontent.com/Ruoxi-TH/hyacine-server/master/scripts/baota-run.sh | bash

set -u

REPO_URL="${REPO_URL:-https://github.com/Ruoxi-TH/hyacine-server.git}"
INSTALL_DIR="${INSTALL_DIR:-/www/wwwroot/hyacine-server}"
API_PORT="${API_PORT:-3000}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-hyacine_$(date +%s | tail -c 8)}"
JWT_ACCESS_SECRET="${JWT_ACCESS_SECRET:-$(openssl rand -hex 32 2>/dev/null || echo hyacine_access_secret_change_me_32bytes_xx)}"
JWT_REFRESH_SECRET="${JWT_REFRESH_SECRET:-$(openssl rand -hex 32 2>/dev/null || echo hyacine_refresh_secret_change_me_32bytesx)}"
NETWORK="hyacine-net"
DATA_DIR="/www/wwwroot/hyacine-data"

# 国内镜像前缀，可覆盖：MIRROR=docker.1ms.run
MIRROR="${MIRROR:-docker.1ms.run}"
IMG_NODE="${IMG_NODE:-${MIRROR}/library/node:20-alpine}"
IMG_POSTGRES="${IMG_POSTGRES:-${MIRROR}/library/postgres:16-alpine}"
IMG_REDIS="${IMG_REDIS:-${MIRROR}/library/redis:7-alpine}"
IMG_NETEASE="${IMG_NETEASE:-${MIRROR}/binaryify/netease_cloud_music_api:latest}"

log(){ printf '[hyacine] %s\n' "$*"; }
fail(){ printf '[hyacine] ERROR: %s\n' "$*" >&2; exit 1; }

need_docker(){
  command -v docker >/dev/null 2>&1 || fail "请先在宝塔安装 Docker 管理器"
  docker info >/dev/null 2>&1 || fail "Docker 未启动"
}

prepare_repo(){
  mkdir -p "$(dirname "$INSTALL_DIR")" "$DATA_DIR/postgres" "$DATA_DIR/redis"
  if [[ -d "$INSTALL_DIR/.git" ]]; then
    log "update repo: $INSTALL_DIR"
    git -C "$INSTALL_DIR" pull --ff-only || true
  else
    command -v git >/dev/null 2>&1 || fail "缺少 git"
    rm -rf "$INSTALL_DIR"
    log "clone $REPO_URL -> $INSTALL_DIR"
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

# 清理坏掉的本地镜像缓存（text/html 拉失败残留）
clean_bad_images(){
  docker image prune -f >/dev/null 2>&1 || true
}

pull_image(){
  local img="$1"
  log "pull $img"
  if ! docker pull "$img"; then
    fail "拉取镜像失败: $img （可换 MIRROR=docker.m.daocloud.io 再试）"
  fi
}

run_postgres(){
  # 不映射主机 5432，避免和宝塔/本机 postgres 冲突；容器内互通即可
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
  log "building api image with $IMG_NODE ..."
  # 用国内 node 镜像，避免 docker.io 拉失败
  if ! docker build \
    -t hyacine-server:latest \
    -f - "$INSTALL_DIR" <<EOF
FROM ${IMG_NODE} AS base
WORKDIR /app
ENV PNPM_HOME=/pnpm \\
    PATH=/pnpm:\$PATH \\
    CI=true \\
    COREPACK_ENABLE_DOWNLOAD_PROMPT=0
RUN corepack enable

FROM base AS deps
RUN apk add --no-cache python3 make g++
COPY package.json pnpm-lock.yaml ./
COPY prisma ./prisma
RUN pnpm install --frozen-lockfile

FROM deps AS build
COPY nest-cli.json tsconfig.json tsconfig.build.json ./
COPY src ./src
RUN pnpm prisma:generate && pnpm build

FROM base AS production
ENV NODE_ENV=production
RUN apk add --no-cache openssl tini \\
  && addgroup -S hyacine \\
  && adduser -S -G hyacine hyacine
WORKDIR /app
COPY package.json pnpm-lock.yaml ./
COPY prisma ./prisma
RUN pnpm install --frozen-lockfile --prod \\
  && pnpm prisma:generate \\
  && chown -R hyacine:hyacine /app
COPY --from=build --chown=hyacine:hyacine /app/dist ./dist
USER hyacine
EXPOSE 3000
HEALTHCHECK --interval=15s --timeout=5s --start-period=30s --retries=5 \\
  CMD wget -qO- http://127.0.0.1:3000/api/v1/health >/dev/null 2>&1 || exit 1
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["sh", "-c", "pnpm prisma:deploy && node dist/main"]
EOF
  then
    fail "api 镜像构建失败"
  fi

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
  log "health timeout，看日志: docker logs -f hyacine-api"
  docker logs --tail 80 hyacine-api || true
  return 1
}

main(){
  need_docker
  prepare_repo
  ensure_network
  stop_old
  clean_bad_images

  pull_image "$IMG_POSTGRES"
  pull_image "$IMG_REDIS"
  pull_image "$IMG_NETEASE"
  pull_image "$IMG_NODE"

  log "start postgres"
  run_postgres
  sleep 5

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
MIRROR=${MIRROR}
EOF
  chmod 600 "$INSTALL_DIR/.env.runtime" || true

  log "done"
  docker ps --filter name=hyacine-
  log "API: http://${ip}:${API_PORT}"
  log "health: http://${ip}:${API_PORT}/api/v1/health"
  log "手机端填: http://${ip}:${API_PORT}"
  log "密码已保存: ${INSTALL_DIR}/.env.runtime"
  log "如 pull 失败可换: MIRROR=docker.m.daocloud.io bash scripts/baota-run.sh"
}

main "$@"