#!/usr/bin/env bash
# 在 docker + Xvfb 裡跑遊戲並截圖。
# 用法：tools/screenshot.sh [等待秒數] [輸出檔名]
#   GAME_ARGS="…"  傳給遊戲的參數
#   GAME_KEYS="…"  截圖前先送這幾個鍵（空白分隔的 xdotool 鍵名）
#   GAME_CLICKS="x,y …" 截圖前依序點擊螢幕座標
#   GAME_CLICK_WAIT="秒" 每次點擊後等待（預設 1）
#   GAME_STABLE=1 靜態畫面必須連續兩張逐位元相同才接受
#   GAME_HOLD="x,y" 截圖前把滑鼠按在這個螢幕座標上不放（下拉選單是按住式的）
#   GAME_KEYDOWN="q" 截圖前按住這個鍵不放（查詢是「按住 Q ＋ 按住左鍵」）
#   GAME_CONFIG_HOME="容器路徑" 隔離玩家設定目錄（持久化驗收用）
#   GAME_BIN="容器路徑" 改用既有的執行檔（發行包／AppImage）代替 `go run`。
#     驗的就不再是工作樹的原始碼，而是**打包後的那顆二進位**。
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
  -e XDG_CONFIG_HOME="${GAME_CONFIG_HOME:-/tmp/chengshi-config}" \
  -e LIBGL_ALWAYS_SOFTWARE=1 -e GALLIUM_DRIVER=llvmpipe \
  -w /src simcity-go:1.25 \
  bash -c "
    set -e
    Xvfb :99 -screen 0 1920x1050x24 -nolisten tcp >/tmp/xvfb.log 2>&1 &
    export DISPLAY=:99
    for i in \$(seq 1 40); do xdpyinfo >/dev/null 2>&1 && break; sleep 0.25; done
    # AppImage 在容器裡沒有 FUSE，用官方的解包執行模式。
    export APPIMAGE_EXTRACT_AND_RUN=1
    LAUNCH='${GAME_BIN:-}'
    if [ -n \"\$LAUNCH\" ]; then
      \"\$LAUNCH\" -data 'workplace/dos110/SIMCITY 1.10' -mute ${GAME_ARGS:-} \
          >/tmp/game.log 2>&1 &
    else
      go run ./cmd/chengshi -data 'workplace/dos110/SIMCITY 1.10' -mute ${GAME_ARGS:-} \
          >/tmp/game.log 2>&1 &
    fi
    GAME=\$!
    sleep $WAIT
    # ⚠ 組合鍵不能用 \`xdotool key ctrl+c\`：按下與放開會落在同一個輪詢畫格裡，
    # 遊戲讀到 C 的時候 Ctrl 可能還沒按下或已經放開，快捷鍵就**靜默失效**——
    # 畫面看起來正常，只是該關的視窗還開著，而截圖之後才發現。
    for k in ${GAME_KEYS:-}; do
      case \"\$k\" in
        *+*)
          xdotool keydown \"\${k%%+*}\"; sleep 0.2
          xdotool key \"\${k##*+}\"; sleep 0.2
          xdotool keyup \"\${k%%+*}\"
          ;;
        *) xdotool key --clearmodifiers \$k ;;
      esac
      sleep 1
    done
    for p in ${GAME_CLICKS:-}; do
      xdotool mousemove \${p%,*} \${p#*,}; xdotool click 1; sleep ${GAME_CLICK_WAIT:-1}
    done
    KD='${GAME_KEYDOWN:-}'
    if [ -n \"\$KD\" ]; then xdotool keydown \$KD; sleep 0.3; fi
    HOLD='${GAME_HOLD:-}'
    if [ -n \"\$HOLD\" ]; then
      xdotool mousemove \${HOLD%,*} \${HOLD#*,}
      sleep 0.3; xdotool mousedown 1; sleep 1
    fi
    if kill -0 \$GAME 2>/dev/null; then
      if [ '${GAME_STABLE:-0}' = 1 ]; then
        prev=''
        stable=''
        for i in \$(seq 1 20); do
          cur="/tmp/stable-\$i.png"
          xwd -root -silent | convert xwd:- "\$cur"
          if [ -n "\$prev" ]; then
            ae=\$(compare -metric AE "\$prev" "\$cur" null: 2>&1 || true)
            ae=\${ae%% *}
            if [ "\$ae" = 0 ]; then
              stable="\$cur"
              break
            fi
          fi
          prev="\$cur"
          sleep 0.1
        done
        if [ -z "\$stable" ]; then
          mkdir -p workplace/shots/stable-debug
          cp /tmp/stable-*.png workplace/shots/stable-debug/
          echo 'FAIL：20 次擷取沒有連續兩張完全相同，拒絕使用可能撕裂的畫面'
          kill \$GAME 2>/dev/null || true
          exit 1
        fi
        cp "\$stable" workplace/shots/$SHOT
      else
        xwd -root -silent | convert xwd:- workplace/shots/$SHOT
      fi
      kill \$GAME 2>/dev/null || true
      echo '== 截圖完成 =='
    else
      echo '== 遊戲已結束 =='
    fi
    tail -20 /tmp/game.log
  "
