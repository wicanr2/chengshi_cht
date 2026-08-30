#!/usr/bin/env bash
# 在容器裡跑 DOS 原版。不要直接執行，用 tools/dosbox.sh。
set -euo pipefail

mkdir -p /tmp/game /tmp/capture
cp -r "/orig/SIMCITY 1.10/." /tmp/game/
chmod -R u+w /tmp/game

# SIMCITY.CFG 的 Sound 欄決定用哪個發聲裝置：
#   I 內建 PC 喇叭／S Covox（LPT 上的 DAC）／T Tandy／N 無聲
# 要把 4 位元 PCM 錄乾淨就得走 Covox：PC 喇叭是 PWM，DOSBox 0.74 的
# 喇叭模型跟不上，錄到的只會是一段拖很長的方波（振幅剛好 5000，
# 那是 DOSBox pcspeaker 的固定音量常數，不是遊戲的聲音）。
if [ -n "${CFG_SOUND:-}" ]; then
  sed -i "s/^Sound: .*/Sound: $CFG_SOUND/" /tmp/game/SIMCITY.CFG
fi
if [ -n "${CFG_SCREEN:-}" ]; then
  sed -i "s/^Screen Mode: .*/Screen Mode: $CFG_SCREEN/" /tmp/game/SIMCITY.CFG
fi
# 換圖形風格會連音效檔一起換（WESTCEGA → DATA/WEST_SND.PSF）。
# 這是「同一個動作、不同長度的音效」預測實驗的開關。
if [ -n "${CFG_GFX:-}" ]; then
  sed -i "s/^Graphics Set: .*/Graphics Set: $CFG_GFX/" /tmp/game/SIMCITY.CFG
fi
# Tandy 模式會去找 C:\tdy\<圖形集>.pgf——螢幕模式決定的是**目錄**不是檔名
# （docs/re/16-dos-oracle.md §3）。這份副本沒有 tdy\，所以 Tandy 一直卡在
# 「Please wait - loading graphics」。TDY_FROM 把別的模式目錄複製過去，
# 讓遊戲載得下去；畫面會是錯的（EGA 的資料餵給 Tandy），但**發聲那條路
# 會活起來**——目的是錄 Tandy DAC 的輸出，不是看畫面。
if [ -n "${TDY_FROM:-}" ]; then
  mkdir -p /tmp/game/tdy
  cp /tmp/game/"$TDY_FROM"/* /tmp/game/tdy/ 2>/dev/null || true
  ls /tmp/game/tdy | head -3
fi

# PREP 是在**遊戲副本上**跑的一段 shell，在 DOSBox 起來之前執行。
# 用途是做「把某個檔案弄壞／搬走，看遊戲抱不抱怨」這種排除實驗——
# 那是判斷「同名檔案有兩份時，遊戲讀的是哪一份」唯一不必反組譯的辦法。
#
# ⚠ 只動 /tmp/game 這份副本，原始素材是唯讀掛載的（tools/dosbox.sh）。
if [ -n "${PREP:-}" ]; then
  ( cd /tmp/game && eval "$PREP" )
  echo "PREP 跑完：$PREP"
fi

head -5 /tmp/game/SIMCITY.CFG

cp /conf/dosbox.conf /tmp/dosbox.conf
[ -n "${MACHINE:-}" ] && sed -i "s/^machine=.*/machine=$MACHINE/" /tmp/dosbox.conf
[ -n "${RUN:-}" ] && echo "$RUN" >> /tmp/dosbox.conf

Xvfb :99 -screen 0 1024x768x24 -nolisten tcp >/tmp/xvfb.log 2>&1 &
export DISPLAY=:99
for i in $(seq 1 40); do xdpyinfo >/dev/null 2>&1 && break; sleep 0.25; done

# SDL 1.2 的 disk 音訊驅動：不必按 DOSBox 的錄音快速鍵，整段輸出直接落檔。
# 延遲要對上 blocksize／rate（1024 @ 22050 ≈ 46 ms），不然 callback 會用
# 最快速度被呼叫，錄出來的長度與實際時間無關（25 秒錄成 15 分鐘）。
export SDL_AUDIODRIVER=disk
export SDL_DISKAUDIOFILE="/out/$PREFIX.raw"
export SDL_DISKAUDIODELAY=46
rm -f "$SDL_DISKAUDIOFILE"

"${DOSBOX_BIN:-dosbox-x}" -conf /tmp/dosbox.conf >"/out/$PREFIX.log" 2>&1 &
D=$!
START=$(date +%s%3N)

# 每一步都記時間戳。事後要把錄音切開，靠的就是這份時間表——
# 沒有它，錄到的只是一長條聲音，分不出哪一段是哪個動作。
: > "/out/$PREFIX.marks"
mark() { local t=$(( $(date +%s%3N) - START )); printf "%6d.%03d  %s\n" $((t/1000)) $((t%1000)) "$*" >> "/out/$PREFIX.marks"; }

shot() { xwd -root -silent | convert xwd:- "/out/$PREFIX-$1.png"; }

# goto：把**遊戲裡的**游標移到畫面座標 (x,y)。
#
# ⚠ 不能只送一次絕對座標。DOS 的滑鼠驅動吃的是**相對位移**，DOSBox 把
# 主機游標的移動換算成位移送進去——一旦遊戲自己搬過游標（對話框開啟時
# 它會把游標移到預設鈕上），主機端與遊戲端就對不齊了，之後每一次點擊都
# 落在別的地方，而且畫面上完全看不出來。實測症狀：開頭幾個點擊（標題
# 畫面、劇本選擇）都對，跑一陣子之後預算對話框的按鈕怎麼點都點不到。
#
# 作法是每次先把主機游標移到視窗左上角**外面**，讓遊戲端的游標被夾在
# (0,0)，再移到目標——這樣送進去的位移剛好等於目標座標。
goto() {
  # ⚠ 用螢幕原點 (0,0)，不要用「視窗左上角減幾百」——xdotool 不吃負座標
  # （會報 `unrecognized option '-208'`）。視窗本來就不在 (0,0)，
  # 移到螢幕原點就已經在視窗左上角外面了。
  xdotool mousemove 0 0; sleep 0.1
  xdotool mousemove $((OFFX + $1)) $((OFFY + $2))
}

sleep 8

# 動作腳本裡的座標是 **DOS 畫面座標**（640×350 的左上角是 0,0）。
# DOSBox 0.74 把視窗放在螢幕左上角，兩者剛好一樣；DOSBox-X 會把畫面
# 置中，差了將近兩百個像素——照抄座標的話每一次點擊都落在畫面外，
# 而且完全沒有錯誤訊息，只是「按了沒反應」。所以執行時問一次。
OFFX=0; OFFY=0
WID=$(xdotool search --name "DOSBox" 2>/dev/null | tail -1 || true)
if [ -n "$WID" ]; then
  eval "$(xdotool getwindowgeometry --shell "$WID" 2>/dev/null | grep -E '^(X|Y|WIDTH|HEIGHT)=')"
  OFFX=${X:-0}; OFFY=${Y:-0}
  echo "視窗在 ($OFFX,$OFFY)，大小 ${WIDTH:-?}×${HEIGHT:-?}"
fi
mark "視窗原點 $OFFX,$OFFY"

mark 開機完成

# 動作腳本：一行一個動作。
#   key <鍵名>          送一個鍵
#   click <x> <y>       在畫面座標點一下
#   drag <x> <y> <x2> <y2>  按住拖曳
#   press <x> <y>       按住不放（選單是按住式的，放開才選中）
#   move <x> <y>        移動游標（按住的時候用）
#   release             放開
#   wait <秒>           等待
#   shot <名稱>         截圖
#   mark <說明>         只記時間戳
while read -r cmd a b c d; do
  case "$cmd" in
    "" | "#"*) ;;
    key)   mark "key $a";   xdotool key --clearmodifiers "$a" ;;
    # keyrep <鍵> <次數>：連按同一個鍵。捲動到地圖邊界用得上——
    # 頂到邊界之後鏡頭一定被夾在 (0,0)，就不必猜初始鏡頭是多少。
    keyrep) mark "keyrep $a x$b"
            xdotool key --clearmodifiers --repeat "$b" --repeat-delay 30 "$a" ;;
    click) mark "click $a $b"; goto $a $b; sleep 0.3
           xdotool mousedown 1; sleep 0.15; xdotool mouseup 1 ;;
    press) mark "press $a $b"; goto $a $b; sleep 0.3; xdotool mousedown 1 ;;
    # rclick：右鍵。原版的查詢是按住右鍵看格子資料（要對拍那個彈出視窗）。
    rclick) mark "rclick $a $b"; goto $a $b; sleep 0.3
            xdotool mousedown 3; sleep 0.6 ;;
    rrelease) mark rrelease; xdotool mouseup 3 ;;
    # keydown／keyup：按住一個鍵不放。原版的質詢是「按住 Q ＋ 左鍵」
    # （參考附表），一次 key 送不出這種組合。
    keydown) mark "keydown $a"; xdotool keydown "$a" ;;
    keyup) mark "keyup $a"; xdotool keyup "$a" ;;
    # ⚠ move **不重新歸位**：它是給按住式選單用的（按住標題 → 移到項目 →
    # 放開）。歸位會把游標拖出選單再拉回來，選單當場就關了。
    move)  mark "move $a $b";  xdotool mousemove $((a+OFFX)) $((b+OFFY)) ;;
    release) mark "release";   xdotool mouseup 1 ;;
    drag)  mark "drag $a $b -> $c $d"
           goto $a $b; sleep 0.3; xdotool mousedown 1
           xdotool mousemove $((c+OFFX)) $((d+OFFY)); sleep 0.3; xdotool mouseup 1 ;;
    wait)  sleep "$a" ;;
    shot)  shot "$a"; mark "shot $a" ;;
    # snap：把遊戲**當下寫出來的城市檔**抓一份走。
    # 需要它是因為 DOS 版的存檔對話框每次都預設同一個檔名，第二次存檔
    # 會蓋掉第一次；要取兩個時間點的樣本就得在中間搬走。
    snap)  mark "snap $a"
           mkdir -p /out/save
           for f in /tmp/game/*.[cC][tT][yY]; do
             [ -e "$f" ] && cp "$f" "/out/save/$a.cty" && echo "snap $a <= $f"
           done ;;
    mark)  mark "$a $b $c $d" ;;
    *)     echo "不認得的動作：$cmd" >&2 ;;
  esac
done < /conf/actions.txt

sleep "$SECS"
shot final

# 把遊戲寫出來的城市檔帶出去。DOS 版存的是原版 `.cty`，remake 讀得起來，
# 所以它是「原版跑到什麼狀態」唯一精確、機器可讀的取樣管道
# ——螢幕截圖看得到的東西，比不出人口、資金與評分。
mkdir -p /out/save
find /tmp/game -maxdepth 2 -iname '*.cty' -print -exec cp {} /out/save/ \; || true
ls -l /out/save 2>/dev/null || true
# ⚠ DOSBox-X 在有模態對話框時會吞掉 SIGTERM，`wait` 就永遠不回來
# （實測：一個六秒的實驗掛了八分鐘還在跑）。給它兩秒再補 SIGKILL。
kill $D 2>/dev/null || true
for _ in 1 2 3 4; do kill -0 $D 2>/dev/null || break; sleep 0.5; done
kill -9 $D 2>/dev/null || true
wait $D 2>/dev/null || true
# POST 是在 DOSBox 收工之後、還在容器裡時跑的一段 shell，工作目錄同樣是
# 遊戲副本。對稱於 PREP，用來把遊戲寫出來的東西帶回 /out
# （例如把標準輸出導向檔案之後要取回那份檔案）。
if [ -n "${POST:-}" ]; then
  ( cd /tmp/game && eval "$POST" ) || true
  echo "POST 跑完：$POST"
fi

echo "== 時間表 =="; cat "/out/$PREFIX.marks"
ls -l /out/"$PREFIX"*
