#!/usr/bin/env bash
# 在已啟動的 Xvfb 中驗證完整版 AppImage 的正常玩家入口與滑鼠選單。
set -euo pipefail

: "${APPIMAGE:?缺少 APPIMAGE}"
: "${RECEIPT_DIR:?缺少 RECEIPT_DIR}"
mkdir -p "$RECEIPT_DIR"
LOG="$RECEIPT_DIR/appimage-player-path.log"

APPIMAGE_EXTRACT_AND_RUN=1 "$APPIMAGE" -mute >"$LOG" 2>&1 &
PID=$!
cleanup() {
  kill "$PID" 2>/dev/null || true
  wait "$PID" 2>/dev/null || true
}
trap cleanup EXIT

WID=
for _ in $(seq 1 80); do
  WID=$(xdotool search --name '^城市' 2>/dev/null | tail -n 1 || true)
  [ -n "$WID" ] && break
  kill -0 "$PID" 2>/dev/null || break
  sleep 0.25
done
[ -n "$WID" ] || { echo '找不到 AppImage 遊戲視窗' >&2; exit 1; }
xdotool windowmove "$WID" 0 0
xdotool windowfocus "$WID"
sleep 0.5
import -window "$WID" "$RECEIPT_DIR/01-title-menu.png"

# 招牌第三個按鈕 SELECT SCENARIO（原版 640×350 座標，remake 為整數三倍）。
xdotool mousemove --window "$WID" 948 714 click 1
sleep 0.5
import -window "$WID" "$RECEIPT_DIR/02-scenario-menu.png"

# 第一個劇本縮圖。成功後會進城市並顯示劇本簡介。
xdotool mousemove --window "$WID" 243 360 click 1
sleep 0.5
import -window "$WID" "$RECEIPT_DIR/03-scenario-brief.png"
xdotool key --window "$WID" space
sleep 0.3

# OPTIONS 標題中心，再點第 4 索引「調整速度」。
xdotool mousemove --window "$WID" 750 27 click 1
sleep 0.2
xdotool mousemove --window "$WID" 750 252 click 1
sleep 0.4
import -window "$WID" "$RECEIPT_DIR/04-speed-menu.png"

# 速度浮動視窗第 4 列（暫停）。這一擊先前完全沒有 consumer。
xdotool mousemove --window "$WID" 600 369 click 1
sleep 0.4
import -window "$WID" "$RECEIPT_DIR/05-speed-picked.png"

different() {
  local a=$1 b=$2 label=$3 raw_metric metric
  raw_metric=$(compare -metric AE "$a" "$b" null: 2>&1 || true)
  metric=$(awk -v value="$raw_metric" 'BEGIN {
    if (value !~ /^[0-9]+([.][0-9]+)?([eE][+-]?[0-9]+)?$/) exit 1
    printf "%.0f", value + 0
  }') || {
    echo "$label 的畫面差分不是數值：$raw_metric" >&2
    return 1
  }
  [ "$metric" -gt 1000 ] || { echo "$label 沒有形成可辨識畫面轉移：$metric" >&2; return 1; }
  printf 'pass  %s（差異 %s pixels）\n' "$label" "$metric"
}

different "$RECEIPT_DIR/01-title-menu.png" "$RECEIPT_DIR/02-scenario-menu.png" \
  'AppImage 啟動先進招牌選單，滑鼠可進劇本選單'
different "$RECEIPT_DIR/02-scenario-menu.png" "$RECEIPT_DIR/03-scenario-brief.png" \
  '劇本選單可由滑鼠進入正常遊戲路徑'
different "$RECEIPT_DIR/04-speed-menu.png" "$RECEIPT_DIR/05-speed-picked.png" \
  'OPTIONS→調整速度的細項可由滑鼠選取並關閉'

grep -Eqi 'panic|runtime error' "$LOG" && { echo 'AppImage 玩家路徑有 panic' >&2; exit 1; }
printf 'pass  AppImage 玩家路徑沒有 panic\n'
