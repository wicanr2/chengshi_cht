#!/usr/bin/env bash
# tools/promo_video.sh 的第一階段：在 Xvfb 裡跑遊戲，逐段擷取畫格。
#
# 每個段落自己一個目錄（`frames/<段名>/NNNN.png`），第二階段照段落編碼再串。
# 字卡、六格對照與配樂都在第二階段，這裡只出遊戲畫面。
#
# 兩個踩過的坑，寫在這裡免得再犯：
#   一、**組合鍵不能用 `xdotool key ctrl+c` 這種寫法**。按下與放開幾乎在
#       同一瞬間，遊戲是逐畫格輪詢按鍵的，抓不抓得到看運氣。要分段送。
#   二、**訊息框會吃掉所有按鍵**（`game.go:635` 的 `g.picture == ""`），
#       城市跑起來就會跳。每格之前先送一次空白鍵。
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

SEG=""; N=0; TOTAL=0; GAME_PGID=; WID=

start_seg() { SEG=$1; N=0; mkdir -p "$OUT/$SEG"; }

grab() {
  N=$((N + 1)); TOTAL=$((TOTAL + 1))
  xwd -root -silent | convert xwd:- "$(printf '%s/%s/%04d.png' "$OUT" "$SEG" "$N")" 2>/dev/null
}

# grabs <幾格>：連續擷取，每格之前清一次訊息框。
grabs() { local i; for i in $(seq 1 "$1"); do xdotool key --clearmodifiers space; grab; done; }

# combo <修飾鍵> <鍵>：把修飾鍵按住跨畫格再送，見檔頭第一個坑。
combo() {
  xdotool keydown "$1"; sleep 0.2
  xdotool key "$2";     sleep 0.2
  xdotool keyup "$1";   sleep 0.3
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
  GAME_PGID=; WID=
  sleep 0.4
}

# launch <遊戲參數…>：開起來、等視窗、把滑鼠移開。
launch() {
  setsid /tmp/chengshi -data "$DATA" -mute "$@" >/tmp/game.log 2>&1 &
  GAME_PGID=$!
  local ok=0
  for _ in $(seq 1 900); do
    if WID=$(xdotool search --name '^城市' 2>/dev/null | tail -n 1) && [ -n "$WID" ]; then ok=1; break; fi
    sleep 0.05
  done
  if [ "$ok" = 0 ]; then
    echo "場景沒開起來：$*" >&2; tail -3 /tmp/game.log >&2; stop_game; return 1
  fi
  xdotool windowmove "$WID" 0 0
  sleep 2.2
  xdotool mousemove 1919 1049
  sleep 0.3
}

done_seg() { stop_game; printf '  %-10s %3d 格（累計 %d）\n' "$SEG" "$N" "$TOTAL"; }

echo "== 擷取 =="

# 一、原版招牌。**不能帶任何參數**：`-load`／`-scenario`／`-demo`／`-window`／
#     `-cam`／`-seed` 任一有值就直接進城市，拍到的會是隨機地形。
start_seg title; launch; grabs 12; done_seg

# 二、中文介面：把災難選單拉開給人看。
start_seg menu; launch -load cities/TAIWAN.CTY -cam 40,20
xdotool key --clearmodifiers 0; sleep 0.4
combo alt d
grabs 16; done_seg

# 三、地形巡覽（暫停，按住方向鍵連續運鏡）。
pan() {
  start_seg "$1"; launch -load "cities/$2" -cam "$3"
  xdotool key --clearmodifiers 0; sleep 0.5
  xdotool keydown Right
  local i; for i in $(seq 1 "$4"); do grab; done
  xdotool keyup Right
  done_seg
}
pan taiwan    TAIWAN.CTY    0,30 24
pan kaohsiung KAOHSIUN.CTY  0,28 14
pan taipei    TAIPEI.CTY    0,28 14
pan taichung  TAICHUNG.CTY  0,28 14
pan tainan    TAINAN.CTY    0,28 14

# 三之二、地形編輯器：三個旋鈕調一調，按開始長出一張新地圖。
#
# ⚠ 這一段**不能用 `grabs`**：它每格會送一次空白鍵，而空白鍵在對話框裡是
# 「按下目前選取的控制項」——會把參數改掉，甚至直接按到開始。
# 座標是原版像素 ×3（UIScale）：對話框原點 (172,95)，`►` 在第 10／21 欄、
# 第 5 列，`開始` 在第 3 欄、第 8 列。
still() { local i; for i in $(seq 1 "${1:-1}"); do grab; done; }
poke() { xdotool mousemove "$1" "$2" click 1; sleep 0.12; grab; }
start_seg terrain; launch -window terrain -seed 7
still 5
for _ in 1 2 3 4 5 6 7 8; do poke 777 525; done   # 樹木 ►
for _ in 1 2 3 4 5;       do poke 1041 525; done  # 湖泊 ►
for _ in 1 2 3 4;         do poke 1137 525; done  # 彎曲 ◄
still 4
xdotool mousemove 693 651 click 1; sleep 0.6      # 開始
still 4
xdotool mousemove 936 735 click 1; sleep 1.4      # 建造新城市 → 確定
still 3
xdotool mousemove 1919 1049; sleep 0.3
combo ctrl c
still 10
done_seg

# 四、實機遊玩：台灣地圖上蓋一座城市、快轉 25 年，關掉 City Form 讓城市看得見。
start_seg build; launch -load cities/TAIWAN.CTY -demo 25
xdotool key --clearmodifiers space; sleep 0.3
xdotool key --clearmodifiers 2;     sleep 0.4
grabs 8
combo ctrl c
grabs 22; done_seg

# 五、六種災難。災難選單是 Alt-D，往下第 n 列再 Enter；
#     順序照 `internal/ui/windows.go` 的 disasterItems：
#     0 火災、1 洪水、2 空難、3 龍捲風、4 地震、5 怪獸。
start_seg disaster; launch -load cities/TAIWAN.CTY -demo 25
xdotool key --clearmodifiers space; sleep 0.3
xdotool key --clearmodifiers 2;     sleep 0.4
combo ctrl c
for row in 0 1 2 3 4 5; do
  combo alt d
  for _ in $(seq 1 "$row"); do xdotool key Down; sleep 0.08; done
  xdotool key Return; sleep 0.5
  grabs 10
done
done_seg

# 六、三個資料視窗：預算、統計圖、評估。
start_seg windows; launch -load cities/TAIWAN.CTY -demo 25
xdotool key --clearmodifiers space; sleep 0.3
xdotool key --clearmodifiers 0;     sleep 0.4
for k in b g u; do combo ctrl "$k"; grabs 8; combo ctrl c; done
done_seg

# 七、地圖圖層：City Form 開著，點過幾個圖層圖示。
start_seg layers; launch -load cities/TAIWAN.CTY -demo 25
xdotool key --clearmodifiers space; sleep 0.3
xdotool key --clearmodifiers 0;     sleep 0.4
for y in 168 248 328 472 552; do
  xdotool mousemove --window "$WID" 775 "$y" click 1; sleep 0.35
  xdotool mousemove 1919 1049
  grabs 4
done
done_seg

# 八、劇本選單（原版的第二幅 .PPF）與一個劇本。
#     座標與 tools/appimage_player_path_inner.sh 用的同一組。
start_seg scen; launch
xdotool mousemove --window "$WID" 948 714 click 1; sleep 0.8
xdotool mousemove 1919 1049; grabs 10
xdotool mousemove --window "$WID" 243 360 click 1; sleep 0.8
xdotool mousemove 1919 1049; grabs 6
xdotool key --clearmodifiers space; sleep 0.5
grabs 8; done_seg

echo "共 $TOTAL 格"
