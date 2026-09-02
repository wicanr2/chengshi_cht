#!/usr/bin/env bash
# 產生推廣影片：五張台灣地圖在本重製版裡跑起來的運鏡。
#
# 兩個階段用兩個 image，因為工具不在同一個地方：遊戲要跑在 simcity-go
# （有 Go 與 Xvfb），編碼要跑在 simcity-sc2k-audio（有 ffmpeg）。
#
# 產出（都在 workplace/promo/，gitignore）：
#   promo.mp4  H.264，給發行包與網頁
#   promo.gif  給 README 內嵌（GitHub 不會播 mp4）
#
# ⚠ 影片裡的地圖圖塊來自玩家自備的原版；影片本身是本重製版跑出來的畫面。
#
# 用法：tools/promo_video.sh
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
OUT=workplace/promo
mkdir -p "$OUT"

echo "############ 第一階段：擷取 ############"
docker run --rm \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" \
  --memory 4g --cpus 4 --pids-limit 512 \
  --network none \
  -v "$ROOT:/src" \
  -v "$ROOT/workplace/gocache:/gocache" -v "$ROOT/workplace/gomod:/gomod" \
  -e GOCACHE=/gocache -e GOMODCACHE=/gomod -e GOFLAGS=-mod=mod -e HOME=/tmp \
  -e XDG_CONFIG_HOME=/tmp/chengshi-config \
  -e LIBGL_ALWAYS_SOFTWARE=1 -e GALLIUM_DRIVER=llvmpipe \
  -w /src simcity-go:1.25 bash /src/tools/promo_video_inner.sh

FRAMES=$(find "$OUT/frames" -name '*.png' | wc -l)
[ "$FRAMES" -gt 0 ] || { echo "沒有擷取到任何一格" >&2; exit 1; }
echo "擷取到 $FRAMES 格"

echo
echo "############ 第二階段：編碼 ############"
docker run --rm \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" \
  --memory 2g --cpus 2 --pids-limit 128 \
  --network none \
  -v "$ROOT/$OUT:/work" -w /work \
  simcity-sc2k-audio:bookworm-r1 bash -c '
set -e
# 1920×1050 縮到 960，畫面是整數倍縮放的整數分之一，圖塊不會糊掉。
ffmpeg -y -loglevel error -framerate 12 -pattern_type glob -i "frames/*.png" \
  -vf "scale=960:-2:flags=neighbor" -c:v libx264 -pix_fmt yuv420p -crf 20 promo.mp4
# GIF 走兩趟調色盤，不然十六色的畫面會被抖動糊成一片。
ffmpeg -y -loglevel error -framerate 12 -pattern_type glob -i "frames/*.png" \
  -vf "scale=640:-2:flags=neighbor,palettegen=stats_mode=diff" palette.png
ffmpeg -y -loglevel error -framerate 12 -pattern_type glob -i "frames/*.png" -i palette.png \
  -lavfi "scale=640:-2:flags=neighbor[v];[v][1:v]paletteuse=dither=none" promo.gif
rm -f palette.png
'

echo
ls -la "$OUT"/promo.mp4 "$OUT"/promo.gif | awk '{printf "  %-10s %d 位元組\n", $9, $5}'
echo "完成。影片在 $OUT/"
