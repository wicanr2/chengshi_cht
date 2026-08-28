# Micropolis（1993 年的 X11／Tcl/Tk 版）在現代 Linux 上的建置環境。
# 用途只有一個：把原版編出來當 oracle，供 Go 版逐 tick 對拍與畫面對照。
# 不散布它的產物；Micropolis 本身是 GPL-3.0，封存由使用者自備（見 CLAUDE.md §1.2）。
FROM debian:bookworm-slim

ARG DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
      build-essential \
      libx11-dev libxext-dev libxpm-dev \
      xvfb x11-utils x11-apps imagemagick \
      ca-certificates make gcc byacc flex \
    && rm -rf /var/lib/apt/lists/*

COPY compat.h /opt/compat.h

# 各層 makefile 自己寫死 CFLAGS，環境變數傳不進去，所以改用 gcc 包裝器：
# 一律補上「當年的預設行為」旗標與相容標頭，封存的原始碼不動一個 byte。
RUN printf '%s\n' \
      '#!/bin/sh' \
      'exec /usr/bin/gcc -std=gnu89 -fcommon -fno-strict-aliasing -w -include /opt/compat.h "$@"' \
      > /usr/local/bin/gcc \
    && chmod +x /usr/local/bin/gcc

WORKDIR /work
