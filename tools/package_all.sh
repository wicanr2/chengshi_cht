#!/usr/bin/env bash
# 建立 dist-all/<版本>/{full,release,promo}。
#
# release 只含可公開的引擎、授權與匯入說明；full 另外包入本機合法持有的
# SimCity 1.10 資料與 music/，因此只能留在本機。所有實際建置與封裝都在
# Docker 內完成，主機只負責啟動容器。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VER="${1:-$(date +%Y%m%d)}"
OUT="$ROOT/dist-all/$VER"

case "$VER" in
  *[!A-Za-z0-9._-]*|'')
    echo "版本只能使用英數、點、底線與連字號：$VER" >&2
    exit 2
    ;;
esac

if [ -e "$OUT" ]; then
  echo "拒絕覆蓋既有交付目錄：$OUT" >&2
  echo "請換版本號，或先由人工確認並移走該目錄。" >&2
  exit 2
fi

DATA="$ROOT/workplace/dos110/SIMCITY 1.10"
MUSIC="$ROOT/music"
if [ ! -d "$DATA/DATA" ]; then
  echo "找不到本機完整版來源：$DATA" >&2
  exit 2
fi
if ! find "$MUSIC" -maxdepth 1 -type f \( -iname '*.ogg' -o -iname '*.wav' \) -print -quit \
    | grep -q .; then
  echo "找不到本機音樂：$MUSIC（需要至少一首 .ogg 或 .wav）" >&2
  exit 2
fi

mkdir -p "$OUT"
keep_output=0
cleanup_partial() {
  if [ "$keep_output" -eq 0 ] && [ -d "$OUT" ]; then
    rm -rf -- "$OUT"
  fi
}
trap cleanup_partial EXIT

if ! docker image inspect simcity-osxcross:15.5 >/dev/null 2>&1; then
  echo "缺少既有的 simcity-osxcross:15.5，無法履行 macOS universal 發行承諾" >&2
  exit 2
fi

docker run --rm \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" \
  --memory 6g --cpus 4 --pids-limit 512 \
  --network none \
  -v "$ROOT:/src" \
  -v "$ROOT/workplace/gocache:/gocache" \
  -v "$ROOT/workplace/gomod:/gomod" \
  -e GOCACHE=/gocache -e GOMODCACHE=/gomod -e GOFLAGS=-mod=mod -e HOME=/tmp \
  -e VER="$VER" -e MAC_OUT="/src/workplace/package-mac-$VER" \
  -w /src simcity-osxcross:15.5 bash /src/tools/build-mac-inner.sh

docker run --rm \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" \
  --memory 4g --cpus 2 --pids-limit 512 \
  --network none \
  -v "$ROOT:/src" \
  -v "$ROOT/workplace/gocache:/gocache" \
  -v "$ROOT/workplace/gomod:/gomod" \
  -e GOCACHE=/gocache -e GOMODCACHE=/gomod -e GOFLAGS=-mod=mod -e HOME=/tmp \
  -e VER="$VER" \
  -w /src simcity-go:1.25 bash /src/tools/package_all_inner.sh

keep_output=1
trap - EXIT

echo
echo "交付產物：$OUT"
