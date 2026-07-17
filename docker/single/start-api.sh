#!/bin/sh
set -eu

export DATABASE_URL="${DATABASE_URL:-file:/data/hyacine.db}"
export REDIS_URL="${REDIS_URL:-redis://127.0.0.1:6379}"
export NETEASE_API_BASE="${NETEASE_API_BASE:-http://127.0.0.1:3001}"
export PORT="${PORT:-3000}"
export CORS_ORIGIN="${CORS_ORIGIN:-*}"
export JWT_ACCESS_TTL="${JWT_ACCESS_TTL:-15m}"
export JWT_REFRESH_TTL="${JWT_REFRESH_TTL:-30d}"

if [ -z "${JWT_ACCESS_SECRET:-}" ]; then
  JWT_ACCESS_SECRET="$(cat /proc/sys/kernel/random/uuid 2>/dev/null || date +%s)_hyacine_access_secret_32b"
  export JWT_ACCESS_SECRET
fi
if [ -z "${JWT_REFRESH_SECRET:-}" ]; then
  JWT_REFRESH_SECRET="$(cat /proc/sys/kernel/random/uuid 2>/dev/null || date +%s)_hyacine_refresh_secret_32b"
  export JWT_REFRESH_SECRET
fi

mkdir -p /data
chown -R hyacine:hyacine /data /app || true

# 等 redis 就绪
i=0
while [ "$i" -lt 30 ]; do
  if redis-cli -h 127.0.0.1 -p 6379 ping 2>/dev/null | grep -q PONG; then
    break
  fi
  i=$((i + 1))
  sleep 1
done

cd /app
# 没有 migrations 时用 db push
if [ -d prisma/migrations ] && [ "$(ls -A prisma/migrations 2>/dev/null || true)" ]; then
  su -s /bin/sh hyacine -c "pnpm exec prisma migrate deploy"
else
  su -s /bin/sh hyacine -c "pnpm exec prisma db push --skip-generate"
fi

exec su -s /bin/sh hyacine -c "node dist/main"