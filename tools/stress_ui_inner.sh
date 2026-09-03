#!/usr/bin/env bash
set -uo pipefail
cd /src
export DISPLAY=:97
Xvfb :97 -screen 0 1920x1050x24 -nolisten tcp >/tmp/xvfb.log 2>&1 &
for _ in $(seq 1 40); do xdpyinfo >/dev/null 2>&1 && break; sleep 0.25; done
go build -o /tmp/chengshi ./cmd/chengshi || exit 1

setsid /tmp/chengshi -data "workplace/dos110/SIMCITY 1.10" -mute \
  -load cities/TAIWAN.CTY -demo 20 >/tmp/game.log 2>&1 &
PG=$!
for _ in $(seq 1 900); do xdotool search --name '^城市' >/dev/null 2>&1 && break; sleep 0.05; done
PID=$(pgrep -n -f '/tmp/chengshi' || true)
[ -n "$PID" ] || { echo "遊戲沒起來"; tail -5 /tmp/game.log; exit 1; }
sleep 3
xdotool key --clearmodifiers space; sleep 0.3
xdotool key --clearmodifiers 4; sleep 0.3   # 最快

END=$(( $(date +%s) + MINUTES*60 ))
NEXT=0
TOOLS=(z x v r p g o f k t)
printf '%6s %8s %10s %9s  %s\n' 秒 顏色數 RSS_kB CPU秒 備註
while [ "$(date +%s)" -lt "$END" ]; do
  # 一輪操作：換工具、在地圖上點幾下
  t=${TOOLS[$((RANDOM % ${#TOOLS[@]}))]}
  xdotool key --clearmodifiers "$t" 2>/dev/null
  for _ in 1 2 3 4 5 6; do
    x=$((300 + RANDOM % 1400)); y=$((200 + RANDOM % 700))
    xdotool mousemove $x $y click 1 2>/dev/null
  done
  xdotool key --clearmodifiers space 2>/dev/null

  now=$(date +%s)
  if [ "$now" -ge "$NEXT" ]; then
    NEXT=$((now + 10))
    if ! kill -0 "$PID" 2>/dev/null; then
      echo "行程不見了（崩潰）"; tail -20 /tmp/game.log; break
    fi
    xwd -root -silent | convert xwd:- /tmp/s.png 2>/dev/null
    k=$(identify -format %k /tmp/s.png 2>/dev/null || echo -1)
    rss=$(awk '/VmRSS/{print $2}' /proc/$PID/status 2>/dev/null || echo -1)
    cpu=$(awk '{printf "%.1f", ($14+$15)/100}' /proc/$PID/stat 2>/dev/null || echo -1)
    note=""
    [ "$k" -gt 64 ] 2>/dev/null && note="⚠ 顏色數異常"
    printf '%6s %8s %10s %9s  %s\n' "$((now - (END - MINUTES*60)))" "$k" "$rss" "$cpu" "$note"
    cp /tmp/s.png "/src/workplace/stress-$((now % 100000)).png" 2>/dev/null
  fi
done
kill -TERM -"$PG" 2>/dev/null; sleep 0.5; kill -KILL -"$PG" 2>/dev/null
echo "遊戲 log 尾巴："; tail -5 /tmp/game.log
