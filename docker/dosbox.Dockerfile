# 跑 DOS 原版當 oracle。
#
# 用途不是「玩」，是回答只有原版跑起來才回答得了的問題：八段音效各對應
# 哪一個事件、取樣率多少、破解版到底有沒有拔掉手冊查驗。
#
# 音訊用 SDL 1.2 的 disk 驅動倒出來（`SDL_AUDIODRIVER=disk`），不必按
# DOSBox 的錄音快速鍵——headless 底下少一個會掉的環節。
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
      dosbox \
      xvfb x11-utils x11-apps xdotool imagemagick \
      python3 python3-pil \
      ca-certificates \
 && rm -rf /var/lib/apt/lists/*
