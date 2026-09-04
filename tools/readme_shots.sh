#!/usr/bin/env bash
# 重產 README 的畫面截圖。
#
# 為什麼要有腳本：這些圖是「現在的 remake 長什麼樣」的證據，版面一改就過期。
# 手動截圖沒有紀錄用了哪些參數，下次改完版面沒有人知道怎麼截出同一張。
#
# ⚠ 圖裡的地圖圖塊來自**玩家自己那份原版**（CLAUDE.md §8）：
# 本專案只放 remake 跑起來的畫面，不放原版遊戲本身的截圖。
#
# 用法：tools/readme_shots.sh
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
OUT=docs/images
mkdir -p "$OUT"

shot() { # 檔名 GAME_ARGS GAME_KEYS 等待秒數
  local name="$1" args="$2" keys="${3:-}" wait="${4:-8}"
  GAME_ARGS="$args" GAME_KEYS="$keys" ./tools/screenshot.sh "$wait" "_rm.png" >/dev/null 2>&1
  cp workplace/shots/_rm.png "$OUT/$name"
  echo "  $name  ($args)"
}

echo "重產 README 截圖："
# 底特律 1972，基本外觀。space 關掉劇本簡介、0 暫停讓畫面穩定。
shot city.png       "-scenario 6 -style base -cam 30,30" "space 0"
# 同一座城市換中世紀資料片。
shot style-medi.png "-scenario 6 -style medi -cam 30,30" "space 0"
# 地圖視窗的犯罪率圖層（圖層 6，順序照訊息檔第 10 段）。
# ⚠ 這一張**不能按 `0` 暫停**：地圖視窗開著的時候 `1`–`9`／`0` 是切圖層，
# 按下去會把 `-layer 6` 蓋掉（先前那一版拍到的其實是「消防範圍」）。
shot maps.png       "-scenario 6 -style base -window maps -layer 6" "space"
# 劇本簡介（中世紀的改寫）。不要按 space，簡介才留在畫面上。
shot brief.png      "-scenario 6 -style medi" "" 6
# 評估視窗。
shot eval.png       "-scenario 6 -style base -window eval" "space 0"
# 縮小到 1/4（remake 加的功能，原版沒有）。按兩次 `-`。
shot zoom.png       "-scenario 6 -style base -cam 0,0" "space 0 minus minus"

# im 是 ImageMagick 的 docker 包裝（主機不裝東西，CLAUDE.md §6）。
# 工作目錄固定是 workplace/shots，所以檔名一律相對它。
im() { # 子命令 參數…
  local sub="$1"; shift
  docker run --rm --log-opt max-size=10m --log-opt max-file=3 \
    -u "$(id -u):$(id -g)" -v "$ROOT/workplace/shots:/w" -w /w \
    --entrypoint "$sub" dpokidov/imagemagick "$@" 2>/dev/null
}

# 六個資料片風格的對照圖，兩片資料片各一張。
#
# ⚠ 三張要**同一座城市、同一個鏡頭**，不然比的就不是美術而是地形。
# 所以六張都用 `-scenario 6 -cam 30,30`，`space` 關掉劇本簡介、`0` 暫停
# ——不暫停的話模擬會繼續跑，隨機災難會彈訊息框蓋住城市，而且**災難會把
# 鏡頭捲到事發地點**，六張就對不齊了（實測踩過：其中一張變成一片空地）。
#
# 縮放用 `-filter point -resize 33.334%`，把 UIScale 3 的畫面還原成
# **原版 1:1 的像素**。非整數倍縮放會把點陣美術糊掉，而這幾張的重點
# 正是美術本身。
echo "  六個資料片風格："
for sty in asia medi west fusa feur moon; do
  GAME_ARGS="-scenario 6 -style $sty -cam 30,30" GAME_KEYS="space 0" \
    ./tools/screenshot.sh 8 "_sty-$sty.png" >/dev/null 2>&1
  im convert "_sty-$sty.png" -crop 726x900+0+60 +repage \
    -filter point -resize 33.334% "_e-$sty.png"
  echo "    $sty"
done
im montage -background '#202020' -tile 3x1 -geometry +4+4 \
  _e-asia.png _e-medi.png _e-west.png +dither -colors 32 \
  -define png:compression-level=9 styles-ancient.png
im montage -background '#202020' -tile 3x1 -geometry +4+4 \
  _e-fusa.png _e-feur.png _e-moon.png +dither -colors 32 \
  -define png:compression-level=9 styles-future.png
cp workplace/shots/styles-ancient.png workplace/shots/styles-future.png "$OUT/"
echo "  styles-ancient.png / styles-future.png"

# 地形編輯器的參數對話框（原版 `TERRAIN.EXE` 的那一個）。
# ⚠ 這裡不能用 shot()：它會直接把 1920×1050 的原圖複製進 docs/images，
# 而這一張要縮回原版 1:1。所以自己叫 screenshot.sh，再在 workplace/shots
# 裡縮（`im()` 掛的是那個目錄，吃不到 docs/images 的路徑）。
GAME_ARGS="-window terrainparams" ./tools/screenshot.sh 9 "_tp0.png" >/dev/null 2>&1
im convert _tp0.png -filter point -resize 33.334% \
  +dither -colors 32 -define png:compression-level=9 terrain-params.png
cp workplace/shots/terrain-params.png "$OUT/"
echo "  terrain-params.png"

# 六種顯示模式與四種模式的左欄面板。
#
# ⚠ 需要**六種顯示模式的圖形檔都齊全**的資料目錄：1.10 缺 CGA 與 Tandy，
# 1.03 缺 mcga 與資料片，兩片資料片沒有基本外觀。沒有那個目錄就跳過，
# 不要拿不齊的資料產一張少兩格的圖。
MODES_DATA="${MODES_DATA:-workplace/allmodes}"
if [ -d "$ROOT/$MODES_DATA/CGA" ] && [ -d "$ROOT/$MODES_DATA/tdy" ]; then
  echo "  六種顯示模式："
  for m in cega sega tdy mcga mono cga; do
    GAME_ARGS="-data $MODES_DATA -mode $m -style asia -load cities/TAIPEI.CTY" \
      ./tools/screenshot.sh 9 "_mode-$m.png" >/dev/null 2>&1
    im convert "_mode-$m.png" -filter point -resize 33.334% "_m-$m.png"
    im convert "_mode-$m.png" -crop 190x820+18+150 +repage \
      -filter point -resize 60% "_p-$m.png"
    echo "    $m"
  done
  im montage -background '#404040' -tile 3x2 -geometry +2+2 \
    _m-cega.png _m-sega.png _m-tdy.png _m-mcga.png _m-mono.png _m-cga.png \
    +dither -colors 64 -define png:compression-level=9 modes.png
  im montage -background '#101010' -tile 4x1 -geometry +8+8 \
    _p-cega.png _p-tdy.png _p-mono.png _p-cga.png \
    +dither -colors 64 -define png:compression-level=9 panels.png
  cp workplace/shots/modes.png workplace/shots/panels.png "$OUT/"
  echo "  modes.png / panels.png"
else
  echo "  ⚠ 找不到六模式齊全的資料目錄（$MODES_DATA），跳過 modes.png／panels.png"
fi

rm -f workplace/shots/_rm.png workplace/shots/_sty-*.png workplace/shots/_e-*.png \
      workplace/shots/_mode-*.png workplace/shots/_m-*.png workplace/shots/_p-*.png \
      workplace/shots/_tp0.png
echo "完成，圖在 $OUT"
