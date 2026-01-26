#!/usr/bin/env bash
set -euo pipefail

GIT_TAG=$(git describe --tags --always 2>/dev/null || echo "unknown")

# Build static linux/amd64 binary
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w -X github.com/dkaser/unraid-fileactivity/fileactivity-watcher/version.Tag=${GIT_TAG}" -o fileactivity-watcher

mkdir -p ../src/usr/local/php/unraid-fileactivity/bin
cp fileactivity-watcher ../src/usr/local/php/unraid-fileactivity/bin/