#!/usr/bin/env bash
set -uo pipefail
cd /src
export DISPLAY=:97
Xvfb :97 -screen 0 1920x1050x24 -nolisten tcp >/tmp/xvfb.log 2>&1 &
for _ in $(seq 1 40); do xdpyinfo >/dev/null 2>&1 && break; sleep 0.25; done
go build -o /tmp/chengshi ./cmd/chengshi || exit 1

# ⚠ **不要加 `-mute`。** 上一輪就是靜音跑的，所以音效那條路一次都沒被走到——
# 而工具音正是「蓋城市蓋久了」會一直觸發的東西。容器裡沒有音效裝置，
# 用 ALSA 的 null PCM 頂上：`-audio-probe` 在這個設定下回 0，
# 代表 `EnableSound` 走得完，播放器真的會被建出來。
printf 'pcm.!default { type null }\nctl.!default { type null }\n' > "$HOME/.asoundrc"

setsid /tmp/chengshi -data "workplace/dos110/SIMCITY 1.10" \
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
# 縮小倍率也要換：`buildZoom` 會對 960 張圖塊逐像素做 GPU 讀回，
# 是整個呈現層最重的一段，固定倍率跑一整輪等於沒測到它。
ZOOMS=(minus equal)
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
  [ $((RANDOM % 8)) -eq 0 ] && xdotool key --clearmodifiers \
    "${ZOOMS[$((RANDOM % 2))]}" 2>/dev/null

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
    # 門檻是**第一次量到的值的兩倍**，不是寫死的常數。
    # 畫面只有十六色，但抗鋸齒與縮放讓實測值落在 700 上下（上一輪量到
    # 694–702）；寫死 64 的話每一列都會被標成異常，等於沒有偵測器。
    [ -z "${BASE:-}" ] && BASE="$k"
    [ "$k" -gt $((BASE * 2)) ] 2>/dev/null && note="⚠ 顏色數暴增（基準 $BASE）"
    printf '%6s %8s %10s %9s  %s\n' "$((now - (END - MINUTES*60)))" "$k" "$rss" "$cpu" "$note"
    cp /tmp/s.png "/src/workplace/stress-$((now % 100000)).png" 2>/dev/null
  fi
done
kill -TERM -"$PG" 2>/dev/null; sleep 0.5; kill -KILL -"$PG" 2>/dev/null
echo "遊戲 log 尾巴："; tail -5 /tmp/game.log
# 卡死偵測器寫出來的現場（internal/ui/watchdog.go）。有就是抓到了。
for f in "$HOME"/.local/share/chengshi/chengshi-freeze-*.log /tmp/chengshi-freeze-*.log; do
  [ -f "$f" ] || continue
  echo "★ 卡死報告：$f"; cp "$f" /src/workplace/ 2>/dev/null
done
