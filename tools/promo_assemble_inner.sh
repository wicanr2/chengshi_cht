#!/usr/bin/env bash
# 推廣影片的第二階段：把字卡與畫格編成段落、串起來、鋪上原版音效。
# 跑在 simcity-sc2k-audio image（那邊才有 ffmpeg）。cwd 是 workplace/promo。
set -euo pipefail

W=960; H=526; FPS=12
T=/tmp/asm; rm -rf "$T"; mkdir -p "$T"
: >"$T/concat.txt"
NSEG=0
TIME=0            # 目前累積到第幾秒，音效的時間點靠它算
CUES=()           # 每筆是「秒:音效編號:音量」

secs() { awk -v a="$1" -v b="$2" 'BEGIN{printf "%.4f", a+b}'; }

# enc <輸入參數…> — 共用的編碼設定。段落之間要能用 concat 直接串，
# 所以尺寸、影格率、像素格式、編碼器參數每一段都必須一樣。
enc() {
  local out=$1; shift
  ffmpeg -y -hide_banner -loglevel error "$@" \
    -r "$FPS" -c:v libx264 -preset medium -crf 20 -pix_fmt yuv420p -an "$out"
  printf "file '%s'\n" "$out" >>"$T/concat.txt"
  NSEG=$((NSEG + 1))
}

# cue <相對這一段開頭幾秒> <音效編號> <音量>
cue() { CUES+=("$(secs "$TIME" "$1"):$2:$3"); }

# card <名> <秒> [cue…]，cue 寫成 offset,sfx,gain
card() {
  local name=$1 dur=$2; shift 2
  local f; f=$(printf '%s/%02d.mp4' "$T" "$NSEG")
  enc "$f" -loop 1 -framerate "$FPS" -t "$dur" -i "cards/$name.png" \
    -vf "scale=$W:$H:flags=neighbor,setsar=1"
  local c; for c in "$@"; do IFS=, read -r o s g <<<"$c"; cue "$o" "$s" "$g"; done
  TIME=$(secs "$TIME" "$dur")
}

# live <段名> [cue…]
live() {
  local name=$1; shift
  local n; n=$(find "frames/$name" -name '*.png' 2>/dev/null | wc -l)
  [ "$n" -gt 0 ] || { echo "段落 $name 一格都沒有" >&2; return 1; }
  local dur; dur=$(awk -v n="$n" -v f="$FPS" 'BEGIN{printf "%.4f", n/f}')
  local f; f=$(printf '%s/%02d.mp4' "$T" "$NSEG")
  enc "$f" -framerate "$FPS" -pattern_type glob -i "frames/$name/*.png" \
    -vf "scale=$W:$H:flags=neighbor,setsar=1"
  local c; for c in "$@"; do IFS=, read -r o s g <<<"$c"; cue "$o" "$s" "$g"; done
  TIME=$(secs "$TIME" "$dur")
  printf '  %-10s %3d 格 %5.1f 秒（累計 %s）\n' "$name" "$n" "$dur" "$TIME"
}

echo "== 編段落 =="
# 音效編號：0 塞車、1 爆炸、2 怪獸、3 警笛、4 船笛、5 小工具、6 大工具、7 失敗。
card 00-open   3.2  0.5,4,1.0
live title          0.2,6,0.4
card 01-cht    2.4  0.1,5,0.5
live menu           0.2,6,0.4
card 02-swmap  2.6  0.1,5,0.5
live taiwan         0.3,4,0.7
live kaohsiung      0.3,0,0.5
card 03-newmap 2.2  0.1,5,0.5
live taipei
live taichung
live tainan         0.4,6,0.4
card 04-build  2.0  0.1,5,0.5
live build          0.3,6,0.7 1.3,6,0.7 2.3,6,0.7
card 05-disast 2.2  0.2,3,0.8
live disaster       0.3,1,0.9 1.8,1,0.9 3.3,2,0.9 5.0,3,0.8 6.5,1,0.9 8.0,2,0.9
card 06-data   2.2  0.1,5,0.5
live windows        0.2,6,0.4
live layers         0.2,5,0.4
card 07-scen   2.0  0.1,5,0.5
live scen           0.3,3,0.6
card 08-modes  2.4  0.1,5,0.5
card 09-grid   3.4  0.1,6,0.4
card 10-end    4.2  0.4,4,1.0

echo "共 $NSEG 段，$TIME 秒"

echo "== 串接 =="
ffmpeg -y -hide_banner -loglevel error -f concat -safe 0 -i "$T/concat.txt" -c copy "$T/video.mp4"

echo "== 音軌（原版八段音效）=="
INPUTS=(); FILTERS=(); n=0
for c in "${CUES[@]}"; do
  t=${c%%:*}; rest=${c#*:}; s=${rest%%:*}; g=${rest#*:}
  ms=$(awk -v t="$t" 'BEGIN{printf "%d", t*1000}')
  INPUTS+=(-i "sfx/$s.wav")
  # ⚠ **音效是第 1 個輸入起跳**，不是第 0 個——第 0 個是上面那支只有畫面的
  # `video.mp4`。從 0 起算的話 ffmpeg 會回「`:a` matches no streams」。
  FILTERS+=("[$((n + 1)):a]aformat=channel_layouts=stereo,adelay=${ms}|${ms},volume=${g}[a$n]")
  n=$((n + 1))
done
MIXIN=""
for ((i = 0; i < n; i++)); do MIXIN+="[a$i]"; done
GRAPH="$(IFS=';'; echo "${FILTERS[*]}");${MIXIN}amix=inputs=$n:normalize=0,alimiter=limit=0.95,aresample=44100,apad[aout]"
# ⚠ 音軌要 `apad` 補到比畫面長，再用 `-shortest` 裁齊。少了 apad 的話，
# 最後一次音效放完音軌就結束，`-shortest` 會照那個長度把**畫面**砍掉。
ffmpeg -y -hide_banner -loglevel error -i "$T/video.mp4" "${INPUTS[@]}" \
  -filter_complex "$GRAPH" -map 0:v -map '[aout]' \
  -c:v copy -c:a aac -b:a 160k -shortest promo.mp4
echo "  $n 次音效落在時間軸上"

echo "== GIF（給 README；GitHub 不播 mp4，GIF 也沒有聲音）=="
# 8 fps／560 寬是為了讓 README 那張控制在 1.5 MB 上下。調色盤走兩趟，
# 不然十六色的畫面會被抖動糊成一片。
ffmpeg -y -hide_banner -loglevel error -i "$T/video.mp4" \
  -vf "fps=8,scale=560:-2:flags=neighbor,palettegen=stats_mode=diff" "$T/pal.png"
ffmpeg -y -hide_banner -loglevel error -i "$T/video.mp4" -i "$T/pal.png" \
  -lavfi "fps=8,scale=560:-2:flags=neighbor[v];[v][1:v]paletteuse=dither=none" promo.gif
