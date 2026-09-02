#!/usr/bin/env bash
# tools/promo_video.sh 的第一階段：在 Xvfb 裡跑遊戲、逐格擷取。
#
# 編碼在第二階段（另一個 image，那邊才有 ffmpeg），這裡只出 PNG 序列。
#
# 運鏡用**按住方向鍵**而不是每格重開遊戲：方向鍵直接捲鏡頭（scrollDir），
# 按住不放就是連續運鏡，一次啟動拍得完一整段。每格重開的話光是啟動就要
# 八秒，而且鏡頭是跳的不是移的。
set -uo pipefail

OUT="${OUT:-/src/workplace/promo/frames}"
DATA='workplace/dos110/SIMCITY 1.10'
rm -rf "$OUT"
mkdir -p "$OUT"
cd /src

export DISPLAY=:99
Xvfb :99 -screen 0 1920x1050x24 -nolisten tcp >/tmp/xvfb.log 2>&1 &
for _ in $(seq 1 40); do xdpyinfo >/dev/null 2>&1 && break; sleep 0.25; done

go build -o /tmp/chengshi ./cmd/chengshi || exit 1

N=0
GAME_PGID=

stop_game() {
  [ -n "$GAME_PGID" ] || return 0
  kill -TERM -"$GAME_PGID" 2>/dev/null
  for _ in $(seq 1 40); do
    xdotool search --name '^城市' >/dev/null 2>&1 || break
    sleep 0.25
  done
  kill -KILL -"$GAME_PGID" 2>/dev/null
  wait "$GAME_PGID" 2>/dev/null
  GAME_PGID=
  sleep 0.4
}

grab() {
  N=$((N + 1))
  xwd -root -silent | convert xwd:- "$(printf '%s/%04d.png' "$OUT" "$N")" 2>/dev/null
}

# scene <幀數> <按住的鍵，空的話靜止> <遊戲參數…>
scene() {
  local frames=$1 hold=$2
  shift 2
  setsid /tmp/chengshi -data "$DATA" -mute "$@" >/tmp/game.log 2>&1 &
  GAME_PGID=$!
  local ok=0
  for _ in $(seq 1 400); do
    if xdotool search --name '^城市' >/dev/null 2>&1; then ok=1; break; fi
    sleep 0.05
  done
  if [ "$ok" = 0 ]; then
    echo "場景沒開起來：$*" >&2
    tail -3 /tmp/game.log >&2
    stop_game
    return 1
  fi
  sleep 2.5
  # 暫停，讓畫面只有運鏡在動——模擬跑起來的話每一格的城市都不一樣，
  # 那在影片裡是雜訊不是內容。
  xdotool key --clearmodifiers 0
  sleep 0.6
  xdotool mousemove 1919 1049
  sleep 0.3
  [ -n "$hold" ] && xdotool keydown "$hold"
  for _ in $(seq 1 "$frames"); do grab; done
  [ -n "$hold" ] && xdotool keyup "$hold"
  stop_game
  printf '  %-10s %s 格（累計 %d）\n' "${*: -1}" "$frames" "$N"
}

echo "== 擷取 =="
# 開場：原版的招牌畫面。
#
# ⚠ **不能帶 `-cam`。** 招牌只有在玩家沒指定要玩哪一座城時才會出現
# （`cmd/chengshi/main.go`：`-load`／`-scenario`／`-demo`／`-window`／`-cam`／
# `-seed` 任一有值就直接進城市）。帶了 `-cam 0,0` 拍到的是隨機地形，
# 而且畫面看起來完全正常——所以這種錯只有真的看畫格才抓得到。
scene 12 "" 
# 台灣：從西岸往東橫越。
scene 26 Right -load cities/TAIWAN.CTY -cam 0,30
# 三張本專案畫的。
scene 18 Right -load cities/TAIPEI.CTY -cam 0,28
scene 18 Right -load cities/TAICHUNG.CTY -cam 0,28
scene 18 Right -load cities/TAINAN.CTY -cam 0,28
# 高雄收尾。
scene 18 Right -load cities/KAOHSIUN.CTY -cam 0,28

echo "共 $N 格，在 $OUT"
