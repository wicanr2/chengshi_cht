#!/usr/bin/env bash
# 在容器裡跑：起 Xvfb、開遊戲、播指定的那一段音效、錄下真正送到音效裝置的位元組。
# 由 tools/audio_capture.sh 呼叫，不要直接跑。
set -uo pipefail
SEG="${1:?要給段編號 0–7}"

# ALSA 的 file 外掛把送出去的位元組原樣寫進檔案，slave 用 null（沒有硬體）。
# ⚠ null 不做節流，寫入速度是磁碟速度不是播放速度——所以 /cap 一定要是
# 有大小上限的 tmpfs，而且只有**最前面那一段**是有效的。
printf 'pcm.!default {\n type file\n file "/cap/out.raw"\n format raw\n slave { pcm "nullpcm" }\n}\npcm.nullpcm { type null }\n' > /tmp/.asoundrc

Xvfb :99 -screen 0 1280x960x24 -nolisten tcp >/tmp/xvfb.log 2>&1 &
export DISPLAY=:99
for i in $(seq 1 40); do xdpyinfo >/dev/null 2>&1 && break; sleep 0.25; done

go build -o /tmp/chengshi ./cmd/chengshi || exit 1
/tmp/chengshi -data "workplace/dos110/SIMCITY 1.10" -seed 7 -scale 1 -sound-test "$SEG" >/tmp/g.log 2>&1 &
G=$!
sleep 8
kill $G 2>/dev/null
sleep 1

echo "=== 段 $SEG ==="
grep -v XGB /tmp/g.log | head -2
python3 /src/tools/audio/scan.py
