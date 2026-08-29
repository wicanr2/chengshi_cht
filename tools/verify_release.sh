#!/usr/bin/env bash
# 驗發行包：解開 dist/ 裡的 tar.gz，在乾淨的環境把它跑起來。
# 先跑 tools/release.sh 打包，再跑這一支。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ls "$ROOT"/dist/chengshi_cht-*-linux-amd64.tar.gz >/dev/null

docker run --rm \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" \
  --memory 4g --cpus 2 --pids-limit 512 \
  --network none \
  -v "$ROOT:/src" \
  -e LIBGL_ALWAYS_SOFTWARE=1 -e GALLIUM_DRIVER=llvmpipe -e HOME=/tmp \
  -w /src simcity-go:1.25 bash tools/verify_release_inner.sh 2>&1 | grep -v "^XGB:"
