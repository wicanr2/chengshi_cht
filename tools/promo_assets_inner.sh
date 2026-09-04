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
# ⚠ 格子的尺寸被**字卡的高度**綁住：09-grid 那張卡是「對照圖 ＋ 三語說明」，
# 三行說明佔 162 像素，扣掉上下兩條橫線之後圖只剩 248 可用。兩列 ＝ 一格
# 最多 124 高。放大回去的話 `promocard` 會直接失敗（它會擋，不會靜靜切掉）。
for n in 1-cga 6-sega 3-cega 2-tdy 4-mcga 5-mono; do
  convert "$P/modes/$n.png" -filter point -resize 186x114 \
    -background black -gravity center -extent 200x124 "/tmp/$n-c.png"
done
convert /tmp/1-cga-c.png /tmp/6-sega-c.png /tmp/3-cega-c.png +append /tmp/row1.png
convert /tmp/2-tdy-c.png /tmp/4-mcga-c.png /tmp/5-mono-c.png +append /tmp/row2.png
convert /tmp/row1.png /tmp/row2.png -append "$P/modes/grid.png"
identify "$P/modes/grid.png"

# 三、字卡。用遊戲自己那套點陣字，字卡與實機畫面的字才會是同一套。
#
# 文字來自 `translations/promo_cards.tsv`（繁中／日文／英文並排）。放在
# `translations/` 是有原因的：`tools/build_font.py` 只掃
# `translations`／`internal/i18n`／`internal/ui`／`docs/manual-cht`，
# 字卡的日文字若寫在這支 `.sh` 裡，圖集不會烘那些字，畫出來會是空格。
#
# 一張卡三行：繁中白、日文青、英文綠。`promocard` 放不下會直接失敗，
# 不會靜靜地把字切掉。
echo "== 字卡 =="
CARDS=translations/promo_cards.tsv
tri() { # 卡名 → 依序印出該卡的三語行，每行前面帶顏色
  awk -F'\t' -v c="$1" '
    $1 ~ "^"c"\\.[0-9]+$" {
      printf "#FFFFFF|%s\n#55FFFF|%s\n#55FF55|%s\n", $2, $3, $4
    }' "$CARDS"
}
card() { # 卡名 [額外參數…]
  local name=$1; shift
  local args=()
  local ln
  while IFS= read -r ln; do args+=(-line "$ln"); done < <(tri "$name")
  [ "${#args[@]}" -gt 0 ] || { echo "字卡 $name 在 $CARDS 裡沒有文字" >&2; exit 1; }
  /tmp/promocard -out "$P/cards/$name.png" "$@" "${args[@]}" >/dev/null
}
card 00-open   -big "城　市"
card 01-cht
card 01b-lang
card 02-swmap
card 03-newmap
card 03b-terr
card 04-build
card 05-disast -big "天　災"
card 06-data
card 07-scen
card 08-modes
card 09-grid   -image "$P/modes/grid.png"
card 10-end    -big "城　市" -line "#FFFF55|github.com/wicanr2/chengshi_cht"
ls "$P/cards" | tr '\n' ' '; echo
