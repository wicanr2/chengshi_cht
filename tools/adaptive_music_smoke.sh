#!/usr/bin/env bash
# 從正常玩家路徑 Alt-D → 火災驗證固定災難曲真的切換並送出非零音訊。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

docker run --rm \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" \
  --memory 4g --cpus 2 --pids-limit 512 \
  --network none \
  -v "$ROOT:/src" \
  -v "$ROOT/workplace/gocache:/gocache" -v "$ROOT/workplace/gomod:/gomod" \
  -e GOCACHE=/gocache -e GOMODCACHE=/gomod -e GOFLAGS=-mod=mod -e HOME=/tmp \
  -e LIBGL_ALWAYS_SOFTWARE=1 -e GALLIUM_DRIVER=llvmpipe \
  -w /src simcity-go:1.25 bash /src/tools/audio/adaptive_music_smoke_inner.sh
