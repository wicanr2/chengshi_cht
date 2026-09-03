#!/usr/bin/env bash
# 地形編輯器：逐一換 TERRAIN.CFG 的螢幕模式跑一次，看哪一種進得了介面。
#
# 為什麼要換模式而不是換模擬器設定：錯誤訊息
# `%dK of VGA/EGA memory / Couldn't load VGA/EGA blocks!` 是**程式自己印的**
# （`sub_1951A`＋0x19810，`sub_167E3` 回 0 才會走到），代表模擬器把它跑對了，
# 失敗在程式的載入流程裡。每一種螢幕模式走**不同的載入函式與不同的資料檔**，
# 所以換模式是把「共用基礎設施壞了」與「這一條路壞了」分開的最短路徑。
#
# 模式代號出自 SIMCITY.CFG 自帶的解碼表（docs/re/20-terrain-editor.md §三）。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
SRC=workplace/te-modes
for m in "${@:-E M e C 2 H T}"; do
  echo "===== 螢幕模式 $m ====="
  printf '%s' "$m" | dd of="$SRC/TERRAIN.CFG" bs=1 seek=0 conv=notrunc status=none
  EXTRA="$SRC" RUN=terrain ACTIONS="$ROOT/tools/dosbox/act-te-mode.txt" \
    TIMEOUT=120 ./tools/dosbox.sh 30 "tem-$m" >/dev/null 2>&1 || echo "（逾時或非零結束）"
  ls workplace/dosbox/tem-$m-*.png 2>/dev/null | tail -1
done
