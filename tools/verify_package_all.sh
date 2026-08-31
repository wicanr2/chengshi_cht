#!/usr/bin/env bash
# 驗證 dist-all/<版本> 的公開包與本機完整版。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VER="${1:-}"
if [ -z "$VER" ]; then
  echo "用法：tools/verify_package_all.sh <版本>" >&2
  exit 2
fi
case "$VER" in
  *[!A-Za-z0-9._-]*|'') echo "不合法的版本：$VER" >&2; exit 2 ;;
esac
if [ ! -d "$ROOT/dist-all/$VER" ]; then
  echo "找不到 dist-all/$VER" >&2
  exit 2
fi

docker run --rm \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" \
  --memory 4g --cpus 2 --pids-limit 512 \
  --network none \
  --tmpfs /cap:rw,size=300m \
  -v "$ROOT:/src" \
  -e VER="$VER" -e HOME=/tmp/player \
  -e LIBGL_ALWAYS_SOFTWARE=1 -e GALLIUM_DRIVER=llvmpipe \
  -w /src simcity-go:1.25 bash /src/tools/verify_package_all_inner.sh
