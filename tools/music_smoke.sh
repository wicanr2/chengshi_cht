#!/usr/bin/env bash
# 驗證本機 music/ 的 OGG 真的走過 Ebiten／oto 並送到音訊裝置。
# 原曲與輸出都屬本機素材，不進公開發行包。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

docker run --rm \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" \
  --memory 4g --cpus 2 --pids-limit 512 \
  --network none \
  --tmpfs /cap:rw,size=300m \
  -v "$ROOT:/src" \
  -v "$ROOT/workplace/gocache:/gocache" -v "$ROOT/workplace/gomod:/gomod" \
  -e GOCACHE=/gocache -e GOMODCACHE=/gomod -e GOFLAGS=-mod=mod -e HOME=/tmp \
  -e LIBGL_ALWAYS_SOFTWARE=1 -e GALLIUM_DRIVER=llvmpipe \
  -w /src simcity-go:1.25 bash /src/tools/audio/music_smoke_inner.sh
