# Go 工具鏈 ＋ Ebiten 的建置相依。
#
# Ebiten 的桌面後端要 cgo，而 cgo 要 OpenGL 與 X11 的開發標頭檔。
# 少了它們的話 `go build` 會噴一堆 "fatal error: X11/Xlib.h: No such file"，
# 而 `go vet` 卻是綠的——因為 vet 不連結。
FROM golang:1.25-bookworm

RUN apt-get update && apt-get install -y --no-install-recommends \
      libgl1-mesa-dev \
      libx11-dev \
      libxrandr-dev \
      libxcursor-dev \
      libxinerama-dev \
      libxi-dev \
      libxxf86vm-dev \
      libasound2-dev \
      pkg-config \
      fonts-noto-cjk \
      python3 python3-pil \
      xvfb x11-utils x11-apps xdotool imagemagick libgl1 libglx-mesa0 mesa-utils \
 && rm -rf /var/lib/apt/lists/*
