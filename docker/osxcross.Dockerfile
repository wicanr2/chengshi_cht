# Go ＋ osxcross：在 Linux 上編 macOS 版。
#
# Ebiten 的 macOS 後端是 Objective-C，一定要 cgo，所以不能像 Windows 那樣
# 用 CGO_ENABLED=0 交叉編。osxcross 補的是 SDK、Mach-O 連結器（cctools-port）
# 與一層包好旗標的 wrapper——clang 本來就編得出 macOS 的機器碼。
#
# ⚠ **這個 image 含 Apple 的 SDK，不要推出去、不要散布。** SDK 的授權只允許
# 在 Apple 硬體上使用；自用交叉編是一回事，散布含 SDK 的 image 是另一回事。
# ⚠ 這條路**不做 notarization**（那一定要 Mac）。產出的 .app 不簽名，
# 首次開啟要右鍵 → 打開。
FROM crazymax/osxcross:15.5-debian AS osxcross

FROM golang:1.25-trixie
RUN apt-get update && apt-get install -y --no-install-recommends \
      clang lld llvm libssl-dev liblzma-dev libxml2-dev zlib1g-dev file \
 && rm -rf /var/lib/apt/lists/*
COPY --from=osxcross /osxcross /osxcross
# 少了這一行 ld64 起不來，clang 只會轉述成「找不到指令」。
RUN echo /osxcross/lib > /etc/ld.so.conf.d/osxcross.conf && ldconfig
ENV PATH="/osxcross/bin:${PATH}"
