#!/usr/bin/env bash
# 宝塔一键整套部署（Compose 一起起 4 个服务）
# curl -fsSL "https://raw.githubusercontent.com/Ruoxi-TH/hyacine-server/master/scripts/baota-run.sh?v=4" | bash

set -u

SCRIPT_VERSION="2026-07-17-v4-compose"
REPO_URL="${REPO_URL:-https://github.com/Ruoxi-TH/hyacine-server.git}"
INSTALL_DIR="${INSTALL_DIR:-/www/wwwroot/hyacine-server}"
API_PORT="${API_PORT:-3000}"
MIRRORS="${MIRRORS:-docker.1ms.run docker.m.daocloud.io dockerproxy.com}"

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
  if docker compose version >/dev/null 2>&1; then
    COMPOSE=(docker compose)
  elif command -v docker-compose >/dev/null 2>&1; then
    COMPOSE=(docker-compose)
  else
    fail "没有 docker compose，请在宝塔 Docker 里启用 Compose"
  fi
}

prepare_repo(){
  mkdir -p "$(dirname "$INSTALL_DIR")"
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

maybe_reexec(){
  if [[ "${HYACINE_REEXEC:-0}" == "1" ]]; then
    return
  fi
  prepare_repo
  if [[ -f "$INSTALL_DIR/scripts/baota-run.sh" ]]; then
    log "re-exec local script $SCRIPT_VERSION"
    export HYACINE_REEXEC=1
    export API_PORT INSTALL_DIR REPO_URL MIRRORS
    exec bash "$INSTALL_DIR/scripts/baota-run.sh"
  fi
}

write_env(){
  if [[ -f .env ]] && ! grep -q 'replace-with-a-strong-database-password' .env 2>/dev/null; then
    # 保留已有密码，只补字段
    grep -qE '^API_PORT=' .env || echo "API_PORT=${API_PORT}" >> .env
    sed -i "s/^API_PORT=.*/API_PORT=${API_PORT}/" .env || true
    if grep -q 'https://your-web-client.example' .env; then
      sed -i 's|https://your-web-client.example|*|' .env
    fi
    grep -qE '^CORS_ORIGIN=' .env || echo 'CORS_ORIGIN=*' >> .env
    log "keep existing .env"
    return
  fi

  cat > .env <<EOF
API_PORT=${API_PORT}
POSTGRES_PASSWORD=$(random_secret | cut -c1-24)
CORS_ORIGIN=*
JWT_ACCESS_SECRET=$(random_secret)
JWT_REFRESH_SECRET=$(random_secret)
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=30d
EOF
  chmod 600 .env || true
  log "generated .env"
}

# 选一个能 pull 的镜像源，写入 .env 供 compose 用
pick_images(){
  local m
  local node_ok="" pg_ok="" redis_ok="" netease_ok=""

  for m in $MIRRORS; do
    log "probe mirror: $m"
    if [[ -z "$pg_ok" ]] && docker pull "$m/library/postgres:16-alpine" >/tmp/hyacine-pull.log 2>&1; then
      pg_ok="$m/library/postgres:16-alpine"
      log "postgres ok: $pg_ok"
    fi
    if [[ -z "$redis_ok" ]] && docker pull "$m/library/redis:7-alpine" >/tmp/hyacine-pull.log 2>&1; then
      redis_ok="$m/library/redis:7-alpine"
      log "redis ok: $redis_ok"
    fi
    if [[ -z "$node_ok" ]] && docker pull "$m/library/node:20-alpine" >/tmp/hyacine-pull.log 2>&1; then
      node_ok="$m/library/node:20-alpine"
      log "node ok: $node_ok"
    fi
    if [[ -z "$netease_ok" ]] && docker pull "$m/binaryify/netease_cloud_music_api:latest" >/tmp/hyacine-pull.log 2>&1; then
      netease_ok="$m/binaryify/netease_cloud_music_api:latest"
      log "netease ok: $netease_ok"
    fi
    if [[ -n "$pg_ok" && -n "$redis_ok" && -n "$node_ok" && -n "$netease_ok" ]]; then
      break
    fi
  done

  # 官方兜底
  if [[ -z "$pg_ok" ]]; then
    docker pull postgres:16-alpine >/tmp/hyacine-pull.log 2>&1 && pg_ok="postgres:16-alpine" || true
  fi
  if [[ -z "$redis_ok" ]]; then
    docker pull redis:7-alpine >/tmp/hyacine-pull.log 2>&1 && redis_ok="redis:7-alpine" || true
  fi
  if [[ -z "$node_ok" ]]; then
    docker pull node:20-alpine >/tmp/hyacine-pull.log 2>&1 && node_ok="node:20-alpine" || true
  fi
  if [[ -z "$netease_ok" ]]; then
    docker pull binaryify/netease_cloud_music_api:latest >/tmp/hyacine-pull.log 2>&1 && netease_ok="binaryify/netease_cloud_music_api:latest" || true
  fi

  [[ -n "$pg_ok" ]] || fail "postgres 镜像拉取失败"
  [[ -n "$redis_ok" ]] || fail "redis 镜像拉取失败"
  [[ -n "$node_ok" ]] || fail "node 镜像拉取失败"
  [[ -n "$netease_ok" ]] || fail "netease 镜像拉取失败"

  # 写入/更新 .env 镜像变量
  for k in POSTGRES_IMAGE REDIS_IMAGE NODE_IMAGE NETEASE_IMAGE; do
    sed -i "/^${k}=/d" .env 2>/dev/null || true
  done
  {
    echo "POSTGRES_IMAGE=${pg_ok}"
    echo "REDIS_IMAGE=${redis_ok}"
    echo "NODE_IMAGE=${node_ok}"
    echo "NETEASE_IMAGE=${netease_ok}"
  } >> .env

  export POSTGRES_IMAGE="$pg_ok"
  export REDIS_IMAGE="$redis_ok"
  export NODE_IMAGE="$node_ok"
  export NETEASE_IMAGE="$netease_ok"
}

stop_old_single_containers(){
  # 清理之前 docker run 方式留下的散装容器
  for c in hyacine-api hyacine-netease hyacine-postgres hyacine-redis; do
    docker rm -f "$c" >/dev/null 2>&1 || true
  done
}

compose_up(){
  log "compose up all services together"
  "${COMPOSE[@]}" pull || true
  "${COMPOSE[@]}" up -d --build --remove-orphans || fail "compose 启动失败"
}

wait_health(){
  local port i
  port="$(grep -E '^API_PORT=' .env | tail -n1 | cut -d= -f2-)"
  port="${port:-$API_PORT}"
  for ((i=1;i<=90;i++)); do
    if curl -fsS "http://127.0.0.1:${port}/api/v1/health" >/dev/null 2>&1; then
      log "health ok"
      return 0
    fi
    sleep 2
  done
  log "health timeout"
  "${COMPOSE[@]}" ps || true
  "${COMPOSE[@]}" logs --tail=80 api || true
  return 1
}

main(){
  log "script $SCRIPT_VERSION"
  need_docker
  maybe_reexec
  prepare_repo
  write_env
  stop_old_single_containers
  pick_images
  compose_up
  wait_health || true

  local ip port
  ip="$(hostname -I 2>/dev/null | awk '{print $1}')"
  ip="${ip:-服务器IP}"
  port="$(grep -E '^API_PORT=' .env | tail -n1 | cut -d= -f2-)"
  port="${port:-3000}"

  log "done - 一套一起装完"
  "${COMPOSE[@]}" ps || true
  log "API: http://${ip}:${port}"
  log "health: http://${ip}:${port}/api/v1/health"
  log "手机端填: http://${ip}:${port}"
  log "目录: $INSTALL_DIR"
}

main "$@"