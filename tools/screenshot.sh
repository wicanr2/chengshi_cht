#!/usr/bin/env bash
# 在 docker + Xvfb 裡跑遊戲並截圖。用法：tools/screenshot.sh [等待秒數] [輸出檔名]
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WAIT="${1:-6}"
SHOT="${2:-game.png}"
mkdir -p "$ROOT/workplace/shots"
docker run --rm \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" \
  --memory 4g --cpus 4 --pids-limit 512 \
  --network none \
  -v "$ROOT:/src" \
  -v "$ROOT/workplace/gocache:/gocache" -v "$ROOT/workplace/gomod:/gomod" \
  -e GOCACHE=/gocache -e GOMODCACHE=/gomod -e GOFLAGS=-mod=mod -e HOME=/tmp \
  -e LIBGL_ALWAYS_SOFTWARE=1 -e GALLIUM_DRIVER=llvmpipe \
  -w /src simcity-go:1.25 \
  bash -c "
    set -e
    Xvfb :99 -screen 0 1280x960x24 -nolisten tcp >/tmp/xvfb.log 2>&1 &
    export DISPLAY=:99
    for i in \$(seq 1 40); do xdpyinfo >/dev/null 2>&1 && break; sleep 0.25; done
    go run ./cmd/chengshi -data 'workplace/dos110/SIMCITY 1.10' ${GAME_ARGS:-} \
        >/tmp/game.log 2>&1 &
    GAME=\$!
    sleep $WAIT
    if kill -0 \$GAME 2>/dev/null; then
      import -window root workplace/shots/$SHOT 2>/dev/null || \
        xwd -root -silent | convert xwd:- workplace/shots/$SHOT
      kill \$GAME 2>/dev/null || true
      echo '== 截圖完成 =='
    else
      echo '== 遊戲已結束 =='
    fi
    tail -20 /tmp/game.log
  "
