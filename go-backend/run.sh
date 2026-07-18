#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
: "${NETEASE_API_BASE:?Set NETEASE_API_BASE, e.g. http://127.0.0.1:3001}"
: "${PORT:=3000}"

cd "$ROOT"
go build -o hyacine-go-server .
exec ./hyacine-go-server