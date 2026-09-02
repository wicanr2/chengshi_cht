#!/usr/bin/env bash
# tools/package_all.sh 的容器內本體；不要在主機直接執行。
set -euo pipefail

: "${VER:?缺少 VER}"
ROOT=/src
DEST="$ROOT/dist-all/$VER"
STAGE="$ROOT/workplace/package-all-$VER"
DATA="$ROOT/workplace/dos110/SIMCITY 1.10"
MUSIC="$ROOT/music"
E220="$ROOT/workplace/e220"
MAC_BUILD="$ROOT/workplace/package-mac-$VER"

cleanup() { rm -rf "$STAGE"; }
trap cleanup EXIT
rm -rf "$STAGE"
mkdir -p "$STAGE/release-linux" "$STAGE/release-windows"
mkdir -p "$STAGE/full-linux" "$STAGE/full-windows"
mkdir -p "$STAGE/release-mac" "$STAGE/full-mac"
mkdir -p "$STAGE/appimage-public" "$STAGE/appimage-full"
mkdir -p "$DEST/release" "$DEST/full" "$DEST/promo"

LD="-s -w -X main.version=$VER"
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
  go build -ldflags "$LD" -o "$STAGE/release-linux/chengshi" ./cmd/chengshi
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
  go build -ldflags "$LD" -o "$STAGE/release-windows/chengshi.exe" ./cmd/chengshi

install_public_docs() {
  local d=$1
  cp "$ROOT/LICENSE" "$d/LICENSE"
  cp "$ROOT/licenses/NotoSansCJK-copyright.txt" "$d/"
  cp "$ROOT/packaging/讀我.txt" "$d/"
  cat >"$d/素材與權利.txt" <<'EOF'
本包只含城市 remake 引擎、內嵌翻譯及依法隨附的授權文件。

本包不含 SimCity 原版 EXE、資料、美術、音效、劇本、手冊或任何版本的配樂。
玩家必須自行合法取得 SimCity 1.10（DOS）資料。若要播放背景音樂，請把自己
有權使用的 .ogg／.wav 放入 music/，或用 -music 指定目錄。

cities/ 裡的五個地圖檔是例外，它們不是原版素材：台北、台中、台南三張由本專案
繪製，台灣與高雄兩張出自軟體世界 1990 年的發行，權利狀態見 LICENSE 附註第五條。
地圖只有地形資料，不含任何 Maxis 的程式或美術。

程式碼授權見 LICENSE；字型來源授權見 NotoSansCJK-copyright.txt。
EOF
}

# cities/ 是本儲存庫公開收錄的地圖：三張本專案畫的，兩張軟體世界 1990 年的
# （權利狀態見 LICENSE 附註第五條）。repo 裡是公開的，發行包就跟著帶。
install_cities() {
  local d=$1
  mkdir -p "$d/cities"
  cp "$ROOT/cities/README.md" "$d/cities/"
  cp "$ROOT"/cities/*.CTY "$d/cities/"
}

install_public_docs "$STAGE/release-linux"
install_public_docs "$STAGE/release-windows"
install_cities "$STAGE/release-linux"
install_cities "$STAGE/release-windows"
if [ ! -x "$MAC_BUILD/城市.app/Contents/MacOS/chengshi" ]; then
  echo "缺少本版本的 macOS universal 輸入：$MAC_BUILD/城市.app" >&2
  exit 2
fi
cp -a "$MAC_BUILD/城市.app" "$STAGE/release-mac/"
install_public_docs "$STAGE/release-mac"
install_cities "$STAGE/release-mac"

# install_e220 把軟體世界 1990 年地形編輯器磁片上的 Maxis 素材放進**完整版**：
# 六種顯示模式的編輯器美術，以及 Maxis 自己的九座示範城市。
#
# ⚠ **這些只能留在本機。** 它們是 Maxis 的內容，與 cities/ 那五張不同
# （後者是軟體世界自己畫的與本專案畫的，見 LICENSE 附註第五條）。
# 公開包的拒絕清單會擋住它們，這裡刻意只寫進 full-*。
install_e220() {
  local d=$1
  [ -d "$E220" ] || return 0
  mkdir -p "$d/地形編輯器素材" "$d/cities"
  cp "$E220"/*.PGF "$E220"/*.PPF "$d/地形編輯器素材/" 2>/dev/null || true
  # 示範城市：磁片上 Maxis 自己那九座。軟體世界的兩張已經在 cities/ 裡了。
  for f in "$E220"/*.CTY; do
    case "$(basename "$f")" in
      TAIWAN.CTY | KAOHSIUN.CTY) continue ;;
    esac
    cp "$f" "$d/cities/"
  done
  cat >"$d/地形編輯器素材/來源.txt" <<'TXT'
本目錄與 cities/ 裡的示範城市取自軟體世界研究中心（高雄）1990 年重新打包的
Maxis《SimCity Terrain Editor》磁片。這些是 Maxis 的美術與城市檔，只能留在
本機或明確授權的私有位置，不得公開散布。

cities/TAIWAN.CTY 與 cities/KAOHSIUN.CTY 不在此限——那兩張是軟體世界自己畫的，
已隨本專案公開，權利狀態見 LICENSE 附註第五條。

美術是地形編輯器自己的畫面，本重製版沒有地形編輯器，所以它們目前只作保存用途。
TXT
}

# 完整版以公開 stage 為底，再加入本機素材。執行檔會自動尋找相鄰的
# SIMCITY 1.10；啟動器固定 cwd，確保相鄰 music/ 也能被找到。
cp -a "$STAGE/release-linux/." "$STAGE/full-linux/"
cp -a "$STAGE/release-windows/." "$STAGE/full-windows/"
cp -a "$STAGE/release-mac/." "$STAGE/full-mac/"
for d in "$STAGE/full-linux" "$STAGE/full-windows" "$STAGE/full-mac"; do
  cp -a "$DATA" "$d/SIMCITY 1.10"
  cp -a "$MUSIC" "$d/music"
  install_e220 "$d"
  rm -f "$d/素材與權利.txt"
  cat >"$d/讀我.txt" <<'EOF'
城市（chengshi_cht）本機完整版

本目錄已包含本機合法來源的 SimCity 1.10 資料與 SimCity 2000 OGG，不必再指定
-data 或 -music。Linux 可執行「啟動城市.sh」或完整版 `.AppImage`；Windows 執行
「啟動城市.bat」或 chengshi.exe；macOS 開啟「城市.app」。不帶參數啟動會先進
招牌遊戲選單，再由滑鼠選新城市、讀檔或悲情城市。

遊戲存檔與語言設定仍寫入使用者資料／設定目錄，不會改寫隨附的原版資料。
這份完整版不得公開散布；詳細界線見「完整版權利邊界.txt」。程式碼與字型授權
分別見 LICENSE 與 NotoSansCJK-copyright.txt。
EOF
  cat >"$d/完整版權利邊界.txt" <<'EOF'
這是依使用者自備合法來源組成的本機完整版，內含仍受保護的 SimCity 原版資料
與 SimCity 2000 衍生 OGG。只能留在本機或明確授權的私有位置；不得提交 Git、
上傳 GitHub Release、公開雲端或再次散布。
EOF
done
cat >"$STAGE/full-linux/啟動城市.sh" <<'EOF'
#!/usr/bin/env sh
set -eu
HERE=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
cd "$HERE"
exec ./chengshi "$@"
EOF
chmod 0755 "$STAGE/full-linux/啟動城市.sh"
cat >"$STAGE/full-windows/啟動城市.bat" <<'EOF'
@echo off
cd /d "%~dp0"
chengshi.exe %*
EOF

build_appimage() {
  local source_dir=$1 appdir=$2 output=$3 mode=$4
  rm -rf "$appdir"
  mkdir -p "$appdir/usr/bin"
  cp "$source_dir/chengshi" "$appdir/usr/bin/chengshi"
  chmod 0755 "$appdir/usr/bin/chengshi"
  cp "$ROOT/packaging/chengshi.desktop" "$appdir/chengshi.desktop"
  cp "$ROOT/packaging/chengshi.svg" "$appdir/chengshi.svg"
  cp "$source_dir/LICENSE" "$source_dir/NotoSansCJK-copyright.txt" \
     "$source_dir/讀我.txt" "$appdir/"
  # 地圖要跟著進 AppImage。少了這一段，tar.gz 的玩家拿得到五張地圖而
  # AppImage 的玩家拿不到——而 AppImage 是 Linux 這邊的主要交付物。
  cp -a "$source_dir/cities" "$appdir/cities"
  if [ "$mode" = full ]; then
    cp -a "$source_dir/SIMCITY 1.10" "$appdir/SIMCITY 1.10"
    cp -a "$source_dir/music" "$appdir/music"
    cp "$source_dir/完整版權利邊界.txt" "$appdir/"
    [ -d "$source_dir/地形編輯器素材" ] \
      && cp -a "$source_dir/地形編輯器素材" "$appdir/地形編輯器素材"
  else
    cp "$source_dir/素材與權利.txt" "$appdir/"
  fi

  /opt/appimage-tools/linuxdeploy.AppImage --appimage-extract-and-run \
    --appdir "$appdir" \
    --executable "$appdir/usr/bin/chengshi" \
    --desktop-file "$appdir/chengshi.desktop" \
    --icon-file "$appdir/chengshi.svg"

  # linuxdeploy 先建立 AppRun → usr/bin/chengshi 的 symlink；必須先移除再寫
  # wrapper。直接用重導向覆寫 symlink 會把真正的 Go 執行檔改寫成 shell，
  # 產生結構看似完整、實際 Permission denied 的假 AppImage。
  rm -f "$appdir/AppRun"
  if [ "$mode" = full ]; then
    cat >"$appdir/AppRun" <<'EOF'
#!/usr/bin/env sh
set -eu
APPDIR=${APPDIR:-$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)}
exec "$APPDIR/usr/bin/chengshi" -data "$APPDIR/SIMCITY 1.10" -music "$APPDIR/music" "$@"
EOF
  else
    cat >"$appdir/AppRun" <<'EOF'
#!/usr/bin/env sh
set -eu
APPDIR=${APPDIR:-$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)}
exec "$APPDIR/usr/bin/chengshi" "$@"
EOF
  fi
  chmod 0755 "$appdir/AppRun"
  ARCH=x86_64 /opt/appimage-tools/appimagetool.AppImage --appimage-extract-and-run \
    --runtime-file /opt/appimage-tools/runtime-x86_64 "$appdir" "$output"
  chmod 0755 "$output"
}

build_appimage "$STAGE/release-linux" "$STAGE/appimage-public" \
  "$DEST/release/chengshi-$VER-linux-amd64.AppImage" public
build_appimage "$STAGE/full-linux" "$STAGE/appimage-full" \
  "$DEST/full/chengshi-$VER-full-local-linux-amd64.AppImage" full

tar -C "$STAGE/release-linux" -czf \
  "$DEST/release/chengshi-$VER-linux-amd64.tar.gz" .
tar -C "$STAGE/full-linux" -czf \
  "$DEST/full/chengshi-$VER-full-local-linux-amd64.tar.gz" .
tar -C "$STAGE/release-mac" -czf \
  "$DEST/release/chengshi-$VER-macos-universal.tar.gz" .
tar -C "$STAGE/full-mac" -czf \
  "$DEST/full/chengshi-$VER-full-local-macos-universal.tar.gz" .

python3 - "$STAGE" "$DEST" "$VER" <<'PY'
import os
import sys
import zipfile

stage, dest, ver = sys.argv[1:]
for src, out in (
    ("release-windows", f"release/chengshi-{ver}-windows-amd64.zip"),
    ("full-windows", f"full/chengshi-{ver}-full-local-windows-amd64.zip"),
):
    root = os.path.join(stage, src)
    with zipfile.ZipFile(os.path.join(dest, out), "w", zipfile.ZIP_DEFLATED) as zf:
        for base, dirs, files in os.walk(root):
            dirs.sort()
            for name in sorted(files):
                path = os.path.join(base, name)
                zf.write(path, os.path.relpath(path, root))
PY

# promo 保存目前 README 使用的畫面與可追溯說明；不把本機 OGG 複製進去。
mkdir -p "$DEST/promo/screenshots"
cp "$ROOT"/docs/images/*.png "$DEST/promo/screenshots/"
# 推廣影片：五張台灣地圖的運鏡。由 tools/promo_video.sh 產生（不進 Git）。
PROMO_VIDEO=0
for f in "$ROOT"/workplace/promo/promo.mp4 "$ROOT"/workplace/promo/promo.gif; do
  [ -s "$f" ] && { cp "$f" "$DEST/promo/"; PROMO_VIDEO=1; }
done
BUILD_HEAD=$(git rev-parse HEAD 2>/dev/null || printf unknown)
cat >"$DEST/promo/README.md" <<EOF
# 城市 $VER 推廣素材

- 來源：本次工作樹 $BUILD_HEAD
- 畫面：README 目前引用的 remake 截圖；圖中原版美術只作專案說明。
- 音訊：本目錄不含 SimCity 或 SimCity 2000 音樂。
- 影片：$( [ "$PROMO_VIDEO" = 1 ] \
    && echo "promo.mp4 與 promo.gif ＝ 五張台灣地圖（台灣／台北／台中／台南／高雄）的運鏡，由 tools/promo_video.sh 產生。畫面上的圖塊來自玩家自備的原版。" \
    || echo "本版尚未建立推廣影片；不得把本目錄冒稱為影片驗收完成。" )
EOF

# 公開包拒絕清單：檔名與內容兩層。exe 是 Windows remake 本體，允許唯一的
# chengshi.exe；其餘原版格式、音訊及原版識別檔一律拒絕。
python3 - "$DEST/release" "$ROOT/cities" <<'PY'
import os
import sys
import tarfile
import zipfile

root, cities_dir = sys.argv[1], sys.argv[2]
bad_ext = {'.pgf', '.ppf', '.psn', '.ptf', '.psf', '.cty', '.v4', '.ogg', '.wav', '.xmi', '.mid'}
bad_names = {'simcity.exe', 'simcity.cfg', 'settings.exe', 'sounddat.v4', 'read.me'}

# 唯一的例外：cities/ 底下、而且**本儲存庫的 cities/ 真的有同名檔**的城市檔。
# 例外綁在「已經公開的東西」上，不是綁在副檔名或目錄名上——這樣就算有人
# 把 Maxis 的示範城市丟進 cities/，只要它沒進 repo 就照樣擋下來。
allowed_cities = {f.lower() for f in os.listdir(cities_dir)} if os.path.isdir(cities_dir) else set()

def check(names, archive):
    for raw in names:
        path = raw.replace('\\', '/')
        name = path.rsplit('/', 1)[-1].lower()
        ext = os.path.splitext(name)[1]
        if ext == '.cty' and '/cities/' in '/' + path and name in allowed_cities:
            continue
        if ext in bad_ext or name in bad_names:
            raise SystemExit(f'公開包混入受保護素材：{archive}: {raw}')

for name in sorted(os.listdir(root)):
    path = os.path.join(root, name)
    if name.endswith('.zip'):
        with zipfile.ZipFile(path) as zf:
            check(zf.namelist(), name)
    elif name.endswith('.tar.gz'):
        with tarfile.open(path, 'r:gz') as tf:
            check(tf.getnames(), name)
PY

# AppImage 也是公開封包，不能只掃 tar／zip。先用 AppImage 自己的唯讀解包入口
# 展開，再以同一份拒絕條件檢查內容。
for app in "$DEST/release"/*.AppImage; do
  scan="$STAGE/appimage-scan"
  rm -rf "$scan"
  mkdir -p "$scan"
  (cd "$scan" && "$app" --appimage-extract >/dev/null)
  # cities/ 是唯一例外，與 tar／zip 的掃描同一個理由；下面另外逐檔核對
  # 它只含本儲存庫收錄的地圖。
  for c in "$scan/squashfs-root"/cities/*.CTY; do
    [ -e "$c" ] || continue
    [ -f "$ROOT/cities/$(basename "$c")" ] || {
      echo "公開 AppImage 的 cities/ 多了 $(basename "$c")：$app" >&2
      exit 1
    }
  done
  if find "$scan/squashfs-root" -type f -not -path '*/cities/*' \( \
      -iname '*.pgf' -o -iname '*.ppf' -o -iname '*.psn' -o -iname '*.ptf' -o \
      -iname '*.psf' -o -iname '*.cty' -o -iname '*.v4' -o -iname '*.ogg' -o \
      -iname '*.wav' -o -iname '*.xmi' -o -iname '*.mid' -o -iname 'SIMCITY.EXE' -o \
      -iname 'SIMCITY.CFG' -o -iname 'SETTINGS.EXE' \) -print -quit | grep -q .; then
    echo "公開 AppImage 混入受保護素材：$app" >&2
    exit 1
  fi
done

python3 - "$DEST/release" "$VER" <<'PY'
import hashlib
import json
import os
import sys

root, ver = sys.argv[1:]
files = []
for name in sorted(os.listdir(root)):
    path = os.path.join(root, name)
    if not os.path.isfile(path):
        continue
    h = hashlib.sha256()
    with open(path, 'rb') as fh:
        for block in iter(lambda: fh.read(1024 * 1024), b''):
            h.update(block)
    files.append({'name': name, 'bytes': os.path.getsize(path), 'sha256': h.hexdigest()})
with open(os.path.join(root, 'MANIFEST.json'), 'w', encoding='utf-8') as fh:
    json.dump({'version': ver, 'rights': 'public-reviewed', 'packages': files}, fh,
              ensure_ascii=False, indent=2)
    fh.write('\n')
PY

for area in release full promo; do
  (
    cd "$DEST/$area"
    find . -type f ! -name SHA256SUMS -print0 | sort -z | xargs -0 sha256sum >SHA256SUMS
  )
done

python3 - "$DEST" "$VER" <<'PY'
import hashlib
import json
import os
import platform
import sys

root, ver = sys.argv[1:]
items = []
for area in ('release', 'full', 'promo'):
    for base, dirs, files in os.walk(os.path.join(root, area)):
        dirs.sort()
        for name in sorted(files):
            path = os.path.join(base, name)
            h = hashlib.sha256()
            with open(path, 'rb') as fh:
                for block in iter(lambda: fh.read(1024 * 1024), b''):
                    h.update(block)
            items.append({
                'path': os.path.relpath(path, root),
                'bytes': os.path.getsize(path),
                'sha256': h.hexdigest(),
                'rights': 'local-private' if area == 'full' else 'public-reviewed',
            })
with open(os.path.join(root, 'MANIFEST.json'), 'w', encoding='utf-8') as fh:
    json.dump({'version': ver, 'builder': platform.platform(), 'files': items}, fh,
              ensure_ascii=False, indent=2)
    fh.write('\n')
PY

rm -rf "$MAC_BUILD"
echo "dist-all/$VER 已建立（Linux／Windows／macOS universal）"
