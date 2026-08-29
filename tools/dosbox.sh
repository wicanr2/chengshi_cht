#!/usr/bin/env bash
# 在 docker + Xvfb 裡跑 DOS 原版，照著動作腳本操作，全程錄音並截圖。
#
# 用途不是玩，是當 oracle：回答只有原版跑起來才回答得了的問題
# （八段音效各對應哪個事件、取樣率、破解版有沒有拔掉手冊查驗）。
#
# 用法：
#   RUN=simcity ACTIONS=tools/dosbox/act-sound.txt tools/dosbox.sh 20 sound
#
# 產物在 workplace/dosbox/：<前綴>-*.png、<前綴>.raw（s16le 立體聲
# 22050 Hz）、<前綴>.marks（動作時間表）、<前綴>.log。
#
# ⚠ 遊戲目錄是**複製**進容器的可寫副本：SimCity 會寫 SIMCITY.CFG 與存檔，
# 原始素材必須保持唯讀。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SECS="${1:-20}"; PREFIX="${2:-run}"
ACTIONS="${ACTIONS:-$ROOT/tools/dosbox/act-none.txt}"
mkdir -p "$ROOT/workplace/dosbox"
[ -f "$ACTIONS" ] || { echo "找不到動作腳本 $ACTIONS"; exit 1; }

# 外層 timeout：容器裡的 DOSBox 卡住時不要把整個工作階段拖住。
# 預設給動作腳本的時間加上開機與收尾的餘裕。
exec timeout "${TIMEOUT:-600}" docker run --rm \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" \
  --memory 2g --cpus 2 --pids-limit 256 \
  --network none \
  -v "$ROOT/workplace/dos110:/orig:ro" \
  -v "$ROOT/workplace/dosbox:/out" \
  -v "$ROOT/${CONF:-tools/dosbox/dosbox-x.conf}:/conf/dosbox.conf:ro" \
  -v "$ACTIONS:/conf/actions.txt:ro" \
  -v "$ROOT/tools/dosbox_inner.sh:/conf/inner.sh:ro" \
  -e HOME=/tmp -e SECS="$SECS" -e PREFIX="$PREFIX" -e RUN="${RUN:-}" \
  -e CFG_SOUND="${CFG_SOUND:-}" -e CFG_SCREEN="${CFG_SCREEN:-}" \
  -e MACHINE="${MACHINE:-}" -e DOSBOX_BIN="${DOSBOX_BIN:-dosbox-x}" \
  -e TDY_FROM="${TDY_FROM:-}" -e CFG_GFX="${CFG_GFX:-}" \
  "${IMAGE:-simcity-dosbox:x}" bash /conf/inner.sh
