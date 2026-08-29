#!/usr/bin/env bash
# 在容器裡跑的發行包驗證本體。不要直接執行，用 tools/verify_release.sh。
#
# 驗的是**發行包本身**，不是 build 出來的執行檔：解到一個乾淨目錄，從那個
# 目錄執行，資料放在別處，家目錄另外給一個。玩家踩得到而 go test 踩不到的
# 問題都在這一段——存檔寫進唯讀的 cwd、包裡漏帶授權條款、少了 -data 時
# 給的是 panic 而不是看得懂的話。
set -uo pipefail

FAIL=0
fail() { echo "FAIL  $*"; FAIL=1; }
pass() { echo "pass  $*"; }

Xvfb :99 -screen 0 1280x960x24 -nolisten tcp >/tmp/xvfb.log 2>&1 &
export DISPLAY=:99
for i in $(seq 1 40); do xdpyinfo >/dev/null 2>&1 && break; sleep 0.25; done

W=/tmp/pkg; rm -rf $W; mkdir -p $W
tar -xzf dist/chengshi_cht-*-linux-amd64.tar.gz -C $W

for f in chengshi LICENSE NotoSansCJK-copyright.txt 讀我.txt; do
  [ -s "$W/$f" ] && pass "包裡有 $f" || fail "包裡少了 $f"
done
# 原版素材一個位元組都不該進來
if find "$W" -iname "*.pgf" -o -iname "*.ptf" -o -iname "*.psn" -o -iname "*.cty" \
     -o -iname "*.exe" | grep -q .; then
  fail "包裡混進了原版素材或原版執行檔"
else
  pass "包裡沒有原版素材"
fi

# macOS 的包在 Linux 上執行不了，只能驗結構。
M=/tmp/mac
if ls dist/chengshi_cht-*-macos-universal.tar.gz >/dev/null 2>&1; then
  rm -rf $M; mkdir -p $M
  tar -xzf dist/chengshi_cht-*-macos-universal.tar.gz -C $M
  for f in "城市.app/Contents/MacOS/chengshi" "城市.app/Contents/Info.plist" \
           LICENSE NotoSansCJK-copyright.txt 讀我.txt; do
    [ -s "$M/$f" ] && pass "macOS 包裡有 $f" || fail "macOS 包裡少了 $f"
  done
  [ -x "$M/城市.app/Contents/MacOS/chengshi" ] \
    && pass "macOS 執行檔保留了執行權限" || fail "macOS 執行檔沒有執行權限（tar 壓法不對）"
  if find "$M" -iname "*.pgf" -o -iname "*.ptf" -o -iname "*.psn" -o -iname "*.cty" | grep -q .; then
    fail "macOS 包裡混進了原版素材"
  else
    pass "macOS 包裡沒有原版素材"
  fi
else
  echo "      （沒有 macOS 包，跳過）"
fi

V=$("$W/chengshi" -version 2>/dev/null)
[ -n "$V" ] && pass "版本：$V" || fail "-version 印不出東西"

# 少了 -data 要給看得懂的話，而且退碼要是 2（用法錯誤），不是崩潰。
MSG=$("$W/chengshi" 2>&1 >/dev/null); RC=$?
echo "$MSG" | grep -q "請用 -data" && pass "少了 -data 有中文說明" \
  || fail "少了 -data 的訊息不對：$(echo "$MSG" | grep -v '^XGB:' | head -1)"
[ "$RC" = 2 ] && pass "少了 -data 的退碼是 2" || fail "少了 -data 的退碼是 $RC"

# 玩家的情境：從包所在的目錄執行，資料在別的地方，家目錄可寫。
export HOME=/tmp/player; rm -rf $HOME; mkdir -p $HOME
SAVE=$HOME/.local/share/chengshi/city.cty
cd $W
./chengshi -data "/src/workplace/dos110/SIMCITY 1.10" -seed 7 >/tmp/pkg.log 2>&1 &
G=$!
ok=0
for i in $(seq 1 90); do
  xdotool search --name "城市" >/dev/null 2>&1 && { ok=1; break; }
  kill -0 $G 2>/dev/null || break
  sleep 1
done
if [ "$ok" = 1 ]; then
  pass "從發行包所在目錄跑得起來"
  sleep 3
  for i in $(seq 1 8); do xdotool key --clearmodifiers ctrl+s; sleep 1; [ -s "$SAVE" ] && break; done
  kill $G 2>/dev/null || true; wait $G 2>/dev/null || true
  if [ -s "$SAVE" ]; then
    n=$(stat -c%s "$SAVE")
    [ "$n" = 27120 ] && pass "Ctrl-S 存到 $SAVE（$n 位元組）" \
                     || fail "存檔大小 $n，應為 27120"
  else
    fail "Ctrl-S 沒有存到 $SAVE —— 預設存檔路徑不可寫？"
  fi
else
  fail "發行包跑不起來"
  kill $G 2>/dev/null || true
fi
grep -vi "^XGB:" /tmp/pkg.log | grep -qi "panic\|runtime error" \
  && { fail "log 裡有 panic"; grep -vi "^XGB:" /tmp/pkg.log | head -10; } \
  || pass "沒有 panic"

echo
[ "$FAIL" = 0 ] && echo "== 發行包驗證通過 ==" || { echo "== 發行包驗證失敗 =="; exit 1; }
