#!/usr/bin/env bash
# DOS 原版與 remake 的抽樣對拍。
#
# 做法：在 DOSBox 裡跑原版、載入一個悲情城市、**立刻**用 Ctrl-S 存檔，
# 把那份 `.cty` 帶出來；remake 從同一個劇本出發跑到同一個 CityTime，
# 再逐格比地圖並列出一組純量。
#
# 用法：MODE=load|run tools/dos_parity.sh [劇本編號…]   （預設 load、八個都跑）
#
# 為什麼只能「抽樣」而不是逐 tick：DOS 版載入時自己重設亂數種子，我們沒有
# 辦法把它的亂數狀態設成一樣，所以逐 tick 完全一致**在原理上做不到**
# （Micropolis 那一側做得到，因為 oracle 讓我們讀得到狀態，見 docs/re/12）。
#
# 兩個取樣點：
#
#   load  載入後**立刻**存檔。跑不到四刻，比出來的幾乎全是載入器的正確性。
#   run   打開 Auto-Budget、催到 Fastest、讓它自己跑一段再存。這一份才是
#         在比模擬——劇本 1 實測跑了 27 個遊戲年。
#
# ⚠ **run 的秒數要卡在劇本時限以內。** 時限到了會判定勝敗，輸的那一刻
# `DoLoseGame` 直接把你踢回標題畫面，存不出東西（劇本 1 用 120 秒踩過）。
# 時限見 s_sim.c:384 的 ScoreWaitTab：劇本 1 是 30 年，2／3／5／7 是 5 年，
# 4／6／8 是 10 年。實測 Fastest 大約 0.7 個遊戲年／秒（cycles 固定 20000）。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODE="${MODE:-load}"          # load｜run
TMPL="$ROOT/tools/dosbox/act-scen-$MODE.txt"

# 每個劇本讓它跑幾秒（只有 MODE=run 用得到）。取時限的八成左右。
RUNSECS=(30 5 5 10 5 10 5 10)
OUT="$ROOT/workplace/dosbox"
mkdir -p "$OUT/save"

# 劇本選擇畫面的座標：兩列四行。
XS=(80 240 402 562)
YS=(103 253)

run_one() {
  local n=$1
  local i=$(( (n-1) % 4 )) j=$(( (n-1) / 4 ))
  local act="$OUT/_act-load.txt"
  sed -e "s/__SCEN_X__/${XS[$i]}/; s/__SCEN_Y__/${YS[$j]}/; s/__RUN__/${RUNSECS[$((n-1))]}/" \
      "$TMPL" > "$act"
  # ⚠ 只清掉這一輪要重生的兩個，**不要** `rm *.cty`——那會把前面幾個劇本
  # 已經取好的樣本一起刪掉（踩過一次，跑完只剩最後一個）。
  rm -f "$OUT/save/A.cty" "$OUT"/save/*.CTY
  TIMEOUT=420 RUN=simcity ACTIONS="$act" "$ROOT/tools/dosbox.sh" 4 "${MODE}$n" >/dev/null 2>&1 || true
  if [ ! -f "$OUT/save/A.cty" ]; then
    echo "劇本 $n：原版沒有存出城市檔——看 $OUT/${MODE}$n-*.png"
    return 1
  fi
  local tag="scen"; [ "$MODE" = run ] && tag="run"
  cp "$OUT/save/A.cty" "$OUT/save/$tag$n.cty"
  # ⚠ 路徑要**相對於 repo 根目錄**：go.sh 在容器裡跑，主機的絕對路徑
  # 在容器裡不存在（症狀是 Go 那邊回「no such file」，而檔案明明在）。
  (cd "$ROOT" && ./tools/go.sh run ./cmd/simtool dosparity-scen "$n" \
      "workplace/dosbox/save/$tag$n.cty")
}

SCENS=("$@")
[ ${#SCENS[@]} -eq 0 ] && SCENS=(1 2 3 4 5 6 7 8)
for n in "${SCENS[@]}"; do
  echo "================ 劇本 $n ================"
  run_one "$n" || true
done
