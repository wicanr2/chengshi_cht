#!/usr/bin/env bash
# 在容器裡跑的 macOS 交叉編。不要直接執行，用 tools/build-mac.sh。
set -euo pipefail

OUT="${MAC_OUT:-dist/mac}"
case "$OUT" in
  dist/mac|/src/workplace/package-mac-*) ;;
  *) echo "拒絕不受控的 macOS 輸出路徑：$OUT" >&2; exit 2 ;;
esac

# 前綴帶 SDK 次版號（SDK 15.5 → darwin24.5）。讀 osxcross-conf，不要寫死。
eval "$(osxcross-conf)"
T="$OSXCROSS_TARGET"
MIN=10.15   # Ebiten 支援的最低 macOS
echo "== osxcross target $T，最低系統 $MIN =="

rm -rf "$OUT"
mkdir -p "$OUT"
for pair in "arm64 arm64" "amd64 x86_64"; do
  set -- $pair
  goarch=$1; carch=$2
  echo "-- 編 darwin/$goarch --"
  CGO_ENABLED=1 GOOS=darwin GOARCH=$goarch \
  CC="$carch-apple-$T-clang" CXX="$carch-apple-$T-clang++" \
  CGO_CFLAGS="-mmacosx-version-min=$MIN" \
  CGO_LDFLAGS="-mmacosx-version-min=$MIN" \
    go build -ldflags "-s -w -X main.version=$VER" \
      -o "$OUT/chengshi-$goarch" ./cmd/chengshi
done

lipo -create "$OUT/chengshi-arm64" "$OUT/chengshi-amd64" -output "$OUT/chengshi"
rm -f "$OUT/chengshi-arm64" "$OUT/chengshi-amd64"

echo
echo "== 靜態驗收 =="
FAIL=0
fail() { echo "FAIL  $*"; FAIL=1; }
pass() { echo "pass  $*"; }

# lipo 印出來的架構順序不固定，逐個找而不是比整串。
INFO=$(lipo -info "$OUT/chengshi")
if echo "$INFO" | grep -q arm64 && echo "$INFO" | grep -q x86_64; then
  pass "universal（arm64 ＋ x86_64）"
else
  fail "不是雙架構：$INFO"
fi

OTOOL="x86_64-apple-$T-otool"   # cctools 只裝了帶前綴的版本
for a in arm64 x86_64; do
  lipo -thin $a "$OUT/chengshi" -output "/tmp/thin-$a"
  # arm64 沒有 LC_CODE_SIGNATURE 的話，Apple Silicon 上會直接 Killed: 9。
  if [ "$a" = arm64 ]; then
    $OTOOL -l "/tmp/thin-$a" | grep -q LC_CODE_SIGNATURE \
      && pass "arm64 有 ad-hoc 簽章" || fail "arm64 少了 LC_CODE_SIGNATURE —— Apple Silicon 會 Killed: 9"
  fi
  minos=$($OTOOL -l "/tmp/thin-$a" | grep -m1 minos | tr -s ' ' | cut -d' ' -f3)
  pass "$a 最低系統 ${minos:-（讀不到）}"
  # 相依只能有系統庫。⚠ 對 fat binary 問 otool -L 會多印一行檔名標頭，
  # 每個架構各一行，看起來像「相依自己」——要先 lipo -thin 拆單弧再查。
  ext=$($OTOOL -L "/tmp/thin-$a" | tail -n +2 | awk '{print $1}' \
        | grep -vE '^(/usr/lib/|/System/Library/)' || true)
  [ -z "$ext" ] && pass "$a 只相依系統庫" || fail "$a 連到非系統庫：$ext"
done

# 這份 binary 真的含有本專案的東西嗎——找一定會出現的字串。
# ⚠ 不能用 `strings`：GNU strings 只認 0x20–0x7E，中文的 UTF-8 位元組
# 全部 >0x7F，會被當成分隔字元把字串切碎。直接 grep -a 二進位檔。
for want in "城市 — 模擬城市繁體中文 remake" "請用 -data 指向解開的 SimCity 1.10 目錄"; do
  grep -aq "$want" "$OUT/chengshi" \
    && pass "含「${want:0:16}…」" || fail "找不到「$want」"
done

# ── .app bundle ────────────────────────────────────────────────────
# 從 Finder 點開時沒有命令列參數，所以執行檔要自己找得到原版目錄
# （cmd/chengshi 的 findDataDir，會看 .app 旁邊與
# ~/Library/Application Support/chengshi/）。
APP="$OUT/城市.app"
rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
cp "$OUT/chengshi" "$APP/Contents/MacOS/chengshi"
cat > "$APP/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key><string>城市</string>
  <key>CFBundleDisplayName</key><string>城市</string>
  <key>CFBundleExecutable</key><string>chengshi</string>
  <key>CFBundleIdentifier</key><string>tw.wicanr2.chengshi</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>CFBundleShortVersionString</key><string>$VER</string>
  <key>CFBundleVersion</key><string>$VER</string>
  <key>LSMinimumSystemVersion</key><string>$MIN</string>
  <key>NSHighResolutionCapable</key><true/>
</dict>
</plist>
PLIST
pass "組好 .app（未簽名）"

echo
[ "$FAIL" = 0 ] && echo "== macOS 靜態驗收通過 ==" || { echo "== macOS 靜態驗收失敗 =="; exit 1; }
