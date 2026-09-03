#!/usr/bin/env bash
# 重現 issue #1 第三項：「蓋一段時間後畫面變雪花亂碼卡死，音樂還在」。
#
# 作法是把玩家那段時間壓縮：模擬跑最快，同時不停選工具、在地圖上點，
# 每十秒記一次三個數字——
#
#   顏色數  畫面只有十六色，變成雪花會爆增，這是最直接的偵測器
#   RSS     記憶體洩漏會單調成長
#   CPU 秒  凍住但 CPU 滿載 ＝ 無窮迴圈；凍住且 CPU 不動 ＝ 死鎖
#
# 用法：MINUTES=6 tools/stress_ui.sh
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
exec docker run --rm \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" \
  --memory 4g --cpus 4 --pids-limit 512 \
  --network none \
  -v "$ROOT:/src" \
  -v "$ROOT/workplace/gocache:/gocache" -v "$ROOT/workplace/gomod:/gomod" \
  -e GOCACHE=/gocache -e GOMODCACHE=/gomod -e GOFLAGS=-mod=mod -e HOME=/tmp \
  -e XDG_CONFIG_HOME=/tmp/cfg -e XDG_CACHE_HOME=/tmp \
  -e LIBGL_ALWAYS_SOFTWARE=1 -e GALLIUM_DRIVER=llvmpipe \
  -e MINUTES="${MINUTES:-6}" \
  -w /src simcity-go:1.25 bash /src/tools/stress_ui_inner.sh
