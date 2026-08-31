FROM debian:bookworm-20250929-slim

# SimCity 2000 的 General MIDI 離線渲染與 OGG 驗收工具鏈。
# XMI → MIDI 由另行固定 commit 的 XMI2MID 完成；這個 image 只負責合成與轉碼。
RUN apt-get update \
 && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
      ffmpeg \
      fluid-soundfont-gm \
      fluidsynth \
 && rm -rf /var/lib/apt/lists/*
