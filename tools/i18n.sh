#!/usr/bin/env bash
# 把原版 `.PTF` 裡新出現的鍵補進翻譯用的 TSV。
#
# ⚠ **它只補空列、只更新長度，一個譯文都不動。** 舊版（`merge.py`）是
# 「python 表 → 整檔重寫」，會把直接編在產出檔裡的譯文洗掉——
# 實際發生過六筆（asia 的「水車」變回「核能發電廠」）。
# 現在 `internal/i18n/messages/*.tsv` 是唯一真相。
#
# **預設空跑**，只印差異；要真的寫入加 --write（然後自己看一遍新增的
# 是不是真的文字——`.PTF` 有些段落夾著屬性位元組，不是給玩家看的）。
#
# 用法：tools/i18n.sh [解開的 SIMCITY 1.10 目錄] [--write]
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DATA="${1:-$ROOT/workplace/dos110/SIMCITY 1.10}"
exec docker run --rm \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" \
  --memory 1g --cpus 1 --pids-limit 128 --network none \
  -v "$ROOT:/src" -v "$DATA:/data:ro" -w /src -e HOME=/tmp \
  simcity-go:1.25 python3 tools/i18n/skeleton.py /data internal/i18n/messages "${2:-}"
