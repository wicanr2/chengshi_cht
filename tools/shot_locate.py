#!/usr/bin/env python3
"""從一張截圖反推「編輯視窗的鏡頭停在地圖的哪一格」。

作法：把畫面上每個 16×16 格解回圖塊編號（比對 `.PGF` 第 0 庫的圖集），
再在城市地圖上滑動找最吻合的位置。**只用有辨識力的圖塊**——泥土與大片
水域到處都是，全部拿來比會得到一堆分數接近的候選，看起來像有答案其實沒有。

它同時是「remake 的鏡頭跟原版一不一樣」的判準。原版的鏡頭沒有存進城市檔
（`docs/spec/ui-layout.md`：`MiscHis` 裡沒有任何視窗欄位），所以只能從畫面反推。

用法：
    tools/shot_locate.py --grid workplace/boston-orig.txt \\
        --origin 256,239 workplace/dosbox/cam-00-view.png
    tools/shot_locate.py --grid workplace/boston-orig.txt --scale 3 \\
        --origin 64,55 --max-x 241 workplace/shots/x.png
"""
import argparse
import os
import sys
from collections import Counter

from PIL import Image


def norm(im):
    px = im.load()
    for y in range(im.size[1]):
        for x in range(im.size[0]):
            r, g, b = px[x, y]
            px[x, y] = (round(r / 85) * 85, round(g / 85) * 85, round(b / 85) * 85)
    return im


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("shots", nargs="+")
    ap.add_argument("--grid", required=True, help="每行一列圖塊編號的地圖")
    ap.add_argument("--atlas", default="workplace/tiles-west.png")
    ap.add_argument("--atlas-cols", type=int, default=30)
    ap.add_argument("--origin", default="256,239")
    ap.add_argument("--scale", type=int, default=1)
    ap.add_argument("--size", type=int, default=16)
    ap.add_argument("--cols", type=int, default=32)
    ap.add_argument("--rows", type=int, default=16)
    ap.add_argument("--max-x", type=int, default=10 ** 6)
    ap.add_argument("--rare", type=int, default=400,
                    help="全圖出現超過這麼多次的圖塊沒有辨識力，不採用")
    ap.add_argument("--expect", default="", help="期望的鏡頭 x,y；不符就退出碼 1")
    a = ap.parse_args()
    ox, oy = (int(v) for v in a.origin.split(","))

    grid = [list(map(int, l.split())) for l in open(a.grid)]
    H, W = len(grid), len(grid[0])
    freq = Counter(v for row in grid for v in row)

    at = norm(Image.open(a.atlas).convert("RGB"))
    idx = {}
    n = (at.size[0] // a.size) * (at.size[1] // a.size)
    for k in range(n):
        bx, by = (k % a.atlas_cols) * a.size, (k // a.atlas_cols) * a.size
        idx.setdefault(at.crop((bx, by, bx + a.size, by + a.size)).tobytes(), k)

    rc = 0
    for p in a.shots:
        im = Image.open(p).convert("RGB")
        if a.scale != 1:
            im = im.resize((im.size[0] // a.scale, im.size[1] // a.scale), Image.NEAREST)
        im = norm(im)
        seen = {}
        for j in range(a.rows):
            for i in range(a.cols):
                x, y = ox + a.size * i, oy + a.size * j
                if x + a.size > min(a.max_x, im.size[0]) or y + a.size > im.size[1]:
                    continue
                k = idx.get(im.crop((x, y, x + a.size, y + a.size)).tobytes())
                if k is not None and 0 < freq.get(k, 0) <= a.rare:
                    seen[(i, j)] = k
        sc = []
        for cy in range(H - a.rows + 1):
            for cx in range(W - a.cols + 1):
                sc.append((sum(1 for (i, j), k in seen.items()
                               if grid[cy + j][cx + i] == k), cx, cy))
        sc.sort(reverse=True)
        top = sc[0]
        second = next((s for s in sc[1:] if (s[1], s[2]) != (top[1], top[2])), (0, 0, 0))
        line = (f"{os.path.basename(p):26s} 有辨識力的格 {len(seen):3d} → "
                f"鏡頭 ({top[1]},{top[2]}) 對上 {top[0]}，次佳 {second[0]}")
        if a.expect:
            ex, ey = (int(v) for v in a.expect.split(","))
            if (top[1], top[2]) == (ex, ey):
                line += "  pass"
            else:
                line += f"  FAIL（期望 {ex},{ey}）"
                rc = 1
        print(line)
    return rc


if __name__ == "__main__":
    sys.exit(main())
