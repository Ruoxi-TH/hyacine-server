#!/bin/sh
set -eu

# NeteaseCloudMusicApi 默认 3000，改到 3001 避免和 API 冲突
export HOST=127.0.0.1
export PORT=3001

if command -v neteasecloudmusicapi >/dev/null 2>&1; then
  exec neteasecloudmusicapi
fi

# npm 全局包入口兼容
if [ -f /usr/local/lib/node_modules/NeteaseCloudMusicApi/app.js ]; then
  exec node /usr/local/lib/node_modules/NeteaseCloudMusicApi/app.js
fi

if [ -f /usr/local/lib/node_modules/NeteaseCloudMusicApi/server.js ]; then
  exec node /usr/local/lib/node_modules/NeteaseCloudMusicApi/server.js
fi

# 最后兜底：npx
exec npx --yes NeteaseCloudMusicApi@latest