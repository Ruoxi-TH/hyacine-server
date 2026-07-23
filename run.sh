#!/bin/bash
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
cd "$ROOT"

echo "Building hyacine-server..."
go build -o hyacine-server ./cmd/hyacine-server

echo "Starting hyacine-server..."
exec ./hyacine-server