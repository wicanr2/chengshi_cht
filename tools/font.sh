#!/usr/bin/env bash
# 在 docker 裡重烘中文點陣字型。用法：tools/font.sh
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
exec docker run --rm \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" \
  --memory 2g --cpus 2 --pids-limit 256 \
  --network none \
  -v "$ROOT:/src" -w /src \
  -e HOME=/tmp \
  simcity-go:1.25 python3 tools/build_font.py internal/textfont/assets
