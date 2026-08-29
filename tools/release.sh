#!/usr/bin/env bash
# 打發行包。用法：tools/release.sh [版本]
#
# 包裡只有本專案自己的東西：執行檔（字型與譯文都內嵌在裡面）、授權條款、
# 讀我。**原版遊戲的圖形、音效、劇本一個位元組都不進去** —— 玩家自備。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VER="${1:-$(date +%Y%m%d)}"
OUT="$ROOT/dist"
rm -rf "$OUT"; mkdir -p "$OUT"

# Windows 走 CGO_ENABLED=0：Ebiten 的 Windows 後端是純 Go（自己載 DLL），
# 不需要 mingw。Linux 反過來一定要 cgo，因為它要連 X11 與 OpenGL。
# macOS 需要 Objective-C（Ebiten 的後端），交叉編要 osxcross，
# 由 tools/build-mac.sh 另外處理——它需要一個含 Apple SDK 的 image，
# 那個 image 不能散布，所以不是每台機器都有。沒有就跳過。
docker run --rm \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" \
  --memory 4g --cpus 2 --pids-limit 512 \
  --network none \
  -v "$ROOT:/src" \
  -v "$ROOT/workplace/gocache:/gocache" -v "$ROOT/workplace/gomod:/gomod" \
  -e GOCACHE=/gocache -e GOMODCACHE=/gomod -e GOFLAGS=-mod=mod -e HOME=/tmp \
  -e VER="$VER" \
  -w /src simcity-go:1.25 bash -euo pipefail -c '
    LD="-s -w -X main.version=$VER"
    CGO_ENABLED=1 GOOS=linux   GOARCH=amd64 go build -ldflags "$LD" -o dist/linux-amd64/chengshi     ./cmd/chengshi
    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$LD" -o dist/windows-amd64/chengshi.exe ./cmd/chengshi
  '

for p in linux-amd64 windows-amd64; do
  d="$OUT/$p"
  cp "$ROOT/LICENSE" "$d/LICENSE"
  cp "$ROOT/licenses/NotoSansCJK-copyright.txt" "$d/"
  cp "$ROOT/packaging/讀我.txt" "$d/"
done

# macOS：有 osxcross image 才做。
if docker image inspect simcity-osxcross:15.5 >/dev/null 2>&1; then
  "$ROOT/tools/build-mac.sh" "$VER"
  d="$OUT/mac"
  cp "$ROOT/LICENSE" "$d/LICENSE"
  cp "$ROOT/licenses/NotoSansCJK-copyright.txt" "$d/"
  cp "$ROOT/packaging/讀我.txt" "$d/"
  rm -f "$d/chengshi"   # 已經在 .app 裡了
else
  echo "沒有 simcity-osxcross:15.5，跳過 macOS（見 docker/osxcross.Dockerfile）"
fi

cd "$OUT"
tar -czf "chengshi_cht-$VER-linux-amd64.tar.gz" -C linux-amd64 .
[ -d mac ] && tar -czf "chengshi_cht-$VER-macos-universal.tar.gz" -C mac .
python3 - "$VER" <<'PY'
import os
import sys
import zipfile

ver = sys.argv[1]
src = "windows-amd64"
with zipfile.ZipFile(f"chengshi_cht-{ver}-windows-amd64.zip", "w",
                     zipfile.ZIP_DEFLATED) as z:
    for name in sorted(os.listdir(src)):
        z.write(os.path.join(src, name), name)
PY
rm -rf linux-amd64 windows-amd64 mac
ls -lh
echo
echo "發行包在 $OUT"
