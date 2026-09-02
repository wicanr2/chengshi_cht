#!/usr/bin/env bash
# tools/promo_video.sh 的第一階段：在 Xvfb 裡跑遊戲、逐格擷取，順便合成配樂。
#
# 編碼在第二階段（另一個 image，那邊才有 ffmpeg），這裡只出 PNG 序列與 WAV。
#
# 運鏡用**按住方向鍵**而不是每格重開遊戲：方向鍵直接捲鏡頭（scrollDir），
# 按住不放就是連續運鏡，一次啟動拍得完一整段。每格重開的話光是啟動就要
# 八秒，而且鏡頭是跳的不是移的。
set -uo pipefail

OUT="${OUT:-/src/workplace/promo/frames}"
DATA='workplace/dos110/SIMCITY 1.10'
FPS="${FPS:-12}"
rm -rf "$OUT"
mkdir -p "$OUT"
cd /src

export DISPLAY=:99
Xvfb :99 -screen 0 1920x1050x24 -nolisten tcp >/tmp/xvfb.log 2>&1 &
for _ in $(seq 1 40); do xdpyinfo >/dev/null 2>&1 && break; sleep 0.25; done

go build -o /tmp/chengshi ./cmd/chengshi || exit 1
go build -o /tmp/promomusic ./tools/promomusic || exit 1

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
#
# 速度用 SPEED 環境變數指定，預設 0（暫停）。巡覽地形時要暫停：模擬跑起來
# 的話每一格的城市都不一樣，那在影片裡是雜訊不是內容。**遊玩段落相反**，
# 要留著速度，圖塊動畫的條件就是 `DoAnimation && SimSpeed != 0`
# （`internal/ui/game.go:497`），暫停時整個畫面是死的。
scene() {
  local frames=$1 hold=$2 speed="${SPEED:-0}"
  shift 2
  setsid /tmp/chengshi -data "$DATA" -mute "$@" >/tmp/game.log 2>&1 &
  GAME_PGID=$!
  local ok=0
  for _ in $(seq 1 600); do
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
  xdotool key --clearmodifiers "$speed"
  sleep 0.6
  xdotool mousemove 1919 1049
  sleep 0.3
  [ -n "$hold" ] && xdotool keydown "$hold"
  for _ in $(seq 1 "$frames"); do grab; done
  [ -n "$hold" ] && xdotool keyup "$hold"
  stop_game
  printf '  %-38s %3s 格（累計 %d）\n' "$*" "$frames" "$N"
}

# play_scene <幀數> <遊戲參數…>：實機遊玩段落。
#
# 跟 scene 有三處不一樣：
#   一、留著模擬速度。圖塊動畫的條件是 `DoAnimation && SimSpeed != 0`
#       （`internal/ui/game.go:497`），暫停時整個畫面是死的。
#   二、每格之前送一次空白鍵。城市跑起來就會跳訊息框，而訊息框不只擋畫面，
#       還會吃掉所有按鍵（`game.go:635` 那個 `g.picture == ""` 條件）。
#   三、開場十格之後送 `Ctrl-C` 關掉 City Form 視窗。`-demo` 用 `LookAt`
#       把鏡頭對準新蓋的城市，而那個中心點在 City Form 底下——視窗關掉
#       才看得到城市。前十格留著視窗是為了先看清楚這是哪一張地圖。
play_scene() {
  local frames=$1 speed="${SPEED:-2}" reveal="${REVEAL:-10}"
  shift
  setsid /tmp/chengshi -data "$DATA" -mute "$@" >/tmp/game.log 2>&1 &
  GAME_PGID=$!
  local ok=0
  for _ in $(seq 1 900); do
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
  xdotool key --clearmodifiers space
  sleep 0.3
  xdotool key --clearmodifiers "$speed"
  sleep 0.4
  xdotool mousemove 1919 1049
  sleep 0.3
  local i
  for i in $(seq 1 "$frames"); do
    # Ctrl-C 關掉 City Form。**不能用 `xdotool key ctrl+c`**——那個寫法
    # 按下與放開幾乎在同一瞬間，遊戲是逐畫格輪詢按鍵的，抓不抓得到看運氣
    # （實際拍過一次，視窗完全沒關）。改成把 Control 按住跨幾個畫格再送 c。
    if [ "$i" = "$((reveal + 1))" ]; then
      xdotool keydown ctrl; sleep 0.2
      xdotool key c;        sleep 0.2
      xdotool keyup ctrl;   sleep 0.3
    fi
    xdotool key --clearmodifiers space
    grab
  done
  stop_game
  printf '  %-38s %3s 格（累計 %d）\n' "$*" "$frames" "$N"
}

echo "== 擷取 =="
# 開場：原版的招牌畫面。
#
# ⚠ **不能帶 `-cam`。** 招牌只有在玩家沒指定要玩哪一座城時才會出現
# （`cmd/chengshi/main.go`：`-load`／`-scenario`／`-demo`／`-window`／`-cam`／
# `-seed` 任一有值就直接進城市）。帶了 `-cam 0,0` 拍到的是隨機地形，
# 而且畫面看起來完全正常——所以這種錯只有真的看畫格才抓得到。
scene 12 ""
# 實機遊玩：在台灣地圖上蓋一座起始城市、快轉 25 年，然後留著速度讓它繼續跑。
play_scene 36 -load cities/TAIWAN.CTY -demo 25
# 地形巡覽：暫停，只有鏡頭在動。
scene 26 Right -load cities/TAIWAN.CTY -cam 0,30
scene 18 Right -load cities/TAIPEI.CTY -cam 0,28
scene 18 Right -load cities/TAICHUNG.CTY -cam 0,28
scene 18 Right -load cities/TAINAN.CTY -cam 0,28
scene 18 Right -load cities/KAOHSIUN.CTY -cam 0,28
# 高雄收尾，也是實機遊玩。
play_scene 30 -load cities/KAOHSIUN.CTY -demo 20

echo "共 $N 格，在 $OUT"

# 配樂長度配合畫格數，尾巴多留一點讓它淡出。
SEC=$(awk -v n="$N" -v f="$FPS" 'BEGIN{printf "%.3f", n/f + 0.4}')
/tmp/promomusic -sec "$SEC" -o /src/workplace/promo/music.wav || exit 1
