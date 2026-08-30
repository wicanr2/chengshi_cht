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
# 原點是 **(64,55)** 的原版座標，畫布放大三倍（internal/ui/classic.go）。
#
# ⚠⚠ 這兩個數字**必須跟 `internal/ui/classic.go` 的 `editViewX`／`editViewY`
# 一致**，改一邊就要改另一邊。2026-08-30 把 `editViewY` 從 54 訂正成 55
# （54 那一列是地圖區的白色外框，圖塊從 55 開始），這裡忘了跟著改，
# 結果每一次點擊都往下偏一像素——**只有落在格線附近的那幾下會跨格**，
# 所以症狀是「六格的道路只蓋出三格」，不是「點不到」。
# 底下的 `check_view_origin` 會在開跑前用截圖驗這兩個數字。
# 少加這個位移的話，每一次點擊都落在離目標好幾格的地方，而遊戲照樣
# 蓋得出東西——症狀是「試玩腳本蓋的城市長得不對」，不是「點不到」。
# 鏡頭用 `-cam` 直接擺到空地的左上角，不靠置中也不靠捲動——
# 捲動的步數會被地圖邊界夾住而算錯，而且錯了照樣點得下去，只是點在別格。
UIS=3; VIEWX=64; VIEWY=55
CAMX=$FX; CAMY=$FY; PX=$((16 * UIS))
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
run_game -seed "$SEED" -save "$SAVE" -cam "$FX,$FY"
shot 01-新城市
alive || fail "開場就崩了"

# 先按 F1 暫停。蓋東西這一段要能逐步判斷「這一下有沒有蓋出來」，
# 而時鐘一走、煙囪一冒，畫面自己就會變，判準就廢了。
key 0

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
key 4   # 恢復最快速度

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
  local last=0
  for i in $(seq 1 6); do
    key "$1"; shot "$2"
    local d; d=$(diffpct "$OUT/02-蓋好.png" "$OUT/$2.png")
    # 門檻 200000：換成原版版面之後視窗小多了（統計圖只有 304×125 原版像素
    # ＝ 342000 螢幕像素），舊的 400000 會讓統計圖永遠判成「沒開」，
    # 而迴圈每按一次就 toggle 一次，所以看起來像快速鍵壞了。
    if [ "$d" -ge 300000 ]; then pass "$3 開起來了（${d} 像素）"; close_window; return 0; fi
    last=$d
  done
  # 失敗時要印出量到的數字。只說「沒開」的話分不出「快速鍵沒作用」與
  # 「視窗開了但門檻訂太高」——後者在版面改小之後發生過。
  fail "$3 沒開（最後一次量到 ${last} 像素）"
  close_window
}

open_window "ctrl+m" 03-地圖   "地圖視窗"
open_window "ctrl+g" 04-統計圖 "統計圖視窗"
open_window "ctrl+b" 05-預算   "預算視窗"
open_window "ctrl+u" 06-評估   "評估視窗"

echo "== 第三段：查詢與捲動 =="
key q; click $((FX+1)) $((FY+1)); sleep 0.5; shot 07-查詢
key z
# 門檻 80000：編輯視窗的地圖區只有 176×257 原版像素（＝ 407000 螢幕像素），
# 而地形大片同色，捲動之後真正變色的沒有想像中多。舊的 200000 是照
# 舊版面（1024×768 的地圖區）訂的。
do_until 80000 "方向鍵捲動" xdotool key --clearmodifiers --repeat 30 --repeat-delay 30 Right
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
  run_game -load "$SAVE" -cam "$FX,$FY"
  sleep 1; shot 10-讀檔
  alive || fail "讀檔後崩了"
  stop_game
  grep -q "讀取城市檔" /tmp/game.log && pass "讀檔路徑走通" || fail "讀檔沒有回報"
fi

echo "== 第六段：建造新城市對話框 =="
# 原版的「建造新城市」要先選市名與技術等級才進得了遊戲（說明書 p.11）。
# 判準不是畫面，是**存檔裡的資金**：艱難的起始資金是 $5,000
# （Micropolis w_util.c:177），簡易是 $20,000——選錯等級一眼就看得出來。
NCSAVE=/tmp/pt/hard.cty
rm -f "$NCSAVE"
run_game -window newcity -save "$NCSAVE"
sleep 1; shot 13-新城市對話框
# 「艱難」那一列與「確定」鈕的原版座標見 internal/ui/newcity.go。
xdotool mousemove $((284 * 3)) $((216 * 3)); sleep 0.05
xdotool mousedown 1; sleep 0.15; xdotool mouseup 1; sleep 0.3
shot 14-選艱難
xdotool mousemove $((312 * 3)) $((245 * 3)); sleep 0.05
xdotool mousedown 1; sleep 0.15; xdotool mouseup 1; sleep 0.5
shot 15-新城市
for i in $(seq 1 8); do key "ctrl+s"; [ -s "$NCSAVE" ] && break; sleep 0.5; done
alive || fail "新城市對話框之後崩了"
stop_game
if [ -s "$NCSAVE" ]; then
  NCSUM=$(go run ./cmd/simtool inspect "$NCSAVE")
  echo "      $NCSUM"
  ncfunds=$(echo "$NCSUM" | tr ' ' '\n' | grep '^funds=' | cut -d= -f2)
  [ "$ncfunds" = 5000 ] && pass "艱難的起始資金 \$5,000" \
    || fail "艱難的起始資金是 $ncfunds，應為 5000"
else
  fail "新城市沒存出檔"
fi

echo "== 第七段：劇本（另一種風格）=="
run_game -scenario 5 -style asia
sleep 1; shot 11-劇本簡介
do_until 100000 "劇本簡介關得掉" key space
shot 12-劇本城市
alive || fail "劇本崩了"
stop_game

echo "== 第八段：與原版對拍 =="
# 判準不是「看起來像」，是**逐位元等於原版**。兩層：
#
#   ① 美術對拍（一定跑）：地圖區每一格拿去比對 `.PGF` 第 0 庫的 960 張圖塊。
#      只需要玩家自備的原版資料檔，不需要 DOSBox。
#   ② 畫面對拍（有基準才跑）：直接跟**跑起來的原版**逐格比。
#      基準要先跑 `tools/screen_parity.sh` 產生（那一支要 DOSBox），
#      產物在 workplace/screen-parity/dos.png，不進版控（CLAUDE.md §8）。
#
# 為什麼要有這一段：2026-08-30 一天抓到四個 remake 的錯，四個都**沒有症狀**
# （編得過、測得過、玩得動、目視也看不出來）：
#   ① EGA 調色盤照抄檔案原值 0/80/160/240（螢幕上是 0/85/170/255）；
#   ② 工具盤畫在 (6,53) 而不是 (8,55)；
#   ③ 地圖圖塊把色號 0 當透明（那是真正的黑：道路標線、建築輪廓）；
#   ④ **城市檔的檔頭讀成 144／地圖起點讀成 3264**，正確是 128／3248——
#      整張地圖往下平移 8 列。前三個美術對拍就擋得下來，第四個只有
#      畫面對拍看得出來（存檔對拍兩邊同樣偏移，互相抵銷）。
#
# 門檻 150：露出來的是 176 格（City Form 視窗蓋住 x≥241），
# 扣掉底列被工具帶蓋到的 11 格與工具游標框的 2 格，上限是 163。
run_game -scenario 1 -style west -cam 0,0
key space          # 關掉劇本簡介
key 0              # 暫停，畫面才是靜態的
sleep 1
shot 13-對拍
stop_game

ATLAS=/tmp/tiles-west.png
if go run ./tools/tileatlas "$DATA/CEGA/WESTCEGA.PGF" "$ATLAS" >/dev/null 2>&1 ||
   go run ./tools/tileatlas "$DATA/cega/westcega.pgf" "$ATLAS" >/dev/null 2>&1; then
  if python3 tools/shot_tilescan.py --atlas "$ATLAS" --scale 3 --origin 64,55 \
       --max-x 241 --max-y 311 --min-hit 150 "$OUT/13-對拍.png"; then
    pass "地圖區逐格等於原版美術"
  else
    fail "地圖區與原版美術對不上——顏色、位置或透明處理有一個錯了"
  fi
else
  fail "產不出圖塊圖集（找不到 WESTCEGA.PGF？）"
fi

DOSREF=workplace/screen-parity/dos.png
if [ -f "$DOSREF" ]; then
  if python3 tools/shot_diff_cells.py "$DOSREF" "$OUT/13-對拍.png" --min-hit 150; then
    pass "畫面逐格等於跑起來的原版"
  else
    fail "畫面與原版對不上"
  fi
else
  echo "      （沒有原版基準，跳過畫面對拍——跑 tools/screen_parity.sh 產生）"
fi

if grep -qi "panic\|runtime error" /tmp/game.log; then
  fail "log 裡有 panic"; tail -30 /tmp/game.log
fi

rm -f "$OUT/_before.png" "$OUT/_after.png" "$OUT/_tmp.png"
echo
[ "$FAIL" = 0 ] && echo "== 試玩通過，截圖在 $OUT ==" || { echo "== 試玩失敗 =="; exit 1; }
