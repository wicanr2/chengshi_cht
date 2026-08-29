#!/usr/bin/env bash
# 在 docker + Xvfb 裡跑 Micropolis oracle，並可截圖。
# 用法：
#   tools/oracle/run.sh                       # 跑起來、等 8 秒、截一張圖
#   tools/oracle/run.sh 20 out.png            # 等 20 秒再截圖到 workplace/oracle/out.png
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SRC="$ROOT/workplace/oracle-build/micropolis-activity"
OUT="$ROOT/workplace/oracle"
IMAGE="simcity-oracle:bookworm"
WAIT="${1:-8}"
SHOT="${2:-startup.png}"

[ -x "$SRC/res/sim" ] || { echo "找不到 $SRC/res/sim —— 先跑 tools/oracle/build.sh"; exit 1; }
mkdir -p "$OUT"

docker run --rm \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" \
  --memory 2g --cpus 2 --pids-limit 256 \
  --network none \
  -v "$SRC:/work" -v "$OUT:/out" \
  -w /work \
  -e SIMHOME=/work \
  -e HOME=/tmp \
  "$IMAGE" \
  bash -c "
    set -e
    Xvfb :99 -screen 0 1280x1024x24 -nolisten tcp >/tmp/xvfb.log 2>&1 &
    XVFB=\$!
    export DISPLAY=:99
    for i in \$(seq 1 40); do xdpyinfo >/dev/null 2>&1 && break; sleep 0.25; done
    res/sim >/out/sim.stdout 2>/out/sim.stderr &
    SIM=\$!
    sleep $WAIT
    if kill -0 \$SIM 2>/dev/null; then
      echo '== sim 還活著，截圖 =='
      import -window root /out/$SHOT 2>/dev/null || xwd -root -silent | convert xwd:- /out/$SHOT
      kill \$SIM 2>/dev/null || true
    else
      echo '== sim 已經結束，離開碼 =='; wait \$SIM || echo \"exit=\$?\"
    fi
    kill \$XVFB 2>/dev/null || true
  "
echo "--- stdout ---"; tail -20 "$OUT/sim.stdout" 2>/dev/null || true
echo "--- stderr ---"; tail -20 "$OUT/sim.stderr" 2>/dev/null || true
ls -la "$OUT"
