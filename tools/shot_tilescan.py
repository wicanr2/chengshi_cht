#!/usr/bin/env python3
"""逐格比對截圖的地圖區與 `.PGF` 第 0 庫的 960 張圖塊。

**這是版面對拍最強的判準**：它問的是「畫面上這一格有沒有逐位元等於原版美術」，
不是「看起來像不像」。差一階顏色、差兩像素位置、把色號 0 當透明——三種錯都
沒有症狀（編得過、測得過、玩得動、目視也看不出來），三種都被這個判準抓到過。

它也對模擬進度免疫：城市長成什麼樣不影響「每一格是不是某一張圖塊」。
逐點比對做不到這件事（見 `tools/shot_diff.py` 的警告）。

內建正對照：3×3 工具游標的外框一定會被報成 8 格不明（中心格不會）。
所以「不明 0 格」反而可疑——多半是根本沒掃到地圖。

實測基準（Wild West 圖形集）：

    原版 DOSBox          512 格中 504 格命中，不明 8 格 = 工具游標外框
    remake（修正前）      176 格中  88 格命中
    remake（修正後）      176 格中 163 格命中

remake 剩下的 13 格是：底列被工具帶蓋到（工具帶比原版高三像素，
`docs/spec/ui-layout.md` 的已知偏差）與黃色的工具游標框。

用法：

    # 先產圖集（風格要跟截圖一致）
    tools/go.sh run ./tools/tileatlas "<...>/CEGA/WESTCEGA.PGF" workplace/tiles-west.png

    # 原版 DOSBox 截圖（1024×768，遊戲區在 192,184）
    tools/shot_tilescan.py --origin 256,239 workplace/dosbox/x-03.png

    # remake 截圖（3 倍畫布）
    tools/shot_tilescan.py --scale 3 --origin 64,55 --max-x 241 --min-hit 150 \\
        workplace/shots/parity.png

`--min-hit` 沒達到就以退出碼 1 結束，給試玩腳本當關卡用。
完整的方法論見 `docs/re/16-dos-oracle.md` §九。
"""
import argparse
import os
import sys

from PIL import Image

# EGA 的四階。`.PGF` 存的是 0/80/160/240（六位元 VGA 值乘 4 的近似），
# 螢幕上顯示的是 0/85/170/255。兩邊都正規化之後才比得了。
def norm(im):
    px = im.load()
    for y in range(im.size[1]):
        for x in range(im.size[0]):
            r, g, b = px[x, y]
            px[x, y] = (round(r / 85) * 85, round(g / 85) * 85, round(b / 85) * 85)
    return im


def load_atlas(path, size, cols):
    at = norm(Image.open(path).convert("RGB"))
    tiles = {}
    n = (at.size[0] // size) * (at.size[1] // size)
    for k in range(n):
        ox, oy = (k % cols) * size, (k // cols) * size
        tiles.setdefault(at.crop((ox, oy, ox + size, oy + size)).tobytes(), k)
    return tiles


def scan(path, tiles, args):
    im = Image.open(path).convert("RGB")
    if args.scale != 1:
        im = im.resize((im.size[0] // args.scale, im.size[1] // args.scale),
                       Image.NEAREST)
    im = norm(im)
    ox, oy = args.origin
    s = args.size
    ok, bad = 0, []
    for j in range(args.rows):
        for i in range(args.cols):
            x, y = ox + s * i, oy + s * j
            if x + s > min(args.max_x, im.size[0]) or y + s > min(args.max_y, im.size[1]):
                continue
            if im.crop((x, y, x + s, y + s)).tobytes() in tiles:
                ok += 1
            else:
                bad.append((i, j))
    return ok, bad


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("shots", nargs="+")
    ap.add_argument("--atlas", default="workplace/tiles-west.png")
    ap.add_argument("--origin", default="256,239",
                    help="第一格的左上角螢幕座標（縮放之後的座標）")
    ap.add_argument("--scale", type=int, default=1, help="截圖先縮小這個倍率")
    ap.add_argument("--size", type=int, default=16, help="圖塊邊長")
    ap.add_argument("--cols", type=int, default=32)
    ap.add_argument("--rows", type=int, default=16)
    ap.add_argument("--atlas-cols", type=int, default=30)
    ap.add_argument("--max-x", type=int, default=10 ** 6,
                    help="超過這個 x 的格子不算（被別的視窗蓋住）")
    ap.add_argument("--max-y", type=int, default=10 ** 6)
    ap.add_argument("--min-hit", type=int, default=0,
                    help="命中數低於此值就以退出碼 1 結束")
    a = ap.parse_args()
    a.origin = tuple(int(v) for v in a.origin.split(","))

    if not os.path.exists(a.atlas):
        print(f"找不到圖集 {a.atlas}——先跑 tools/go.sh run ./tools/tileatlas", file=sys.stderr)
        return 2
    tiles = load_atlas(a.atlas, a.size, a.atlas_cols)

    rc = 0
    for p in a.shots:
        ok, bad = scan(p, tiles, a)
        tag = ""
        if a.min_hit:
            tag = "  pass" if ok >= a.min_hit else f"  FAIL（門檻 {a.min_hit}）"
            if ok < a.min_hit:
                rc = 1
        print(f"{os.path.basename(p):28s} 命中 {ok:4d} 格，不明 {len(bad):3d} 格 "
              f"{bad[:12]}{tag}")
    return rc


if __name__ == "__main__":
    sys.exit(main())
