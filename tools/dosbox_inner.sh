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
    click) mark "click $a $b"
           xdotool mousemove $((a+OFFX)) $((b+OFFY)); sleep 0.3
           xdotool mousedown 1; sleep 0.15; xdotool mouseup 1 ;;
    press) mark "press $a $b"
           xdotool mousemove $((a+OFFX)) $((b+OFFY)); sleep 0.3; xdotool mousedown 1 ;;
    move)  mark "move $a $b";  xdotool mousemove $((a+OFFX)) $((b+OFFY)) ;;
    release) mark "release";   xdotool mouseup 1 ;;
    drag)  mark "drag $a $b -> $c $d"
           xdotool mousemove $((a+OFFX)) $((b+OFFY)); sleep 0.3; xdotool mousedown 1
           xdotool mousemove $((c+OFFX)) $((d+OFFY)); sleep 0.3; xdotool mouseup 1 ;;
    wait)  sleep "$a" ;;
    shot)  shot "$a"; mark "shot $a" ;;
    mark)  mark "$a $b $c $d" ;;
    *)     echo "不認得的動作：$cmd" >&2 ;;
  esac
done < /conf/actions.txt

sleep "$SECS"
shot final
kill $D 2>/dev/null || true; wait $D 2>/dev/null || true
echo "== 時間表 =="; cat "/out/$PREFIX.marks"
ls -l /out/"$PREFIX"*
