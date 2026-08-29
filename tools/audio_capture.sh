#!/usr/bin/env bash
# 錄下遊戲真正送到音效裝置的位元組，驗「聲音有沒有出去、長度對不對」。
#
# 用法：tools/audio_capture.sh [段編號…]        （預設全部八段）
#
# 為什麼要這個：單元測試只能驗到 internal/audio 的輸出，驗不到
# 「Ebiten 有沒有真的把那些位元組送出去」。這支用 ALSA 的 file 外掛
# 把裝置寫入原樣落成檔案，再量非零區間的長度，和預期的段長對照。
#
# 實測（2026-08-30，48000 Hz float32 立體聲）：
#   段 4 船笛      預期 0.948 秒  錄到 0.934 秒  峰值 0.996
#   段 5 工具成功  預期 0.071 秒  錄到 0.069 秒  峰值 0.747
#   閒置時全 0——所以「有非零」本身就是聲音出去了的證據。
#
# ⚠ 錄下來的是 **float32** 立體聲，不是 int16：Ebiten 內部混音就是 float32。
# 照 int16 讀會得到剛好兩倍的長度與假的峰值 32768。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SEGS=("$@")
if [ ${#SEGS[@]} -eq 0 ]; then SEGS=(0 1 2 3 4 5 6 7); fi
for seg in "${SEGS[@]}"; do
  docker run --rm \
    --log-opt max-size=10m --log-opt max-file=3 \
    -u "$(id -u):$(id -g)" \
    --memory 4g --cpus 4 --pids-limit 512 \
    --network none \
    --tmpfs /cap:rw,size=300m \
    -v "$ROOT:/src" \
    -v "$ROOT/workplace/gocache:/gocache" -v "$ROOT/workplace/gomod:/gomod" \
    -e GOCACHE=/gocache -e GOMODCACHE=/gomod -e GOFLAGS=-mod=mod -e HOME=/tmp \
    -e LIBGL_ALWAYS_SOFTWARE=1 -e GALLIUM_DRIVER=llvmpipe \
    -w /src simcity-go:1.25 bash /src/tools/audio/capture_inner.sh "$seg" \
    2>&1 | grep -E "^(=== 段|總 float32|非零跨度|音效沒開起來)"
done
