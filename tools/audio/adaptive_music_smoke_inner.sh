#!/usr/bin/env bash
set -euo pipefail

cat >/tmp/.asoundrc <<'EOF'
pcm.!default { type null }
EOF

Xvfb :99 -screen 0 1280x960x24 -nolisten tcp >/tmp/adaptive-xvfb.log 2>&1 &
XVFB=$!
trap 'kill ${GAME:-} "$XVFB" 2>/dev/null || true' EXIT
export DISPLAY=:99
for _ in $(seq 1 40); do xdpyinfo >/dev/null 2>&1 && break; sleep 0.25; done

go build -o /tmp/chengshi ./cmd/chengshi
/tmp/chengshi -data "workplace/dos110/SIMCITY 1.10" -music music -seed 19 -scale 1 \
  >/tmp/adaptive-music.log 2>&1 &
GAME=$!

for _ in $(seq 1 80); do
  xdotool search --name '城市' >/dev/null 2>&1 && break
  kill -0 "$GAME" 2>/dev/null || break
  sleep 0.1
done
WID=$(xdotool search --name '城市' 2>/dev/null | head -n 1 || true)
[ -n "$WID" ] || {
  sed '/^XGB:/d' /tmp/adaptive-music.log
  echo 'FAIL：遊戲視窗沒有出現' >&2
  exit 1
}
xdotool windowfocus --sync "$WID"
sleep 3

# 正常玩家路徑：開災難選單，第一列就是火災，再按 Enter。若隨機落點沒找到
# 可燃格，重試最多四次；成功時模擬層才會送 MsgFire，播放器才可切歌。
for _ in $(seq 1 4); do
  xdotool windowfocus "$WID" 2>/dev/null || true
  xdotool key --clearmodifiers alt+d
  sleep 0.08
  xdotool key --clearmodifiers Return
  for _ in $(seq 1 8); do
    grep -q 'SC2000-10004.ogg' /tmp/adaptive-music.log && break 2
    sleep 0.05
  done
done

grep -q '情境配樂：事件 1 → SC2000-10004.ogg' /tmp/adaptive-music.log || {
  sed '/^XGB:/d' /tmp/adaptive-music.log
  echo 'FAIL：Alt-D 火災沒有切到固定曲 SC2000-10004.ogg' >&2
  exit 1
}

kill "$GAME" 2>/dev/null || true
wait "$GAME" 2>/dev/null || true
echo 'pass：Alt-D 火災切到 SC2000-10004.ogg，解碼器與播放器成功啟動'
