#!/usr/bin/env bash
set -euo pipefail

# Build static linux/amd64 binary
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o fileactivity-watcher

mkdir -p ../src/usr/local/php/unraid-fileactivity/bin
cp fileactivity-watcher ../src/usr/local/php/unraid-fileactivity/bin/