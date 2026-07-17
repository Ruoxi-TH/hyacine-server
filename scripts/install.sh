#!/usr/bin/env bash
# Ubuntu 一键安装 Docker 并部署 hyacine-server
# 用法（宝塔终端直接跑这一条）：
#   curl -fsSL https://raw.githubusercontent.com/Ruoxi-TH/hyacine-server/master/scripts/install.sh | bash
# 已在仓库目录：
#   bash scripts/install.sh

set -u

REPO_URL="${REPO_URL:-https://github.com/Ruoxi-TH/hyacine-server.git}"
if [[ -z "${INSTALL_DIR:-}" ]]; then
  if [[ -d /www/wwwroot ]]; then
    INSTALL_DIR="/www/wwwroot/hyacine-server"
  else
    INSTALL_DIR="$HOME/hyacine-server"
  fi
fi
API_PORT="${API_PORT:-3000}"

log() { printf '[hyacine] %s\n' "$*"; }
fail() { printf '[hyacine] ERROR: %s\n' "$*" >&2; exit 1; }

need_root_or_sudo() {
  if [[ "$(id -u)" -eq 0 ]]; then
    SUDO=""
  elif command -v sudo >/dev/null 2>&1; then
    SUDO="sudo"
  else
    fail "need root or sudo"
  fi
}

install_docker() {
  if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    log "docker already installed"
    return
  fi

  need_root_or_sudo
  log "installing docker..."

  if command -v apt-get >/dev/null 2>&1; then
    $SUDO apt-get update -y
    $SUDO apt-get install -y ca-certificates curl gnupg
    $SUDO install -m 0755 -d /etc/apt/keyrings
    if [[ ! -f /etc/apt/keyrings/docker.gpg ]]; then
      curl -fsSL https://download.docker.com/linux/ubuntu/gpg | $SUDO gpg --dearmor -o /etc/apt/keyrings/docker.gpg
      $SUDO chmod a+r /etc/apt/keyrings/docker.gpg
    fi
    . /etc/os-release
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu ${VERSION_CODENAME} stable" \
      | $SUDO tee /etc/apt/sources.list.d/docker.list >/dev/null
    $SUDO apt-get update -y
    $SUDO apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
  else
    curl -fsSL https://get.docker.com | $SUDO sh
  fi

  if [[ "$(id -u)" -ne 0 ]]; then
    $SUDO usermod -aG docker "$USER" || true
  fi

  $SUDO systemctl enable --now docker || true
  command -v docker >/dev/null 2>&1 || fail "docker install failed"
  docker compose version >/dev/null 2>&1 || fail "docker compose plugin missing"
  log "docker installed"
}

random_secret() {
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex 32
  elif command -v python3 >/dev/null 2>&1; then
    python3 - <<'PY'
import secrets
print(secrets.token_hex(32))
PY
  else
    date +%s%N | sha256sum | awk '{print $1}'
  fi
}

prepare_repo() {
  if [[ -f "./docker-compose.yml" && -f "./Dockerfile" ]]; then
    ROOT_DIR="$(pwd)"
    log "use current directory: $ROOT_DIR"
  else
    if [[ -d "$INSTALL_DIR/.git" ]]; then
      log "update existing repo: $INSTALL_DIR"
      git -C "$INSTALL_DIR" pull --ff-only || true
    else
      log "clone $REPO_URL -> $INSTALL_DIR"
      command -v git >/dev/null 2>&1 || {
        need_root_or_sudo
        $SUDO apt-get update -y && $SUDO apt-get install -y git
      }
      git clone "$REPO_URL" "$INSTALL_DIR"
    fi
    ROOT_DIR="$INSTALL_DIR"
  fi
  cd "$ROOT_DIR"
}

prepare_env() {
  if [[ ! -f .env ]]; then
    cp .env.deploy.example .env
    log "created .env"
  fi

  if grep -q 'replace-with-a-strong-database-password' .env; then
    sed -i "s/replace-with-a-strong-database-password/$(random_secret | cut -c1-24)/" .env
  fi
  if grep -q 'replace-with-a-random-secret-at-least-32-characters' .env; then
    sed -i "0,/replace-with-a-random-secret-at-least-32-characters/{s/replace-with-a-random-secret-at-least-32-characters/$(random_secret)/}" .env
  fi
  if grep -q 'replace-with-a-different-random-secret-at-least-32-characters' .env; then
    sed -i "s/replace-with-a-different-random-secret-at-least-32-characters/$(random_secret)/" .env
  fi
  if grep -q 'https://your-web-client.example' .env; then
    sed -i 's|https://your-web-client.example|*|' .env
  fi

  if grep -qE '^API_PORT=' .env; then
    sed -i "s/^API_PORT=.*/API_PORT=${API_PORT}/" .env
  else
    echo "API_PORT=${API_PORT}" >> .env
  fi
}

compose() {
  if docker compose version >/dev/null 2>&1; then
    docker compose "$@"
  elif command -v docker-compose >/dev/null 2>&1; then
    docker-compose "$@"
  else
    need_root_or_sudo
    $SUDO docker compose "$@"
  fi
}

wait_health() {
  local url="http://127.0.0.1:${API_PORT}/api/v1/health"
  local i
  for ((i = 1; i <= 60; i++)); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      log "health ok: $url"
      return 0
    fi
    sleep 2
  done
  log "health check timeout, check logs: docker compose logs -f api"
  return 1
}

main() {
  install_docker
  prepare_repo
  prepare_env

  log "building & starting..."
  compose up -d --build

  wait_health || true
  compose ps

  local ip
  ip="$(hostname -I 2>/dev/null | awk '{print $1}')"
  ip="${ip:-你的服务器IP}"

  log "done"
  log "API: http://${ip}:${API_PORT}"
  log "health: http://${ip}:${API_PORT}/api/v1/health"
  log "手机端后端地址填: http://${ip}:${API_PORT}"
}

main "$@"
