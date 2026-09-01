#!/usr/bin/env bash
# 驗**發行的 AppImage** 與 DOS 原版對不對得上（畫面與操作流程，不含文字）。
#
# 跟 tools/screen_parity.sh 的差別：那一支的 remake 側是工作樹的原始碼
# （`go run`），驗的是「這份程式碼畫得對不對」。這一支的 remake 側是
# **打包完的那顆 AppImage**，多驗了包裝這一層：建置旗標、內嵌字型與圖集、
# AppImage runtime、資料路徑解析。這幾層壞掉的話，原始碼側的對拍全綠，
# 玩家拿到手的東西照樣是錯的。
#
# 兩個階段：
#   A 同源證明 —— 十四幕，AppImage 與現建版逐像素比。相同的話，既有的
#     原版對拍數字整批繼承到發行包，不必為包裝層再跟原版吵一次像素。
#   B 原版抽查 —— 現跑 DOSBox 產基準，拿 AppImage 的截圖重算數字。
#
# 用法：tools/appimage_parity.sh [版本]        （預設 v.1.0.0-20260901）
#       STAGE=a|b|ab tools/appimage_parity.sh  （只跑其中一階段）
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
VER="${1:-v.1.0.0-20260901}"
STAGE="${STAGE:-ab}"
APP="dist-all/$VER/full/chengshi-$VER-full-local-linux-amd64.AppImage"
[ -x "$APP" ] || { echo "找不到完整版 AppImage：$APP" >&2; exit 2; }
OUT=workplace/appimage-parity
mkdir -p "$OUT"

if [[ "$STAGE" == *a* ]]; then
  echo "############ 階段 A：同源證明 ############"
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
    -e APPIMAGE="/src/$APP" -e VER="$VER" \
    -e A_KIND="${A_KIND:-dev}" -e B_KIND="${B_KIND:-app}" \
    -e ONLY="${ONLY:-}" -e EARLY_PAUSE="${EARLY_PAUSE:-1}" \
    -w /src simcity-go:1.25 bash /src/tools/appimage_parity_inner.sh \
    2>&1 | tee "$OUT/stage-a.log"
fi

if [[ "$STAGE" == *b* ]]; then
  echo
  echo "############ 階段 B：現跑原版抽查 ############"
  # remake 側全部改由 AppImage 產生。GAME_BIN 是容器內路徑。
  export GAME_BIN="/src/$APP"

  echo "== B1／B2：招牌與劇本選單（原版的兩幅 .PPF）=="
  RUN=simcity ACTIONS="$ROOT/tools/dosbox/act-ppf.txt" timeout 300 \
    ./tools/dosbox.sh 20 apppf >/dev/null 2>&1
  GAME_STABLE=1 ./tools/screenshot.sh 7 app-title.png >/dev/null 2>&1
  GAME_CLICKS="948,714" GAME_STABLE=1 ./tools/screenshot.sh 7 app-scenmenu.png >/dev/null 2>&1
  for pair in "title:apppf-00-title:app-title" "scenmenu:apppf-01-scen:app-scenmenu"; do
    label=${pair%%:*}; rest=${pair#*:}; dos=${rest%%:*}; rem=${rest#*:}
    docker run --rm --log-opt max-size=10m --log-opt max-file=3 \
      -u "$(id -u):$(id -g)" --memory 1g --cpus 1 --pids-limit 64 --network none \
      -v "$ROOT:/src" -w /src simcity-dosbox:x python3 tools/shot_diff_ui.py \
      "workplace/dosbox/$dos.png" "workplace/shots/$rem.png" --profile ppf \
      --state "DOS1.10/AppImage $VER;PPF=$label;fade=complete" \
      --classification exact-state --out "$OUT/$label-report"
  done

  echo
  echo "== B3／B4：編輯視窗 512 格與 City Form（現跑 DOS 基準）=="
  MIN_HIT="${MIN_HIT:-490}" ./tools/screen_parity.sh 1 west
  cp -r workplace/screen-parity "$OUT/screen-parity" 2>/dev/null || true
fi

echo
echo "收據在 $OUT/"
