#!/usr/bin/env bash
# 在 docker 裡跑 Go（不裝到系統）。用法：tools/go.sh test ./... / build ./... / vet ./...
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE="${GO_IMAGE:-simcity-go:1.25}"
mkdir -p "$ROOT/workplace/gocache" "$ROOT/workplace/gomod"
# 把 SIMCITY_* 傳進容器（診斷用的開關都走這個前綴：SIMCITY_DEEP、
# SIMCITY_SEGS、SIMCITY_TRACE）。
ENVARGS=()
while IFS='=' read -r k _; do
  case "$k" in SIMCITY_*) ENVARGS+=(-e "$k") ;; esac
done < <(env)
exec docker run --rm \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" \
  --memory 4g --cpus 4 --pids-limit 512 \
  -v "$ROOT:/src" \
  -v "$ROOT/workplace/gocache:/gocache" \
  -v "$ROOT/workplace/gomod:/gomod" \
  -e GOCACHE=/gocache -e GOMODCACHE=/gomod -e GOFLAGS=-mod=mod -e HOME=/tmp \
  ${ENVARGS+"${ENVARGS[@]}"} \
  -w /src "$IMAGE" go "$@"
