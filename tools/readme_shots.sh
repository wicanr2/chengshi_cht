#!/usr/bin/env bash
# 重產 README 的畫面截圖。
#
# 為什麼要有腳本：這些圖是「現在的 remake 長什麼樣」的證據，版面一改就過期。
# 手動截圖沒有紀錄用了哪些參數，下次改完版面沒有人知道怎麼截出同一張。
#
# ⚠ 圖裡的地圖圖塊來自**玩家自己那份原版**（CLAUDE.md §8）：
# 本專案只放 remake 跑起來的畫面，不放原版遊戲本身的截圖。
#
# 用法：tools/readme_shots.sh
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
OUT=docs/images
mkdir -p "$OUT"

shot() { # 檔名 GAME_ARGS GAME_KEYS 等待秒數
  local name="$1" args="$2" keys="${3:-}" wait="${4:-8}"
  GAME_ARGS="$args" GAME_KEYS="$keys" ./tools/screenshot.sh "$wait" "_rm.png" >/dev/null 2>&1
  cp workplace/shots/_rm.png "$OUT/$name"
  echo "  $name  ($args)"
}

echo "重產 README 截圖："
# 底特律 1972，基本外觀。space 關掉劇本簡介、0 暫停讓畫面穩定。
shot city.png       "-scenario 6 -style base -cam 30,30" "space 0"
# 同一座城市換中世紀資料片。
shot style-medi.png "-scenario 6 -style medi -cam 30,30" "space 0"
# 地圖視窗的犯罪率圖層（圖層 6，順序照訊息檔第 10 段）。
# ⚠ 這一張**不能按 `0` 暫停**：地圖視窗開著的時候 `1`–`9`／`0` 是切圖層，
# 按下去會把 `-layer 6` 蓋掉（先前那一版拍到的其實是「消防範圍」）。
shot maps.png       "-scenario 6 -style base -window maps -layer 6" "space"
# 劇本簡介（中世紀的改寫）。不要按 space，簡介才留在畫面上。
shot brief.png      "-scenario 6 -style medi" "" 6
# 評估視窗。
shot eval.png       "-scenario 6 -style base -window eval" "space 0"
# 縮小到 1/4（remake 加的功能，原版沒有）。按兩次 `-`。
shot zoom.png       "-scenario 6 -style base -cam 0,0" "space 0 minus minus"
rm -f workplace/shots/_rm.png
echo "完成，圖在 $OUT"
