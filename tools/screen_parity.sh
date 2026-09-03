#!/usr/bin/env bash
# **畫面對拍**：把 DOS 原版與 remake 的編輯視窗逐格比到位元組。
#
# 跟 `tools/dos_parity.sh` 的差別：那一支比的是**存檔**（模擬跑出來的狀態），
# 這一支比的是**畫面**（呈現層畫出來的像素）。兩者抓得到的錯完全不同——
# 存檔對拍對「調色盤偏暗一階」「工具盤位置差兩像素」「圖塊色號 0 當透明」
# 完全無感，而畫面對拍全部擋得下來。
#
# 它也是唯一抓得到「兩邊都錯得一樣」那類 bug 的判準。2026-08-30 就靠它抓到
# 城市檔的**檔頭長度讀錯 16 個位元組**：地圖整張往下平移 8 列，而純量、
# 地物格數、甚至 remake 與 DOS 存檔的逐格對拍全部正常——因為兩邊都經過
# 同一個錯誤的讀法，偏移互相抵銷。
#
# 用法：tools/screen_parity.sh [劇本編號] [風格]      （預設 1 west）
#
# 流程：
#   1. DOSBox 載入劇本，**用方向鍵把鏡頭頂到左上角**（一定被夾在 0,0），
#      暫停、截圖、Ctrl-S 存檔。存檔只用來當「原版那一刻的地圖」的旁證。
#   2. remake 用同一個劇本與 `-cam 0,0`，同樣以 Ctrl-C 關閉 City Form 後截圖。
#   3. 兩張都降到 640×350，逐格比對完整 32×16 編輯區。
#
# ⚠ 原版素材與原版畫面都不進版控（CLAUDE.md §8），所以基準每次現跑。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
SCEN="${1:-1}"
STYLE="${2:-west}"
DATA="workplace/dos110/SIMCITY 1.10"
OUT=workplace/screen-parity/$STYLE-$SCEN
mkdir -p "$OUT"

# 劇本選擇畫面的八個位置（640×350 的畫面座標）。
SLOT_X=(80 240 402 562 80 240 402 562)
SLOT_Y=(103 103 103 103 253 253 253 253)
i=$((SCEN - 1))

ACT="$OUT/act.txt"
cat > "$ACT" <<EOF
# 由 tools/screen_parity.sh 產生。
wait 3
click 316 240
wait 5
click ${SLOT_X[$i]} ${SLOT_Y[$i]}
wait 8
key Return
wait 2
key Return
wait 3
click 311 190
wait 2
key 0
wait 2
shot 00-city-form
key ctrl+c
wait 2
# 頂到左上角：鏡頭被地圖邊界夾住，所以一定是 (0,0)，不必猜初始鏡頭。
keyrep Up 40
keyrep Left 60
wait 2
shot 01-edit
key ctrl+s
wait 3
key Return
wait 3
EOF

# ⚠ DOS 那一側的風格是 `SIMCITY.CFG` 的 `Graphics Set` 決定的，不是命令列。
# 少了這一行，換風格只換到 remake 那一邊，比出來的是兩套不同的美術。
echo "== 原版（DOSBox）=="
GFX=$(echo "$STYLE" | tr '[:lower:]' '[:upper:]')CEGA
RUN=simcity CFG_GFX="$GFX" ACTIONS="$ROOT/$ACT" timeout 300 ./tools/dosbox.sh 50 sp >/dev/null 2>&1
cp workplace/dosbox/sp-00-city-form.png "$OUT/dos-city-form.png"
cp workplace/dosbox/sp-01-edit.png "$OUT/dos.png"

echo "== remake =="
GAME_ARGS="-scenario $SCEN -style $STYLE -cam 0,0" GAME_KEYS="space 0" \
  ./tools/screenshot.sh 7 sp-remake-city-form.png >/dev/null 2>&1
cp workplace/shots/sp-remake-city-form.png "$OUT/remake-city-form.png"
GAME_ARGS="-scenario $SCEN -style $STYLE -cam 0,0" GAME_KEYS="space 0 ctrl+c" \
  ./tools/screenshot.sh 7 sp-remake.png >/dev/null 2>&1
cp workplace/shots/sp-remake.png "$OUT/remake.png"

# 風格對應的圖形檔，拿來產圖塊圖集（給定位工具用）。
PGF=$(ls "$DATA"/CEGA/* 2>/dev/null | grep -i "${STYLE}cega.pgf" | head -1)
[ -n "$PGF" ] && ./tools/go.sh run ./tools/tileatlas "$PGF" workplace/tiles-$STYLE.png >/dev/null

echo
echo "== 逐格比對 =="
docker run --rm \
  --memory 1g --cpus 1 --pids-limit 64 --network none \
  -u "$(id -u):$(id -g)" -v "$ROOT:/src:ro" -w /src \
  simcity-dosbox:x python3 tools/shot_diff_cells.py \
  "$OUT/dos.png" "$OUT/remake.png" --min-hit "${MIN_HIT:-490}"

echo
echo "== City Form 逐區差分收據 =="
docker run --rm \
  --memory 1g --cpus 1 --pids-limit 64 --network none \
  -u "$(id -u):$(id -g)" -v "$ROOT:/src" -w /src \
  simcity-dosbox:x python3 tools/shot_diff_ui.py \
  "$OUT/dos-city-form.png" "$OUT/remake-city-form.png" --profile base \
  --state "DOS1.10/remake;scenario=$SCEN;style=$STYLE;cam=0,0;paused;layer=0;city-form-front" \
  --classification exact-state --out "$OUT/city-form-report"
