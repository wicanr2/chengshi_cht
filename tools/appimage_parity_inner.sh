#!/usr/bin/env bash
# tools/appimage_parity.sh 第一階段的容器內本體：**同源證明**。
#
# 同一個 Xvfb、同一份城市檔、同一組輸入，分別餵給
#   （甲）工作樹原始碼用發行旗標現建的執行檔
#   （乙）發行包裡那顆 AppImage
# 然後逐像素比。兩邊相同，既有的原版對拍數字才能整批繼承到發行包。
#
# ⚠ **這個遊戲的畫面本身就不是逐次可重播的**，判準必須把這件事算進去：
#
#   1. 模擬在劇本簡介蓋著的時候照跑（`updateTitle` 只擋招牌與劇本選單），
#      而 AppImage 比現建版多花一兩秒解包——用固定秒數等待的話，兩邊
#      「拍照時已經跑了幾刻」不一樣。對策是等視窗出現、並讀一份存成
#      `SimSpeed=0` 的城市檔，讓 `world.Frame()` 完全不前進。
#   2. 就算世界凍住了，**讀檔本身會擲亂數**：`game.LoadCity` 走的是
#      `LoadCitySeed(path, sim.RandomSeed())`，而 `RandomSeed()` 是
#      `time.Now().UnixNano()`（`internal/sim/rand.go:147`）；`DoSimInit`
#      的那次 `MapScan` 用得到它。`-seed` 只接到「開新城市」那條路，
#      `-load` 與 `-scenario` 都沒有（`cmd/chengshi/main.go:165`）。
#      所以同一顆二進位讀同一份檔，兩次啟動的畫面就已經會差幾百個像素。
#
# 對策是**成對重複量測**：兩側各拍兩張，先量出「自己跟自己」的雜訊底線，
# 再看「乙側對甲側」的最小差距有沒有超過它。沒有世界的畫面雜訊底線是 0，
# 判準自動收斂回逐像素相同。
#
# 環境變數：
#   A_KIND／B_KIND  dev 現建版／app 發行 AppImage。設成相同就是正對照。
#   ONLY            只跑這幾幕（空白分隔）
set -uo pipefail

: "${APPIMAGE:?缺少 APPIMAGE}"
: "${VER:?缺少 VER}"
OUT="${OUT:-/src/workplace/appimage-parity}"
A_KIND="${A_KIND:-dev}"
B_KIND="${B_KIND:-app}"
ONLY="${ONLY:-}"
DATA='workplace/dos110/SIMCITY 1.10'
PAUSED="$OUT/paused-scen1.cty"
mkdir -p "$OUT/shots" "$OUT/reports"
cd /src

export APPIMAGE_EXTRACT_AND_RUN=1
export DISPLAY=:99
Xvfb :99 -screen 0 1920x1050x24 -nolisten tcp >/tmp/xvfb.log 2>&1 &
for _ in $(seq 1 40); do xdpyinfo >/dev/null 2>&1 && break; sleep 0.25; done

FAIL=0
pass() { printf 'pass  %s\n' "$*"; }
fail() { printf 'FAIL  %s\n' "$*"; FAIL=1; }

echo "== 第 0 步：二進位身分 =="
mkdir -p /tmp/ax && (cd /tmp/ax && "$APPIMAGE" --appimage-extract >/dev/null 2>&1)
APPBIN=/tmp/ax/squashfs-root/usr/bin/chengshi
[ -x "$APPBIN" ] || { echo "AppImage 裡沒有可執行的 chengshi" >&2; exit 1; }
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
  go build -ldflags "-s -w -X main.version=$VER" -o /tmp/chengshi-dev ./cmd/chengshi \
  || { echo "現建失敗" >&2; exit 1; }
A_SHA=$(sha256sum "$APPBIN" | cut -d' ' -f1)
D_SHA=$(sha256sum /tmp/chengshi-dev | cut -d' ' -f1)
echo "AppImage 內建執行檔 : $A_SHA"
echo "工作樹現建執行檔   : $D_SHA"
if [ "$A_SHA" = "$D_SHA" ]; then
  pass "兩顆執行檔逐位元相同"
else
  echo "note  兩顆雜湊不同（建置路徑與工具鏈版本都會改變輸出），改由畫面判定"
fi

GAME_PGID=

launch() { # $1=kind $2=args
  local kind=$1 args=$2
  if [ "$kind" = dev ]; then
    setsid /tmp/chengshi-dev -data "$DATA" -mute $args >/tmp/game-$kind.log 2>&1 &
  else
    setsid "$APPIMAGE" -data "$DATA" -mute $args >/tmp/game-$kind.log 2>&1 &
  fi
  GAME_PGID=$!
}

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
  sleep 0.5
}

# 等視窗真的出現，不要死等固定秒數：AppImage 要多花一兩秒解包。
await_window() {
  for _ in $(seq 1 400); do
    xdotool search --name '^城市' >/dev/null 2>&1 && return 0
    sleep 0.05
  done
  return 1
}

# 靜態畫面：連續兩張解碼像素完全相同才收，避免把撕裂幀當成證據。
grab_stable() {
  local out=$1 prev='' cur ae
  for i in $(seq 1 30); do
    cur=/tmp/stable-$i.png
    xwd -root -silent | convert xwd:- "$cur" 2>/dev/null || continue
    if [ -n "$prev" ]; then
      ae=$(compare -metric AE "$prev" "$cur" null: 2>&1 || true)
      if [ "${ae%% *}" = 0 ]; then cp "$cur" "$out"; rm -f /tmp/stable-*.png; return 0; fi
    fi
    prev=$cur
    sleep 0.15
  done
  rm -f /tmp/stable-*.png
  return 1
}

# ── 準備可重播的起點：一份存成暫停狀態的城市檔 ──────────────────────
prepare_paused_city() {
  echo
  echo "== 準備可重播起點：把劇本 1 存成 SimSpeed=0 的城市檔 =="
  rm -f "$PAUSED"
  launch dev "-scenario 1 -style west -cam 0,0 -save $PAUSED"
  await_window || { echo "存檔用的遊戲沒開起來" >&2; stop_game; return 1; }
  xdotool key --clearmodifiers 0      # 先暫停，再關簡介
  sleep 0.5
  xdotool key --clearmodifiers space  # 關掉劇本簡介
  sleep 1
  xdotool key --clearmodifiers 0      # 補一次，確保是暫停
  sleep 1
  xdotool key --clearmodifiers ctrl+s
  sleep 3
  stop_game
  if [ -s "$PAUSED" ]; then
    pass "起點城市檔已產生（$(stat -c%s "$PAUSED") 位元組，SHA-256 $(sha256sum "$PAUSED" | cut -c1-16)…）"
  else
    fail "起點城市檔沒產生"
    return 1
  fi
}

shoot() { # kind name wait args keys clicks postkeys hold keydown
  local kind=$1 name=$2 wait=$3 args=$4 keys=$5 clicks=$6 postkeys=$7 hold=$8 keydown=$9
  launch "$kind" "$args"
  if ! await_window; then
    fail "$name/$kind 沒有出現遊戲視窗"
    tail -5 /tmp/game-$kind.log
    stop_game
    return 1
  fi
  sleep "$wait"
  # 送鍵會掉。量到過：預算那一幕四張裡有一張根本沒開到視窗，而畫面本身
  # 完全正常——沒有自我檢查的話，那張會被當成「兩顆二進位畫得不一樣」。
  # 所以送完鍵要確認畫面真的變了，沒變就補送一次。
  # （`xdotool windowfocus` 試過，反而讓 graphs／eval 也開始掉鍵，不用。）
  if [ -n "$keys" ]; then
    xwd -root -silent | convert xwd:- /tmp/pre.png 2>/dev/null
    for k in $keys; do xdotool key --clearmodifiers "$k"; sleep 0.6; done
    sleep 0.4
    xwd -root -silent | convert xwd:- /tmp/post.png 2>/dev/null
    local ae
    ae=$(compare -metric AE /tmp/pre.png /tmp/post.png null: 2>&1 || true)
    if [ "${ae%% *}" = 0 ]; then
      echo "      note $name/$kind 送鍵沒有造成畫面變化，補送一次"
      for k in $keys; do xdotool key --clearmodifiers "$k"; sleep 0.6; done
    fi
  fi
  for p in $clicks; do
    xdotool mousemove "${p%,*}" "${p#*,}"; xdotool click 1; sleep 1
  done
  for k in $postkeys; do xdotool key --clearmodifiers "$k"; sleep 0.6; done
  # 滑鼠停到畫面外，游標形狀不進畫面。
  [ -z "$hold" ] && xdotool mousemove 1919 1049
  [ -n "$keydown" ] && { xdotool keydown "$keydown"; sleep 0.3; }
  if [ -n "$hold" ]; then
    xdotool mousemove "${hold%,*}" "${hold#*,}"; sleep 0.3; xdotool mousedown 1; sleep 1
  fi
  grab_stable "$OUT/shots/$name-$kind.png" \
    || fail "$name/$kind 三十次擷取都沒有連續兩張相同（畫面持續在動）"
  [ -n "$hold" ] && xdotool mouseup 1
  [ -n "$keydown" ] && xdotool keyup "$keydown"
  stop_game
  return 0
}

prepare_paused_city || exit 1

# ── 場景表：名稱|等待秒|參數|前置鍵|點擊|後置鍵|按住|按住鍵 ────────────
CITY="-load $PAUSED -style west -cam 0,0"
SCENES=(
  "title|3||||||"
  "scenmenu|3|||948,714|||"
  "brief|3|||948,714 243,360|0||"
  "newcity|3|||606,594|||"
  "edit|3|$CITY|ctrl+c||||"
  "cityform|3|$CITY|||||"
  "menu-system|3|$CITY||||336,24|"
  "menu-options|3|$CITY||||750,24|"
  "menu-disasters|3|$CITY||||1206,24|"
  "menu-windows|3|$CITY||||1662,24|"
  "graphs|3|$CITY|ctrl+g||||"
  "budget|3|$CITY|ctrl+b||||"
  "eval|3|$CITY|ctrl+u||||"
  "query|3|$CITY||||300,300|q"
)

echo
echo "== 第 1 步：十四幕，$A_KIND 對 $B_KIND（每側各三張）=="
for rec in "${SCENES[@]}"; do
  IFS='|' read -r name wait args keys clicks postkeys hold keydown <<<"$rec"
  if [ -n "$ONLY" ] && ! printf '%s\n' $ONLY | grep -qx "$name"; then continue; fi
  printf '\n-- %s --\n' "$name"
  for r in 1 2 3; do
    shoot "$A_KIND" "$name-a$r" "$wait" "$args" "$keys" "$clicks" "$postkeys" "$hold" "$keydown"
    shoot "$B_KIND" "$name-b$r" "$wait" "$args" "$keys" "$clicks" "$postkeys" "$hold" "$keydown"
  done
  A1="$OUT/shots/$name-a1-$A_KIND.png"; A2="$OUT/shots/$name-a2-$A_KIND.png"
  A3="$OUT/shots/$name-a3-$A_KIND.png"
  B1="$OUT/shots/$name-b1-$B_KIND.png"; B2="$OUT/shots/$name-b2-$B_KIND.png"
  B3="$OUT/shots/$name-b3-$B_KIND.png"
  if [ -f "$A1" ] && [ -f "$A2" ] && [ -f "$A3" ] \
     && [ -f "$B1" ] && [ -f "$B2" ] && [ -f "$B3" ]; then
    if out=$(python3 tools/shot_pairset.py --a "$A1" "$A2" "$A3" --b "$B1" "$B2" "$B3" \
               --label "$name" --out "$OUT/reports/$name.json"); then
      pass "$name：$B_KIND 與 $A_KIND 無可分辨差異  $out"
    else
      fail "$name：$B_KIND 與 $A_KIND 差距超過雜訊底線  $out"
    fi
  else
    fail "$name：缺截圖，無法比對"
  fi
done

echo
if [ "$FAIL" = 0 ]; then echo "== 同源證明：全數通過 =="; else echo "== 同源證明：有未通過項 =="; fi
exit "$FAIL"
