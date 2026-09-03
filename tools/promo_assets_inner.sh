#!/usr/bin/env bash
# 推廣影片的非畫面素材：原版音效、六種顯示模式的對照格、字卡。
# 全部在 simcity-go image 裡做（要 Go 與 ImageMagick），第二階段只負責編碼。
set -euo pipefail
cd /src
P=workplace/promo
mkdir -p "$P/sfx" "$P/modes" "$P/cards"

go build -o /tmp/simtool ./cmd/simtool
go build -o /tmp/promocard ./tools/promocard

# 一、原版八段音效。**配樂只能用原版真實素材**（~/.claude/rulebook/93 鐵則 1），
#    而這款遊戲根本沒有背景音樂（docs/re/19-no-music.md 四條證據），
#    所以聲音層就是這八段：0 塞車、1 爆炸、2 怪獸、3 警笛、4 船笛、
#    5／6 工具成功、7 工具失敗（internal/sim/sound.go）。
echo "== 音效 =="
/tmp/simtool sound -file "workplace/dos110/SIMCITY 1.10/SOUNDDAT.PSF" -out "$P/sfx" | tail -2

# 二、六種顯示模式的招牌畫面。CGA、Tandy 與 mono 只有 1.03 有，
#    mcga 只有 1.10 有——兩份都是原廠 DOS 版本，不是同一片磁片。
echo "== 顯示模式 =="
A=1.0/original/1.03
B="workplace/dos110/SIMCITY 1.10"
/tmp/simtool ppf -file $A/CGANTRO.PPF  -mode cga  -out "$P/modes/1-cga.png"  >/dev/null
/tmp/simtool ppf -file $A/SEGANTRO.PPF -mode sega -out "$P/modes/6-sega.png" >/dev/null
/tmp/simtool ppf -file $A/CEGANTRO.PPF -mode cega -out "$P/modes/3-cega.png" >/dev/null
/tmp/simtool ppf -file $A/TDYNTRO.PPF  -mode tdy  -out "$P/modes/2-tdy.png"  >/dev/null
/tmp/simtool ppf -file "$B/mcga/mcgantro.ppf" -mode mcga -pal "$B/mcga/westmcga.pgf" \
  -out "$P/modes/4-mcga.png" >/dev/null
/tmp/simtool ppf -file $A/MONONTRO.PPF -mode mono -out "$P/modes/5-mono.png" >/dev/null
# 疊成 3×2。**不要用 montage**：它預設會畫檔名標籤，需要字型，
# 而這個 image 沒有可寫的 fontconfig 快取，會直接 core dump。
for n in 1-cga 6-sega 3-cega 2-tdy 4-mcga 5-mono; do
  convert "$P/modes/$n.png" -filter point -resize 260x164 \
    -background black -gravity center -extent 280x180 "/tmp/$n-c.png"
done
convert /tmp/1-cga-c.png /tmp/6-sega-c.png /tmp/3-cega-c.png +append /tmp/row1.png
convert /tmp/2-tdy-c.png /tmp/4-mcga-c.png /tmp/5-mono-c.png +append /tmp/row2.png
convert /tmp/row1.png /tmp/row2.png -append "$P/modes/grid.png"
identify "$P/modes/grid.png"

# 三、字卡。用遊戲自己那套點陣字，字卡與實機畫面的字才會是同一套。
echo "== 字卡 =="
card() { /tmp/promocard -out "$P/cards/$1.png" "${@:2}" >/dev/null; }
card 00-open  -big "城　市" \
  -line "模擬城市　1989　繁體中文重製版" \
  -line "Micropolis 原始碼當規格書，用 Go 重寫"
card 01-cht   -line "這款遊戲當年沒有中文版" -line "台灣代理的是英文遊戲 ＋ 中文說明書"
card 02-swmap -line "1990 年，軟體世界在高雄" -line "畫了台灣與高雄兩張地圖"
card 03-newmap -line "台北、台中、台南" -line "這一版補上"
card 03b-terr -line "1990 年隨磁片附的地形編輯器" -line "三個旋鈕，一張新地圖"
card 04-build -line "從一片空地開始"
card 05-disast -big "天災" -line "火災、洪水、空難、龍捲風、地震、怪獸"
card 06-data  -line "預算、統計、評估" -line "每一個欄位都讀得懂"
card 07-scen  -line "八個悲情城市"
card 08-modes -line "CGA、Tandy、EGA、VGA、Mono" -line "六種顯示模式全部解得開"
card 09-grid  -image "$P/modes/grid.png" -line "同一幅招牌，六種顯示卡"
card 10-end   -big "城　市" \
  -line "github.com/wicanr2/chengshi_cht" \
  -line "RRSAL-1.0 授權　非商業免費" \
  -line "遊戲資料請自備合法原版"
ls "$P/cards" | tr '\n' ' '; echo
