#!/bin/bash
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
cd "$ROOT"

# 默认配置文件路径
CONFIG_FILE="./config.json"

# 如果配置文件不存在，自动创建默认配置
if [ ! -f "$CONFIG_FILE" ]; then
  echo "Creating default config.json..."
  cat > "$CONFIG_FILE" << 'EOF'
{
  "port": 3000,
  "netease_api_base": "",
  "log_level": "info",
  "cors": {
    "enabled": true,
    "origins": ["*"]
  },
  "stream": {
    "buffer_size": 32768,
    "timeout": 30
  }
}
EOF
  echo "Created config.json with default settings."
fi

# 编译
echo "Building hyacine-server..."
go build -o hyacine-server ./cmd/hyacine-server

# 启动
echo "Starting hyacine-server..."
echo "Config file: $CONFIG_FILE"
exec ./hyacine-server