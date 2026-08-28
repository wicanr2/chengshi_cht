#!/usr/bin/env bash
# 正常玩家路徑的實機驗證：在 docker + Xvfb 裡真的開遊戲、真的敲鍵、
# 真的點滑鼠，每一步截圖，最後用存檔內容做機械判定。
#
# 為什麼要有這支：單元測試與 headless 對拍全綠，不代表玩家打開來能玩。
# 預設存檔路徑寫不進去、視窗開了關不掉、工具選了點不下去——這些只在
# 真的有視窗、真的有滑鼠事件的時候才會現形。
#
# 用法：tools/playtest.sh [種子]
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SEED="${1:-7}"
DATA='workplace/dos110/SIMCITY 1.10'

# 地形是隨機的，把滑鼠座標寫死會在換種子時點進海裡。先問一次空地在哪。
read -r FX FY < <("$ROOT/tools/go.sh" run ./cmd/simtool flat -seed "$SEED")
echo "種子 $SEED：在 ($FX,$FY) 蓋城"

docker run --rm \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" \
  --memory 4g --cpus 2 --pids-limit 512 \
  --network none \
  -v "$ROOT:/src" \
  -v "$ROOT/workplace/gocache:/gocache" -v "$ROOT/workplace/gomod:/gomod" \
  -e GOCACHE=/gocache -e GOMODCACHE=/gomod -e GOFLAGS=-mod=mod -e HOME=/tmp \
  -e LIBGL_ALWAYS_SOFTWARE=1 -e GALLIUM_DRIVER=llvmpipe \
  -e SEED="$SEED" -e FX="$FX" -e FY="$FY" -e DATA="$DATA" \
  -w /src simcity-go:1.25 \
  bash tools/playtest_inner.sh
