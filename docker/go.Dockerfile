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
      ca-certificates curl file patchelf desktop-file-utils \
 && rm -rf /var/lib/apt/lists/*

# Linux AppImage 工具固定下載內容雜湊。continuous URL 只是上游發行入口；真正的
# 可重現版本由 SHA-256 鎖住，任何上游替換都會讓 image build 失敗即關閉。
ARG LINUXDEPLOY_URL=https://github.com/linuxdeploy/linuxdeploy/releases/download/continuous/linuxdeploy-x86_64.AppImage
ARG APPIMAGETOOL_URL=https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage
ARG APPIMAGE_RUNTIME_URL=https://github.com/AppImage/type2-runtime/releases/download/continuous/runtime-x86_64
ARG LINUXDEPLOY_SHA256=421ca71d5c69ea97c6309276232990d43df1dcece0edfaa26bbf926ff96ed12e
ARG APPIMAGETOOL_SHA256=a6d71e2b6cd66f8e8d16c37ad164658985e0cf5fcaa950c90a482890cb9d13e0
ARG APPIMAGE_RUNTIME_SHA256=1cc49bcf1e2ccd593c379adb17c9f85a36d619088296504de95b1d06215aebbf
RUN mkdir -p /opt/appimage-tools \
 && curl -fL --retry 3 -o /opt/appimage-tools/linuxdeploy.AppImage "$LINUXDEPLOY_URL" \
 && curl -fL --retry 3 -o /opt/appimage-tools/appimagetool.AppImage "$APPIMAGETOOL_URL" \
 && curl -fL --retry 3 -o /opt/appimage-tools/runtime-x86_64 "$APPIMAGE_RUNTIME_URL" \
 && echo "$LINUXDEPLOY_SHA256  /opt/appimage-tools/linuxdeploy.AppImage" | sha256sum -c - \
 && echo "$APPIMAGETOOL_SHA256  /opt/appimage-tools/appimagetool.AppImage" | sha256sum -c - \
 && echo "$APPIMAGE_RUNTIME_SHA256  /opt/appimage-tools/runtime-x86_64" | sha256sum -c - \
 && chmod 0755 /opt/appimage-tools/linuxdeploy.AppImage \
               /opt/appimage-tools/appimagetool.AppImage \
               /opt/appimage-tools/runtime-x86_64
