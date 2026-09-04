#!/usr/bin/env bash
# 重產 README 的畫面截圖，繁體中文／英文／日文各一套。
#
# 為什麼要有腳本：這些圖是「現在的 remake 長什麼樣」的證據，版面一改就過期。
# 手動截圖沒有紀錄用了哪些參數，下次改完版面沒有人知道怎麼截出同一張。
#
# 三種語言各跑一遍，因為 README 有三份，每一份的畫面都要是那個語言：
#   zh-Hant → docs/images/       （正體版 README.md）
#   en      → docs/images/en/    （README_en.md）
#   ja      → docs/images/ja/    （README_jp.md）
# 只重跑一種語言：LANGS=ja tools/readme_shots.sh
#
# ⚠ 圖裡的地圖圖塊來自**玩家自己那份原版**（CLAUDE.md §8）：
# 本專案只放 remake 跑起來的畫面，不放原版遊戲本身的截圖。
#
# 用法：tools/readme_shots.sh
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# im 是 ImageMagick 的 docker 包裝（主機不裝東西，CLAUDE.md §6）。
# 工作目錄固定是 workplace/shots，所以檔名一律相對它。
im() { # 子命令 參數…
  local sub="$1"; shift
  docker run --rm --log-opt max-size=10m --log-opt max-file=3 \
    -u "$(id -u):$(id -g)" -v "$ROOT/workplace/shots:/w" -w /w \
    --entrypoint "$sub" dpokidov/imagemagick "$@" 2>/dev/null
}

# opt 把一張截圖降到 256 色再收進 $OUT。
#
# 原圖有七百多種顏色，全在中文字的邊緣（抗鋸齒）；畫面本體是點陣美術，
# 256 色動到的像素不到 0.4%，檔案卻小一半。三套語言各一份，這個差別
# 是 repo 多兩三 MB 或多五六 MB。`+dither` 是**關掉**抖動——點陣美術
# 抖過會長出原版沒有的雜點。
opt() { # 來源（相對 workplace/shots） 目標檔名
  im convert "$1" +dither -colors 256 -define png:compression-level=9 "$2"
  mv "$ROOT/workplace/shots/$2" "$OUT/$2"
}

shot() { # 檔名 GAME_ARGS GAME_KEYS 等待秒數
  local name="$1" args="$2" keys="${3:-}" wait="${4:-8}"
  GAME_ARGS="$LARG $args" GAME_KEYS="$keys" \
    ./tools/screenshot.sh "$wait" "_rm.png" >/dev/null 2>&1
  opt _rm.png "$name"
  echo "    $name"
}

# 六種顯示模式那兩張要圖形檔齊全的資料目錄：1.10 缺 CGA 與 Tandy，
# 1.03 缺 mcga 與資料片，兩片資料片沒有基本外觀。沒有那個目錄就跳過，
# 不要拿不齊的資料產一張少兩格的圖。
MODES_DATA="${MODES_DATA:-workplace/allmodes}"

for lang in ${LANGS:-zh-Hant en ja}; do
  case "$lang" in
    zh-Hant) OUT=docs/images;      LARG="" ;;
    *)       OUT="docs/images/$lang"; LARG="-lang $lang" ;;
  esac
  mkdir -p "$OUT"
  echo "== $lang → $OUT =="

  # 招牌。四個操作選項照目前語言顯示。
  shot title.png "-style base"
  # 底特律 1972，基本外觀。space 關掉劇本簡介、0 暫停讓畫面穩定。
  shot city.png       "-scenario 6 -style base -cam 30,30" "space 0"
  # 地形編輯器。`-window terrainparams` 開的是參數對話框，Return ＝ 按「開始」，
  # 產完地形回到編輯器本身——編輯器一開機是全空地，直接截會是一片褐色。
  # ⚠ 地形是 `RandomSeed()` 產的，每次跑都不一樣；這張要的是版面不是那張地圖。
  shot terrain-editor.png "-style base -window terrainparams" "Return" 9
  # 地形編修程式的「地形」選單拉下來。**證據性的一張**：畫面上的字
  # （開濶地／綠地／河流／航道／均佈／回手、清除非自然物、綠地平滑、
  # 產生島嶼地形）整組出自軟體世界 220 那本說明書，見
  # docs/manual-cht/naming-crosswalk.md 第二節。
  # ⚠ 按住的座標取選單列的**中間三分之一**，這樣三種語言都點得到那一欄。
  GAME_ARGS="$LARG -window terrain -seed 1990" GAME_HOLD="1040,28" \
    ./tools/screenshot.sh 9 "_rm.png" >/dev/null 2>&1
  opt _rm.png terrain-menu.png
  echo "    terrain-menu.png"

  # 資料片的工具名。點古代亞洲的警察局那一格，視窗左下角的圖形名稱欄
  # 就會顯示該圖形集自己的名字（「衙門」，出自電腦休閒世界 022 p.56）。
  #
  # ⚠ **只在繁中這一輪拍**：這一張是三份 README 共用的，因為它證明的是
  # 「中文用的是 1990 年代代理商說明書的字」——英日兩份 README 也引用同一張，
  # 並在圖說裡註明那是繁中版的畫面。拍成英日文的等於證不到任何事。
  # ⚠ 座標是工具盤第 5 列第 1 欄的格心：CEGA 的格線是
  # (8+2+29c, 55+5+25r)，UIScale 3 → (69, 513)。
  if [ "$lang" = zh-Hant ]; then
    GAME_ARGS="$LARG -scenario 6 -style asia -cam 30,30" GAME_KEYS="space 0" \
      GAME_CLICKS="69,513" ./tools/screenshot.sh 9 "_st0.png" >/dev/null 2>&1
    im convert _st0.png -crop 716x950+16+50 +repage \
      -filter point -resize 66.667% +dither -colors 64 \
      -define png:compression-level=9 _out.png
    mv workplace/shots/_out.png "$OUT/style-tools.png"
    echo "    style-tools.png"
  fi

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
  # 語言設定。`-seed` 寫死，不然每次跑背景那張新地圖都不一樣。
  shot settings.png   "-style base -seed 1990 -window language"

  # 五張地圖檔。軟體世界 1990 年的兩張加上本專案補的三張。
  # `-cam 0,0` 讓 City Form 的取景框停在左上角，`0` 暫停。
  for m in taiwan:TAIWAN kaohsiung:KAOHSIUN taipei:TAIPEI taichung:TAICHUNG tainan:TAINAN; do
    shot "map-${m%%:*}.png" "-load cities/${m##*:}.CTY -style base -cam 0,0" "0"
  done

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
  for sty in asia medi west fusa feur moon; do
    GAME_ARGS="$LARG -scenario 6 -style $sty -cam 30,30" GAME_KEYS="space 0" \
      ./tools/screenshot.sh 8 "_sty-$sty.png" >/dev/null 2>&1
    im convert "_sty-$sty.png" -crop 726x900+0+60 +repage \
      -filter point -resize 33.334% "_e-$sty.png"
  done
  im montage -background '#202020' -tile 3x1 -geometry +4+4 \
    _e-asia.png _e-medi.png _e-west.png +dither -colors 32 \
    -define png:compression-level=9 _out.png
  mv workplace/shots/_out.png "$OUT/styles-ancient.png"
  im montage -background '#202020' -tile 3x1 -geometry +4+4 \
    _e-fusa.png _e-feur.png _e-moon.png +dither -colors 32 \
    -define png:compression-level=9 _out.png
  mv workplace/shots/_out.png "$OUT/styles-future.png"
  echo "    styles-ancient.png / styles-future.png"

  # 地形編輯器的參數對話框（原版 `TERRAIN.EXE` 的那一個）。這一張縮回原版 1:1。
  GAME_ARGS="$LARG -window terrainparams" ./tools/screenshot.sh 9 "_tp0.png" >/dev/null 2>&1
  im convert _tp0.png -filter point -resize 33.334% \
    +dither -colors 32 -define png:compression-level=9 _out.png
  mv workplace/shots/_out.png "$OUT/terrain-params.png"
  echo "    terrain-params.png"

  if [ -d "$ROOT/$MODES_DATA/CGA" ] && [ -d "$ROOT/$MODES_DATA/tdy" ]; then
    for m in cega sega tdy mcga mono cga; do
      GAME_ARGS="$LARG -data $MODES_DATA -mode $m -style asia -load cities/TAIPEI.CTY" \
        ./tools/screenshot.sh 9 "_mode-$m.png" >/dev/null 2>&1
      im convert "_mode-$m.png" -filter point -resize 33.334% "_m-$m.png"
      im convert "_mode-$m.png" -crop 190x820+18+150 +repage \
        -filter point -resize 60% "_p-$m.png"
    done
    im montage -background '#404040' -tile 3x2 -geometry +2+2 \
      _m-cega.png _m-sega.png _m-tdy.png _m-mcga.png _m-mono.png _m-cga.png \
      +dither -colors 64 -define png:compression-level=9 _out.png
    mv workplace/shots/_out.png "$OUT/modes.png"
    echo "    modes.png"
    # 左欄面板那一張是**四個模式的工具盤**，裡面一個字都沒有（C·R·I 是
    # 原版美術），所以三種語言共用同一張，只在正體那一輪產。
    if [ "$lang" = zh-Hant ]; then
      im montage -background '#101010' -tile 4x1 -geometry +8+8 \
        _p-cega.png _p-tdy.png _p-mono.png _p-cga.png \
        +dither -colors 64 -define png:compression-level=9 _out.png
      mv workplace/shots/_out.png "$OUT/panels.png"
      echo "    panels.png"
    fi
  else
    echo "    ⚠ 找不到六模式齊全的資料目錄（$MODES_DATA），跳過 modes.png／panels.png"
  fi
done

rm -f workplace/shots/_rm.png workplace/shots/_sty-*.png workplace/shots/_e-*.png \
      workplace/shots/_mode-*.png workplace/shots/_m-*.png workplace/shots/_p-*.png \
      workplace/shots/_tp0.png workplace/shots/_out.png
echo "完成"
