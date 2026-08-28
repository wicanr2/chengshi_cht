#!/usr/bin/env bash
# 在 docker + Xvfb 裡用 pty 驅動 Micropolis oracle 跑一份 Tcl 指令檔。
# 用法：tools/oracle/drive.sh tools/oracle/tcl/smoke.tcl smoke.json
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SRC="$ROOT/workplace/ref/micropolis/micropolis-activity"
OUT="$ROOT/workplace/oracle"
IMAGE="simcity-oracle:bookworm"
SCRIPT="${1:?要給一個 Tcl 指令檔}"
RESULT="${2:-result.json}"

[ -x "$SRC/res/sim" ] || { echo "找不到 $SRC/res/sim —— 先跑 tools/oracle/build.sh"; exit 1; }
mkdir -p "$OUT"
cp "$SCRIPT" "$OUT/_script.tcl"

docker run --rm \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" \
  --memory 2g --cpus 2 --pids-limit 256 \
  --network none \
  -v "$SRC:/work" -v "$OUT:/out" -v "$ROOT/tools/oracle:/drv:ro" \
  -w /work -e SIMHOME=/work -e HOME=/tmp \
  "$IMAGE" \
  bash -c "
    set -e
    Xvfb :99 -screen 0 1280x1024x24 -nolisten tcp >/tmp/xvfb.log 2>&1 &
    export DISPLAY=:99
    for i in \$(seq 1 40); do xdpyinfo >/dev/null 2>&1 && break; sleep 0.25; done
    python3 /drv/drive.py /out/_script.tcl /out/$RESULT
  "
