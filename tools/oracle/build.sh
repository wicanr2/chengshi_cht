#!/usr/bin/env bash
# 在 docker 裡建置 Micropolis oracle。
# 用法：tools/oracle/build.sh
# 前提：workplace/ref/micropolis/ 已有封存（見 CONTEXT.md 工作清單第 1 項）。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SRC="$ROOT/workplace/ref/micropolis/micropolis-activity"
IMAGE="simcity-oracle:bookworm"

[ -d "$SRC" ] || { echo "找不到 $SRC —— 先取得 Micropolis 封存"; exit 1; }

if ! docker image inspect "$IMAGE" >/dev/null 2>&1; then
  echo "== 建 image $IMAGE =="
  docker build -f "$ROOT/docker/oracle.Dockerfile" -t "$IMAGE" "$ROOT/docker"
fi

echo "== 建置 Micropolis =="
docker run --rm \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" \
  --memory 4g --cpus 4 --pids-limit 512 \
  --network none \
  -v "$SRC:/work" \
  -w /work \
  "$IMAGE" \
  bash -c '
    set -e
    export CFLAGS="$CFLAGS_COMPAT -O2"
    make 2>&1 | tail -40
  '
