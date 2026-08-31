#!/usr/bin/env bash
set -euo pipefail

printf 'pcm.!default {\n type file\n file "/cap/out.raw"\n format raw\n slave { pcm "nullpcm" }\n}\npcm.nullpcm { type null }\n' > /tmp/.asoundrc

Xvfb :99 -screen 0 1280x960x24 -nolisten tcp >/tmp/xvfb.log 2>&1 &
XVFB=$!
trap 'kill ${GAME:-} "$XVFB" 2>/dev/null || true' EXIT
export DISPLAY=:99
for _ in $(seq 1 40); do xdpyinfo >/dev/null 2>&1 && break; sleep 0.25; done

go build -o /tmp/chengshi ./cmd/chengshi
/tmp/chengshi -data "workplace/dos110/SIMCITY 1.10" -music music -seed 7 -scale 1 \
  >/tmp/music-smoke.log 2>&1 &
GAME=$!
for _ in $(seq 1 100); do
  if [ -s /cap/out.raw ]; then
    break
  fi
  sleep 0.05
done
sleep 0.05
kill "$GAME" 2>/dev/null || true
wait "$GAME" 2>/dev/null || true

grep -v '^XGB:' /tmp/music-smoke.log || true

python3 - <<'PY'
import struct
from pathlib import Path

raw = Path('/cap/out.raw').read_bytes()
values = struct.iter_unpack('<f', raw[:len(raw) // 4 * 4])
if not any(v[0] != 0.0 for v in values):
    raise SystemExit('FAIL：Ebiten／oto 輸出全部是靜音')
print(f'pass：本機 OGG 已經過 Ebiten／oto 送出非零音訊（擷取 {len(raw)} bytes）')
PY
