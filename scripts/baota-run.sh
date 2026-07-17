#!/usr/bin/env bash
# 宝塔一键：GitHub 已构建好的单容器镜像，直接 pull + run
# 用法：
#   curl -fsSL "https://raw.githubusercontent.com/Ruoxi-TH/hyacine-server/master/scripts/baota-run.sh?v=6" | bash
# 或本机：
#   bash scripts/baota-run.sh

set -u

SCRIPT_VERSION="2026-07-17-v6-ghcr"
API_PORT="${API_PORT:-3000}"
NAME="${NAME:-hyacine}"
DATA_DIR="${DATA_DIR:-/www/wwwroot/hyacine-data}"
IMAGE="${IMAGE:-ghcr.io/ruoxi-th/hyacine-server:latest}"

# 国内拉 GHCR 可能慢，可改：
# IMAGE=ghcr.io/ruoxi-th/hyacine-server:latest

log(){ printf '[hyacine] %s\n' "$*"; }
fail(){ printf '[hyacine] ERROR: %s\n' "$*" >&2; exit 1; }

random_secret(){
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex 32
  else
    date +%s%N | sha256sum | awk '{print $1}'
  fi
}

need_docker(){
  command -v docker >/dev/null 2>&1 || fail "请先在宝塔安装 Docker 管理器"
  docker info >/dev/null 2>&1 || fail "Docker 未启动"
}

pull_image(){
  log "pull image: $IMAGE"
  if docker pull "$IMAGE"; then
    return 0
  fi
  # 常见代理前缀兜底（若可用）
  local proxies=(
    ""
    "docker.1ms.run/"
    "docker.m.daocloud.io/"
  )
  local p
  for p in "${proxies[@]}"; do
    [[ -z "$p" ]] && continue
    log "try proxy pull: ${p}${IMAGE#https://}"
    if docker pull "${p}${IMAGE}"; then
      IMAGE="${p}${IMAGE}"
      return 0
    fi
  done
  fail "镜像拉取失败。请确认 GitHub Actions 已构建成功，且仓库包为 Public，或本机可访问 ghcr.io"
}

cleanup_old(){
  for c in hyacine hyacine-api hyacine-netease hyacine-postgres hyacine-redis; do
    docker rm -f "$c" >/dev/null 2>&1 || true
  done
}

run_single(){
  local access refresh
  access="$(random_secret)"
  refresh="$(random_secret)"
  mkdir -p "$DATA_DIR"

  log "run ONE container: $NAME"
  docker run -d \
    --name "$NAME" \
    --restart unless-stopped \
    -p "${API_PORT}:3000" \
    -e PORT=3000 \
    -e DATABASE_URL="file:/data/hyacine.db" \
    -e REDIS_URL="redis://127.0.0.1:6379" \
    -e NETEASE_API_BASE="http://127.0.0.1:3001" \
    -e CORS_ORIGIN="*" \
    -e JWT_ACCESS_SECRET="$access" \
    -e JWT_REFRESH_SECRET="$refresh" \
    -e JWT_ACCESS_TTL=15m \
    -e JWT_REFRESH_TTL=30d \
    -v "$DATA_DIR:/data" \
    "$IMAGE" \
    || fail "容器启动失败"
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
  log "health timeout"
  docker logs --tail 80 "$NAME" || true
  return 1
}

main(){
  log "script $SCRIPT_VERSION (pull prebuilt image, ONE container)"
  need_docker
  pull_image
  cleanup_old
  run_single
  wait_health || true

  local ip
  ip="$(hostname -I 2>/dev/null | awk '{print $1}')"
  ip="${ip:-服务器IP}"

  log "done - 只有 1 个容器"
  docker ps --filter "name=^/${NAME}$" || docker ps --filter "name=${NAME}"
  log "API: http://${ip}:${API_PORT}"
  log "手机端填: http://${ip}:${API_PORT}"
}

main "$@"