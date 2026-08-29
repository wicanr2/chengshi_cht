# 跑 DOS 原版當 oracle。
#
# 用途不是「玩」，是回答只有原版跑起來才回答得了的問題：八段音效各對應
# 哪一個事件、取樣率多少、破解版到底有沒有拔掉手冊查驗。
#
# 用 DOSBox-X 而不是 Debian bookworm 的 dosbox 0.74：0.74 的 PC 喇叭
# 放不出遊戲的 4 位元 PCM（只出得了單頻方波），Covox 也接不上（它的
# disney 裝置只認 Disney Sound Source 的 FIFO 交握，而 Covox 是直接
# 對 LPT 資料埠寫值）。細節見 docs/re/16-dos-oracle.md §4。
#
# 音訊用 SDL 的 disk 驅動倒出來（`SDL_AUDIODRIVER=disk`），不必按
# DOSBox 的錄音快速鍵——headless 底下少一個會掉的環節。
FROM debian:trixie-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
      dosbox-x \
      xvfb x11-utils x11-apps xdotool imagemagick \
      python3 python3-pil \
      ca-certificates \
 && rm -rf /var/lib/apt/lists/*
