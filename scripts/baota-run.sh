#!/usr/bin/env bash
# 宝塔一键：单容器部署（1 个容器包含 API+DB+Redis+网易云）
# 用法：
#   HYACINE_REEXEC=1 SKIP_GIT=1 bash scripts/baota-run.sh
#   nohup env HYACINE_REEXEC=1 SKIP_GIT=1 bash scripts/baota-run.sh > /tmp/hyacine-deploy.log 2>&1 &

set -u

SCRIPT_VERSION="2026-07-17-v5.1-single"
REPO_URL="${REPO_URL:-https://github.com/Ruoxi-TH/hyacine-server.git}"
INSTALL_DIR="${INSTALL_DIR:-/www/wwwroot/hyacine-server}"
API_PORT="${API_PORT:-3000}"
NAME="${NAME:-hyacine}"
DATA_DIR="${DATA_DIR:-/www/wwwroot/hyacine-data}"
MIRRORS="${MIRRORS:-docker.1ms.run docker.m.daocloud.io dockerproxy.com}"
SKIP_GIT="${SKIP_GIT:-0}"
GIT_TIMEOUT="${GIT_TIMEOUT:-20}"

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

run_with_timeout(){
  local sec="$1"; shift
  if command -v timeout >/dev/null 2>&1; then
    timeout "$sec" "$@"
  else
    "$@"
  fi
}

prepare_repo(){
  mkdir -p "$(dirname "$INSTALL_DIR")" "$DATA_DIR"

  if [[ "$SKIP_GIT" == "1" ]]; then
    log "skip git (SKIP_GIT=1)"
    [[ -d "$INSTALL_DIR" ]] || fail "目录不存在: $INSTALL_DIR"
    cd "$INSTALL_DIR" || fail "无法进入 $INSTALL_DIR"
    return
  fi

  if [[ -d "$INSTALL_DIR/.git" ]]; then
    log "update repo (timeout ${GIT_TIMEOUT}s): $INSTALL_DIR"
    run_with_timeout "$GIT_TIMEOUT" git -C "$INSTALL_DIR" fetch --all --prune || log "git fetch skip/timeout"
    run_with_timeout "$GIT_TIMEOUT" git -C "$INSTALL_DIR" reset --hard origin/master || log "git reset skip"
  else
    command -v git >/dev/null 2>&1 || fail "缺少 git"
    rm -rf "$INSTALL_DIR"
    log "clone $REPO_URL -> $INSTALL_DIR"
    if ! run_with_timeout 180 git clone "$REPO_URL" "$INSTALL_DIR"; then
      log "clone fail, try proxy"
      run_with_timeout 180 git clone "https://ghproxy.net/https://github.com/Ruoxi-TH/hyacine-server.git" "$INSTALL_DIR" \
        || fail "git clone 失败"
    fi
  fi
  cd "$INSTALL_DIR" || fail "无法进入 $INSTALL_DIR"
}

maybe_reexec(){
  if [[ "${HYACINE_REEXEC:-0}" == "1" ]]; then
    return
  fi
  prepare_repo
  if [[ -f "$INSTALL_DIR/scripts/baota-run.sh" ]]; then
    log "re-exec local $SCRIPT_VERSION"
    export HYACINE_REEXEC=1
    export API_PORT INSTALL_DIR REPO_URL DATA_DIR NAME MIRRORS SKIP_GIT GIT_TIMEOUT
    exec bash "$INSTALL_DIR/scripts/baota-run.sh"
  fi
}

pick_node_image(){
  local m
  for m in $MIRRORS; do
    log "try node image: $m/library/node:20-alpine"
    if docker pull "$m/library/node:20-alpine" >/tmp/hyacine-pull.log 2>&1; then
      NODE_IMAGE="$m/library/node:20-alpine"
      log "node ok: $NODE_IMAGE"
      return 0
    fi
  done
  if docker pull node:20-alpine >/tmp/hyacine-pull.log 2>&1; then
    NODE_IMAGE="node:20-alpine"
    log "node ok: $NODE_IMAGE"
    return 0
  fi
  cat /tmp/hyacine-pull.log || true
  fail "node 镜像拉取失败"
}

cleanup_old(){
  for c in hyacine-api hyacine-netease hyacine-postgres hyacine-redis "$NAME"; do
    docker rm -f "$c" >/dev/null 2>&1 || true
  done
  if [[ -f "$INSTALL_DIR/docker-compose.yml" ]]; then
    (cd "$INSTALL_DIR" && docker compose down >/dev/null 2>&1) || true
  fi
}

build_single(){
  [[ -f "$INSTALL_DIR/Dockerfile.single" ]] || fail "缺少 Dockerfile.single"
  [[ -f "$INSTALL_DIR/tsconfig.build.json" ]] || fail "缺少 tsconfig.build.json，请先更新代码"
  log "build single image..."
  docker build \
    -f "$INSTALL_DIR/Dockerfile.single" \
    --build-arg "NODE_IMAGE=$NODE_IMAGE" \
    -t hyacine-single:latest \
    "$INSTALL_DIR" \
    || fail "单容器镜像构建失败"
}

run_single(){
  local access refresh
  access="$(random_secret)"
  refresh="$(random_secret)"
  mkdir -p "$DATA_DIR"

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
    hyacine-single:latest \
    || fail "单容器启动失败"

  cat > "$INSTALL_DIR/.env.runtime" <<EOF
API_PORT=${API_PORT}
JWT_ACCESS_SECRET=${access}
JWT_REFRESH_SECRET=${refresh}
DATA_DIR=${DATA_DIR}
CONTAINER=${NAME}
MODE=single
EOF
  chmod 600 "$INSTALL_DIR/.env.runtime" || true
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
  docker logs --tail 120 "$NAME" || true
  return 1
}

main(){
  log "script $SCRIPT_VERSION (ONE container)"
  need_docker
  maybe_reexec
  prepare_repo
  pick_node_image
  cleanup_old
  build_single
  run_single
  wait_health || true

  local ip
  ip="$(hostname -I 2>/dev/null | awk '{print $1}')"
  ip="${ip:-服务器IP}"

  log "done - 只有 1 个容器"
  docker ps --filter "name=^/${NAME}$" || docker ps --filter "name=${NAME}"
  log "API: http://${ip}:${API_PORT}"
  log "health: http://${ip}:${API_PORT}/api/v1/health"
  log "手机端填: http://${ip}:${API_PORT}"
  log "log: /tmp/hyacine-deploy.log"
}

main "$@"