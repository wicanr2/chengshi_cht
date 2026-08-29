#!/usr/bin/env bash
# 在 Linux 上編 macOS 版（osxcross），兩個架構各編一次再 lipo 成 universal。
#
# 為什麼不能像 Windows 那樣 CGO_ENABLED=0：Ebiten 的 macOS 後端是
# Objective-C，一定要 cgo。osxcross 補的是 SDK、Mach-O 連結器與 wrapper。
#
# ⚠ 不做 notarization（那一定要 Mac），.app 也不簽名。
#   「未簽」勝過「壞簽」：壞簽直接被 Gatekeeper 拒絕，未簽只是要右鍵 → 打開。
#   執行檔本身的 ad-hoc 簽章才是能不能跑的關鍵，第 5 節會驗。
# ⚠ **Linux 上執行不了 macOS binary**。這裡只做靜態驗收：
#   結構對、相依只有系統庫、arm64 有簽章、含得到本次功能的字串。
#   靜態全過只代表不會因結構問題開不起來，不代表功能正常。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VER="${1:-$(date +%Y%m%d)}"
OUT="$ROOT/dist/mac"
rm -rf "$OUT"; mkdir -p "$OUT"

docker run --rm \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" \
  --memory 6g --cpus 4 --pids-limit 512 \
  --network none \
  -v "$ROOT:/src" \
  -v "$ROOT/workplace/gocache:/gocache" -v "$ROOT/workplace/gomod:/gomod" \
  -e GOCACHE=/gocache -e GOMODCACHE=/gomod -e GOFLAGS=-mod=mod -e HOME=/tmp \
  -e VER="$VER" \
  -w /src simcity-osxcross:15.5 bash /src/tools/build-mac-inner.sh
