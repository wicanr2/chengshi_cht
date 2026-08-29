#!/usr/bin/env bash
# 在 docker 裡跑 IDA Pro 9.4 headless。
#
# 這個專案的反組譯是**退路不是主線**（CLAUDE.md §2.4）：規則層讀 Micropolis，
# 呈現層讀 DOS 資料檔，只有兩邊都答不了的問題才動 IDA。目前唯一的用途是
# docs/re/18-dos-parity.md §6 那個汙染算法的差異。
#
# ⚠ 這份 SIMCITY.EXE 是被改過的破解版（CLAUDE.md §2.1），
# 從它反組譯得到的結論要標明來源版本存疑。
#
# 用法：
#   tools/ida.sh analyze SIMCITY.EXE          # 建 .i64（放 workplace/ida/）
#   tools/ida.sh py tools/ida/x.py SIMCITY.EXE.i64 out.json
#   tools/ida.sh raw idat -A -B SIMCITY.EXE   # 自己下指令
#
# ⚠ 對同一個 .i64 連續快速開關會把資料庫弄壞（"Failed to initialize IDA as
# library (error code 4)"）。要查很多東西就寫一支一次處理完的腳本，
# 不要用 shell 迴圈跑 N 次。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORK="$ROOT/workplace/ida"
IMAGE="${IDA_IMAGE:-ida-pro-9.4-idapython:locked-v1}"
mkdir -p "$WORK"

run() {
  # --network none：IDA 不需要網路。license 只唯讀掛載，不進 log。
  docker run --rm \
    --log-opt max-size=10m --log-opt max-file=3 \
    -u "$(id -u):$(id -g)" \
    --memory 4g --cpus 2 --pids-limit 512 \
    --network none \
    -v "$WORK:/work" -v "$ROOT/tools/ida:/scripts:ro" \
    -w /work -e HOME=/home/ubuntu \
    "$IMAGE" "$@"
}

cmd="${1:?analyze|py|raw}"; shift
case "$cmd" in
  analyze)
    f="${1:?要給執行檔名（相對 workplace/ida/）}"
    run idat -A -B "$f" || true
    ls -l "$WORK/$f.i64" "$WORK/$f.asm" 2>/dev/null || { echo "沒產出 .i64 —— 看 $WORK"; exit 1; }
    ;;
  py)
    script="${1:?腳本}"; db="${2:?.i64}"; out="${3:?輸出檔}"
    run idat -A "-S/scripts/$(basename "$script") /work/$out" "$db"
    # ⚠ exit code 不能當證據（見 knowledge-base/retro/ida-pro-9.4.md）。
    [ -s "$WORK/$out" ] || { echo "$out 沒產出或是空的 —— 腳本沒跑到"; exit 1; }
    ;;
  raw) run "$@" ;;
  *) echo "不認得 $cmd"; exit 2 ;;
esac
