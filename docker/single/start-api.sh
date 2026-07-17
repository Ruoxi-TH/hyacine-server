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

# sqlite 直接用 db push，避免无 migrations
cd /app
su -s /bin/sh hyacine -c "pnpm exec prisma db push --skip-generate"
exec su -s /bin/sh hyacine -c "node dist/main"