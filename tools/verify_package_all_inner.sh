#!/usr/bin/env bash
# tools/verify_package_all.sh 的容器內本體。
set -euo pipefail

: "${VER:?缺少 VER}"
ROOT=/src
BASE="$ROOT/dist-all/$VER"
TMP="/tmp/chengshi-package-$VER"
RECEIPT="$ROOT/workplace/package-verify-$VER"
FAIL=0
XVFB_PID=

pass() { printf 'pass  %s\n' "$*"; }
fail() { printf 'FAIL  %s\n' "$*"; FAIL=1; }
cleanup() {
  if [ -n "$XVFB_PID" ]; then kill "$XVFB_PID" 2>/dev/null || true; fi
  rm -rf "$TMP"
}
trap cleanup EXIT
rm -rf "$RECEIPT"
mkdir -p "$RECEIPT"

for d in full release promo; do
  [ -d "$BASE/$d" ] && pass "有 $d/" || fail "少了 $d/"
  [ -s "$BASE/$d/SHA256SUMS" ] && pass "$d 有 SHA256SUMS" || fail "$d 少了 SHA256SUMS"
done
[ -s "$BASE/MANIFEST.json" ] && pass "有總 manifest" || fail "少了 MANIFEST.json"
[ -s "$BASE/release/MANIFEST.json" ] && pass "公開包有 manifest" || fail "公開包少了 manifest"

for d in full release promo; do
  if [ -d "$BASE/$d" ] && (cd "$BASE/$d" && sha256sum -c SHA256SUMS >/dev/null); then
    pass "$d 雜湊全數吻合"
  else
    fail "$d 雜湊不吻合"
  fi
done

mkdir -p "$TMP/release" "$TMP/full"
REL=$(find "$BASE/release" -maxdepth 1 -name '*-linux-amd64.tar.gz' -print -quit)
FULL=$(find "$BASE/full" -maxdepth 1 -name '*-full-local-linux-amd64.tar.gz' -print -quit)
MAC_REL=$(find "$BASE/release" -maxdepth 1 -name '*-macos-universal.tar.gz' -print -quit)
MAC_FULL=$(find "$BASE/full" -maxdepth 1 -name '*-full-local-macos-universal.tar.gz' -print -quit)
APP_REL=$(find "$BASE/release" -maxdepth 1 -name '*-linux-amd64.AppImage' -print -quit)
APP_FULL=$(find "$BASE/full" -maxdepth 1 -name '*-full-local-linux-amd64.AppImage' -print -quit)
[ -n "$REL" ] || fail "公開版缺 Linux tar.gz"
[ -n "$FULL" ] || fail "完整版缺 Linux tar.gz"
[ -n "$MAC_REL" ] || fail "公開版缺 macOS universal tar.gz"
[ -n "$MAC_FULL" ] || fail "完整版缺 macOS universal tar.gz"
[ -n "$APP_REL" ] || fail "公開版缺 Linux AppImage"
[ -n "$APP_FULL" ] || fail "完整版缺 Linux AppImage"
if [ -n "$REL" ]; then tar -xzf "$REL" -C "$TMP/release"; fi
if [ -n "$FULL" ]; then tar -xzf "$FULL" -C "$TMP/full"; fi

mkdir -p "$TMP/mac-release" "$TMP/mac-full"
if [ -n "$MAC_REL" ]; then tar -xzf "$MAC_REL" -C "$TMP/mac-release"; fi
if [ -n "$MAC_FULL" ]; then tar -xzf "$MAC_FULL" -C "$TMP/mac-full"; fi
for f in "城市.app/Contents/MacOS/chengshi" "城市.app/Contents/Info.plist" \
         LICENSE NotoSansCJK-copyright.txt 讀我.txt 素材與權利.txt; do
  [ -s "$TMP/mac-release/$f" ] && pass "macOS 公開版有 $f" || fail "macOS 公開版少了 $f"
done
[ -x "$TMP/mac-release/城市.app/Contents/MacOS/chengshi" ] \
  && pass "macOS universal 執行檔保留執行權限" || fail "macOS universal 執行檔不可執行"
for f in "SIMCITY 1.10/DATA/MESSAGE.PTF" music/SC2000-10004.ogg 完整版權利邊界.txt; do
  [ -s "$TMP/mac-full/$f" ] && pass "macOS 完整版有 $f" || fail "macOS 完整版少了 $f"
done

for f in chengshi LICENSE NotoSansCJK-copyright.txt 讀我.txt 素材與權利.txt; do
  [ -s "$TMP/release/$f" ] && pass "公開版有 $f" || fail "公開版少了 $f"
done

for pair in "public:$APP_REL" "full:$APP_FULL"; do
  kind=${pair%%:*}
  app=${pair#*:}
  if [ -n "$app" ] && [ -x "$app" ]; then
    pass "$kind AppImage 可執行"
    out="$TMP/appimage-$kind"
    mkdir -p "$out"
    (cd "$out" && "$app" --appimage-extract >/dev/null)
    [ -x "$out/squashfs-root/AppRun" ] \
      && pass "$kind AppImage 可解包且有 AppRun" || fail "$kind AppImage 結構不完整"
  else
    fail "$kind AppImage 不可執行"
  fi
done
if [ -d "$TMP/appimage-public/squashfs-root" ]; then
  if find "$TMP/appimage-public/squashfs-root" -type f \( \
      -iname '*.pgf' -o -iname '*.ppf' -o -iname '*.psn' -o -iname '*.ptf' -o \
      -iname '*.psf' -o -iname '*.cty' -o -iname '*.v4' -o -iname '*.ogg' -o \
      -iname '*.wav' -o -iname '*.xmi' -o -iname '*.mid' -o -iname 'SIMCITY.EXE' \) \
      -print -quit | grep -q .; then
    fail "公開 AppImage 混入原版資料或音樂"
  else
    pass "公開 AppImage 沒有原版資料與音樂"
  fi
fi
for f in "SIMCITY 1.10/DATA/MESSAGE.PTF" music/SC2000-10004.ogg 完整版權利邊界.txt; do
  [ -s "$TMP/appimage-full/squashfs-root/$f" ] \
    && pass "完整版 AppImage 有 $f" || fail "完整版 AppImage 少了 $f"
done

# 公開包的解包內容再掃一次；打包器內的掃描不能替代獨立驗證。
if find "$TMP/release" -type f \( \
    -iname '*.pgf' -o -iname '*.ppf' -o -iname '*.psn' -o -iname '*.ptf' -o \
    -iname '*.psf' -o -iname '*.cty' -o -iname '*.v4' -o -iname '*.ogg' -o \
    -iname '*.wav' -o -iname '*.xmi' -o -iname '*.mid' -o -iname 'SIMCITY.EXE' -o \
    -iname 'SIMCITY.CFG' -o -iname 'SETTINGS.EXE' \) -print -quit | grep -q .; then
  fail "公開版混入原版資料或音樂"
else
  pass "公開版沒有原版資料與音樂"
fi

for f in "SIMCITY 1.10/DATA/MESSAGE.PTF" music/SC2000-10003.ogg 完整版權利邊界.txt; do
  [ -s "$TMP/full/$f" ] && pass "完整版有 $f" || fail "完整版少了 $f"
done
[ -x "$TMP/full/啟動城市.sh" ] && pass "完整版啟動器可執行" || fail "完整版啟動器不可執行"

Xvfb :99 -screen 0 1920x1080x24 -nolisten tcp >/tmp/chengshi-package-xvfb.log 2>&1 &
XVFB_PID=$!
export DISPLAY=:99
for _ in $(seq 1 40); do
  xdpyinfo >/dev/null 2>&1 && break
  sleep 0.25
done
xdpyinfo >/dev/null 2>&1 || fail "Xvfb 未就緒"

# Ebiten 在 main 前就初始化 GLFW，所以連純 `-version` 也需要可用 DISPLAY。
if [ -x "$TMP/release/chengshi" ]; then
  GOT=$("$TMP/release/chengshi" -version 2>/dev/null || true)
  case "$GOT" in
    *"$VER"*) pass "執行檔版本為 $GOT" ;;
    *) fail "執行檔版本不含 $VER：$GOT" ;;
  esac
fi

run_smoke() {
  local name=$1 dir=$2 data=$3
  local cfg="$TMP/config-$name" save="$TMP/save-$name/city.cty" log="$TMP/$name.log"
  mkdir -p "$cfg" "$(dirname "$save")"
  (
    cd "$dir"
    XDG_CONFIG_HOME="$cfg" ./chengshi -data "$data" -save "$save" -mute -seed 7 >"$log" 2>&1 &
    echo $! >"$TMP/$name.pid"
  )
  local pid
  pid=$(cat "$TMP/$name.pid")
  local ok=0 wid=
  for _ in $(seq 1 60); do
    wid=$(xdotool search --pid "$pid" --name '城市' 2>/dev/null | head -n 1 || true)
    if [ -n "$wid" ]; then ok=1; break; fi
    kill -0 "$pid" 2>/dev/null || break
    sleep 0.5
  done
  if [ "$ok" -eq 1 ]; then
    pass "$name 可正常啟動"
    xdotool windowfocus "$wid" 2>/dev/null || true
    for _ in $(seq 1 8); do
      xdotool key --clearmodifiers ctrl+s
      for _ in $(seq 1 4); do [ -s "$save" ] && break 2; sleep 0.25; done
    done
    [ "$(stat -c%s "$save" 2>/dev/null || printf 0)" = 27120 ] \
      && pass "$name 可寫 27120-byte 存檔" || fail "$name 存檔失敗"
  else
    fail "$name 無法啟動"
  fi
  kill "$pid" 2>/dev/null || true
  wait "$pid" 2>/dev/null || true
  for _ in $(seq 1 40); do
    xdotool search --pid "$pid" --name '城市' >/dev/null 2>&1 || break
    sleep 0.1
  done
  grep -Eqi 'panic|runtime error' "$log" && fail "$name log 有 panic" || pass "$name 沒有 panic"
}

if [ -x "$TMP/release/chengshi" ]; then
  run_smoke release "$TMP/release" "$ROOT/workplace/dos110/SIMCITY 1.10"
fi
if [ -x "$TMP/full/chengshi" ]; then
  run_smoke full "$TMP/full" "$TMP/full/SIMCITY 1.10"
fi

# 最後用完整版 AppImage 本身走正常入口：無 -seed／-scenario／-window，必須先到
# 招牌選單；再以真滑鼠進劇本、進城市、開 OPTIONS 與點速度細項。
if [ -n "$APP_FULL" ] && [ -x "$APP_FULL" ]; then
  if APPIMAGE="$APP_FULL" RECEIPT_DIR="$RECEIPT" \
      bash "$ROOT/tools/appimage_player_path_inner.sh"; then
    pass "完整版 AppImage 正常玩家入口與滑鼠速度選單"
  else
    fail "完整版 AppImage 正常玩家入口或滑鼠速度選單失敗"
  fi
fi

# 完整版不得只證明「有 OGG」：把預設 ALSA 裝置導向 float32 raw 檔，
# 實際確認封包內執行檔與封包內 music/ 送出非零樣本。
if [ -x "$TMP/full/chengshi" ]; then
  mkdir -p "$HOME"
  cat >"$HOME/.asoundrc" <<'EOF'
pcm.!default {
 type file
 file "/cap/full-music.raw"
 format raw
 slave { pcm "nullpcm" }
}
pcm.nullpcm { type null }
EOF
  (
    cd "$TMP/full"
    ./chengshi -data "SIMCITY 1.10" -music music -seed 9 -scale 1 \
      >"$TMP/full-music.log" 2>&1 &
    echo $! >"$TMP/full-music.pid"
  )
  MPID=$(cat "$TMP/full-music.pid")
  for _ in $(seq 1 120); do [ -s /cap/full-music.raw ] && break; sleep 0.05; done
  sleep 0.1
  kill "$MPID" 2>/dev/null || true
  if python3 - <<'PY'
import os
import struct

path = '/cap/full-music.raw'
if not os.path.isfile(path) or os.path.getsize(path) == 0:
    raise SystemExit('完整版沒有產生音訊擷取')
with open(path, 'rb') as fh:
    raw = fh.read()
samples = struct.iter_unpack('<f', raw[:len(raw) // 4 * 4])
if not any(value[0] != 0.0 for value in samples):
    raise SystemExit('完整版音訊輸出全部是靜音')
print(f'pass  完整版內建 OGG 送出非零音訊（{len(raw)} bytes）')
PY
  then
    :
  else
    fail "完整版音訊驗證失敗"
  fi
fi

echo
if [ "$FAIL" -eq 0 ]; then
  echo '== dist-all 驗證通過 =='
else
  echo '== dist-all 驗證失敗 =='
  exit 1
fi
