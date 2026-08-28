#!/usr/bin/env bash
# 在 docker 裡把譯文合併進翻譯檔。用法：tools/i18n.sh
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
exec docker run --rm \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" \
  --memory 1g --cpus 1 --pids-limit 128 --network none \
  -v "$ROOT:/src" -w /src -e HOME=/tmp \
  simcity-go:1.25 python3 tools/i18n/merge.py translations/messages
