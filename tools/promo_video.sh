#!/usr/bin/env bash
# 產生推廣影片：老遊戲廣告式的段落片，字卡 ＋ 實機畫面 ＋ 原版音效。
#
# 三個階段用兩個 image，因為工具不在同一個地方：遊戲要跑在 simcity-go
# （有 Go 與 Xvfb），編碼要跑在 simcity-sc2k-audio（有 ffmpeg）。
#
#   一、擷取   tools/promo_video_inner.sh     十二段實機畫面
#   二、素材   tools/promo_assets_inner.sh    原版音效、六種顯示模式對照、字卡
#   三、合成   tools/promo_assemble_inner.sh  編段落、串接、鋪音效、出 GIF
#
# 產出（都在 workplace/promo/，gitignore）：
#   promo.mp4  H.264 ＋ AAC
#   promo.gif  給 README 內嵌（沒有聲音）
#
# ⚠ 畫面上的圖塊與音效都來自玩家自備的原版；影片本身是本重製版跑出來的畫面。
# ⚠ **配樂只用原版真實素材**（~/.claude/rulebook/93 鐵則 1）。這款遊戲沒有
#   背景音樂（docs/re/19-no-music.md），所以聲音層就是原版那八段音效，
#   不自己合成旋律頂替。
#
# 用法：tools/promo_video.sh
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
OUT=workplace/promo
mkdir -p "$OUT"

go_run() {
  docker run --rm \
    --log-opt max-size=10m --log-opt max-file=3 \
    -u "$(id -u):$(id -g)" \
    --memory 4g --cpus 4 --pids-limit 512 \
    --network none \
    -v "$ROOT:/src" \
    -v "$ROOT/workplace/gocache:/gocache" -v "$ROOT/workplace/gomod:/gomod" \
    -e GOCACHE=/gocache -e GOMODCACHE=/gomod -e GOFLAGS=-mod=mod -e HOME=/tmp \
    -e XDG_CONFIG_HOME=/tmp/chengshi-config -e XDG_CACHE_HOME=/tmp \
    -e LIBGL_ALWAYS_SOFTWARE=1 -e GALLIUM_DRIVER=llvmpipe \
    -w /src simcity-go:1.25 bash "/src/$1"
}

echo "############ 一、擷取 ############"
go_run tools/promo_video_inner.sh

FRAMES=$(find "$OUT/frames" -name '*.png' | wc -l)
[ "$FRAMES" -gt 0 ] || { echo "沒有擷取到任何一格" >&2; exit 1; }
echo "擷取到 $FRAMES 格"

echo
echo "############ 二、素材 ############"
go_run tools/promo_assets_inner.sh

echo
echo "############ 三、合成 ############"
docker run --rm \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" \
  --memory 2g --cpus 2 --pids-limit 128 \
  --network none \
  -v "$ROOT/$OUT:/work" -v "$ROOT/tools:/tools:ro" -w /work \
  simcity-sc2k-audio:bookworm-r1 bash /tools/promo_assemble_inner.sh

echo
ls -la "$OUT"/promo.mp4 "$OUT"/promo.gif | awk '{printf "  %-28s %d 位元組\n", $9, $5}'
echo "完成。影片在 $OUT/"
