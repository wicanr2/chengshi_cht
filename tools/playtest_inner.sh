#!/usr/bin/env bash
# 在容器裡跑的試玩腳本本體。不要直接執行，用 tools/playtest.sh。
#
# 走的是玩家真的會走的路徑：開新城市 → 選工具 → 在地圖上蓋 →
# 開四個視窗 → 查詢 → 捲動 → 存檔 → 離開 → 重開讀檔 → 劇本。
# 沒有 debug hook、沒有 -demo。
#
# ⚠ 每一步都是「做完看畫面，沒變就再做一次」，不是「按下去然後睡一秒」。
# 這台機器上同時有別的工作，遊戲隨時可能被餓住幾百毫秒；固定等待在
# 空機上會過關、在忙的時候隨機掉鍵，而掉的每次都是不同的鍵——看起來
# 像遊戲會吃鍵，其實是腳本自己的問題。
set -euo pipefail

OUT=workplace/shots/playtest
SAVE=/tmp/pt/city.cty
FAIL=0
mkdir -p "$OUT" /tmp/pt

# 相機開場置中：120×100 的地圖、32×24 格的視野 → 左上角在 (44,38)。
# 格子座標 → 畫面座標。
#
# ⚠ 版面換成原版的之後，地圖不再從畫面左上角開始：它在編輯視窗裡，
# 原點是 (64,54) 的原版座標，畫布放大三倍（internal/ui/classic.go）。
# 少加這個位移的話，每一次點擊都落在離目標好幾格的地方，而遊戲照樣
# 蓋得出東西——症狀是「試玩腳本蓋的城市長得不對」，不是「點不到」。
# 相機一開始置中：camX ＝ (120 − 11) / 2、camY ＝ (100 − 16) / 2
# （見 internal/ui 的 centerCamera，可見格數是編輯視窗算出來的）。
UIS=3; VIEWX=64; VIEWY=54
CAMX=54; CAMY=42; PX=$((16 * UIS))
sx() { echo $(( VIEWX * UIS + ($1 - CAMX) * PX + PX / 2 )); }
sy() { echo $(( VIEWY * UIS + ($1 - CAMY) * PX + PX / 2 )); }

fail() { echo "FAIL  $*"; FAIL=1; }
pass() { echo "pass  $*"; }

diffpct() { python3 tools/diffpct.py "$1" "$2"; }

start_x() {
  Xvfb :99 -screen 0 1920x1050x24 -nolisten tcp >/tmp/xvfb.log 2>&1 &
  export DISPLAY=:99
  for i in $(seq 1 40); do xdpyinfo >/dev/null 2>&1 && return; sleep 0.25; done
  echo "Xvfb 起不來"; exit 1
}

# 用 xwd 而不是 ImageMagick 的 import：import 抓圖時會抓住整個 X server，
# 那段時間送進去的按鍵直接消失，症狀是「截圖之後的下一個鍵沒反應」。
shot() { xwd -root -silent | convert xwd:- "$OUT/$1.png"; }

key() { xdotool key --clearmodifiers "$1"; sleep 0.3; }

# 點一格。按下與放開之間要留時間：Ebiten 每 1/60 秒才取樣一次輸入，
# 瞬間按放會整個落在兩次取樣之間，遊戲根本不知道有人按過。
click() {
  xdotool mousemove "$(sx "$1")" "$(sy "$2")"
  sleep 0.05; xdotool mousedown 1; sleep 0.15; xdotool mouseup 1; sleep 0.2
}

# 拖曳鋪一條線（道路／電線／鐵軌才吃得到）。
drag() { # x1 y1 x2 y2（同一列）
  xdotool mousemove "$(sx "$1")" "$(sy "$2")"; sleep 0.05
  xdotool mousedown 1; sleep 0.15
  local x=$1
  while [ "$x" -le "$3" ]; do
    xdotool mousemove "$(sx "$x")" "$(sy "$4")"; sleep 0.1
    x=$((x + 1))
  done
  xdotool mouseup 1; sleep 0.2
}

# 做一件事，做到畫面真的變了為止（最多五次）。
# 第一個參數是最小變化百分比，第二個是這一步的名字，其餘是要跑的指令。
do_until() {
  local need=$1 label=$2; shift 2
  shot _before
  for i in 1 2 3 4 5; do
    "$@"
    shot _after
    local d
    d=$(diffpct "$OUT/_before.png" "$OUT/_after.png")
    if [ "$d" -ge "$need" ]; then pass "$label（${d} 像素）"; return 0; fi
  done
  fail "$label 沒生效（畫面只差 ${d} 個像素，要 ${need}）"
  return 0
}

# 先編出執行檔再跑。
#
# ⚠ 不要用 `go run`：它會另外 fork 出真正的遊戲進程，`kill` 打到的是
# go run 自己，遊戲照樣活著。上一局的視窗殘留在畫面上，xdotool 找到的
# 是舊視窗，按鍵也送去舊視窗——截圖看起來「劇本沒載入」，其實是拍到
# 上一局。這種錯誤不會有任何錯誤訊息。
BIN=/tmp/pt/chengshi
go build -o "$BIN" ./cmd/chengshi

run_game() { # 背景啟動遊戲，等到畫面出來
  # -mute：容器裡沒有音效裝置，開了只是多印一行警告。
  "$BIN" -data "$DATA" -mute "$@" >/tmp/game.log 2>&1 &
  GAME=$!
  for i in $(seq 1 120); do
    xdotool search --name "城市" >/dev/null 2>&1 && { sleep 3; return; }
    kill -0 $GAME 2>/dev/null || { echo "遊戲啟動就死了"; cat /tmp/game.log; exit 1; }
    sleep 1
  done
  echo "等不到遊戲視窗"; cat /tmp/game.log; exit 1
}

alive() { kill -0 $GAME 2>/dev/null; }
stop_game() {
  kill $GAME 2>/dev/null || true
  wait $GAME 2>/dev/null || true
  # 等視窗真的消失。X 的視窗要等 server 收掉，比進程結束晚一點；
  # 沒等就開下一局的話，xdotool 會找到還沒消失的舊視窗。
  for i in $(seq 1 40); do
    xdotool search --name "城市" >/dev/null 2>&1 || return 0
    sleep 0.25
  done
}

start_x
rm -f "$SAVE"

echo "== 第一段：新城市，手動蓋 =="
run_game -seed "$SEED" -save "$SAVE"
shot 01-新城市
alive || fail "開場就崩了"

# 先按 F1 暫停。蓋東西這一段要能逐步判斷「這一下有沒有蓋出來」，
# 而時鐘一走、煙囪一冒，畫面自己就會變，判準就廢了。
key F1

# 門檻是量出來的，不是照面積算的：一塊 3×3 的分區佔 9216 個像素，但剛劃
# 好的空地和周圍的泥土幾乎同色，真正變色的只有外框和中間那個字母，
# 實測兩千出頭。而且**換一種城市外觀數字就會變**——基本外觀的電線又細
# 又暗，一整條四格只改一千個像素上下，古代亞洲的水路則明顯得多。
# 所以門檻取「明顯大於 0、明顯小於實測值」的數量級，不追求貼合。
# 這些門檻只有在遊戲暫停時才成立（時鐘與煙囪動畫都停著，雜訊是 0）。
build_plant() { key g; click $((FX+1)) $((FY+1)); }
build_wire()  { key w; drag $((FX+4)) $((FY+1)) $((FX+7)) $((FY+1)); }
build_drop()  { key w; click $((FX+7)) $((FY+2)); }
build_res()   { key z; click $((FX+6)) $((FY+4)); }
build_road()  { key r; drag $((FX+4)) $((FY+6)) $((FX+9)) $((FY+6)); }
build_elec()  { key w; drag $((FX+4)) $((FY+6)) $((FX+9)) $((FY+6)); }
build_com()   { key x; click $((FX+6)) $((FY+8)); }

do_until 4000 "蓋發電廠"       build_plant
do_until  150 "拉電線到住宅區" build_wire
do_until   20 "電線轉折"       build_drop
do_until  400 "劃住宅區"       build_res
do_until  300 "鋪道路"         build_road
do_until  150 "在路上拉電線"   build_elec
do_until  400 "劃商業區"       build_com
shot 02-蓋好
key F4   # 恢復最快速度

echo "== 第二段：四個視窗 =="
# 關視窗要關到真的關掉才往下走。軟體 OpenGL 畫地圖視窗會卡住幾百毫秒，
# 卡住期間 X 的按鍵會排隊，解卡後同一個 tick 一次收完；而 handleKeys 裡
# Esc 的處理排在開視窗的 switch 後面，於是「Esc 關掉上一個」和
# 「Ctrl-G 開下一個」擠在一起時，下一個開了又被同一拍關掉。
close_window() {
  for i in $(seq 1 12); do
    key Escape
    shot _tmp
    [ "$(diffpct "$OUT/02-蓋好.png" "$OUT/_tmp.png")" -lt 90000 ] && return 0
  done
  fail "視窗關不掉"
}
open_window() { # 快速鍵 截圖名 說明
  for i in $(seq 1 6); do
    key "$1"; shot "$2"
    local d; d=$(diffpct "$OUT/02-蓋好.png" "$OUT/$2.png")
    # 門檻 200000：換成原版版面之後視窗小多了（統計圖只有 304×125 原版像素
    # ＝ 342000 螢幕像素），舊的 400000 會讓統計圖永遠判成「沒開」，
    # 而迴圈每按一次就 toggle 一次，所以看起來像快速鍵壞了。
    if [ "$d" -ge 200000 ]; then pass "$3 開起來了（${d} 像素）"; close_window; return 0; fi
  done
  fail "$3 沒開"
  close_window
}

open_window "ctrl+m" 03-地圖   "地圖視窗"
open_window "ctrl+g" 04-統計圖 "統計圖視窗"
open_window "ctrl+b" 05-預算   "預算視窗"
open_window "ctrl+u" 06-評估   "評估視窗"

echo "== 第三段：查詢與捲動 =="
key q; click $((FX+1)) $((FY+1)); sleep 0.5; shot 07-查詢
key z
do_until 200000 "方向鍵捲動" xdotool key --clearmodifiers --repeat 30 --repeat-delay 30 Right
shot 08-捲動後

echo "== 第四段：存檔 =="
for i in $(seq 1 8); do key "ctrl+s"; [ -s "$SAVE" ] && break; sleep 0.5; done
shot 09-存檔
alive || fail "存檔時崩了"
stop_game

if [ -s "$SAVE" ]; then
  n=$(stat -c%s "$SAVE")
  [ "$n" = 27120 ] && pass "存檔落地，$n 位元組" || fail "存檔 $n 位元組，應為 27120"
else
  fail "存檔沒落地：$SAVE"
fi

if [ -s "$SAVE" ]; then
  SUM=$(go run ./cmd/simtool inspect "$SAVE")
  echo "      $SUM"
  get() { echo "$SUM" | tr ' ' '\n' | grep "^$1=" | cut -d= -f2; }
  [ "$(get coal)" -ge 16 ] && pass "發電廠進了存檔" || fail "存檔裡沒有發電廠"
  [ "$(get res)"  -ge 9  ] && pass "住宅區進了存檔" || fail "存檔裡沒有住宅區"
  [ "$(get com)"  -ge 9  ] && pass "商業區進了存檔" || fail "存檔裡沒有商業區"
  [ "$(get road)" -ge 6  ] && pass "道路進了存檔"   || fail "存檔裡道路只有 $(get road) 格"
  [ "$(get wire)" -ge 4  ] && pass "電線進了存檔"   || fail "存檔裡電線只有 $(get wire) 格"
  [ "$(get funds)" -lt 20000 ] && pass "扣了錢：$(get funds)" || fail "蓋了東西卻沒扣錢"
fi

echo "== 第五段：重開讀檔 =="
if [ -s "$SAVE" ]; then
  run_game -load "$SAVE"
  sleep 1; shot 10-讀檔
  alive || fail "讀檔後崩了"
  stop_game
  grep -q "讀取城市檔" /tmp/game.log && pass "讀檔路徑走通" || fail "讀檔沒有回報"
fi

echo "== 第六段：劇本（另一種風格）=="
run_game -scenario 5 -style asia
sleep 1; shot 11-劇本簡介
do_until 100000 "劇本簡介關得掉" key space
shot 12-劇本城市
alive || fail "劇本崩了"
stop_game

if grep -qi "panic\|runtime error" /tmp/game.log; then
  fail "log 裡有 panic"; tail -30 /tmp/game.log
fi

rm -f "$OUT/_before.png" "$OUT/_after.png" "$OUT/_tmp.png"
echo
[ "$FAIL" = 0 ] && echo "== 試玩通過，截圖在 $OUT ==" || { echo "== 試玩失敗 =="; exit 1; }
