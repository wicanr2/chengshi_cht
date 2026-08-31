#!/usr/bin/env python3
"""把 DOS 原版與 remake 的編輯視窗**逐格**比到位元組。

輸入是兩張截圖：原版的 DOSBox 畫面（1024×768，遊戲區在 192,184）與 remake 的
3 倍畫布。兩邊都換算回 640×350 的原版座標，再一格一格比。

為什麼要逐「格」而不是逐「像素」：像素差異的數字沒有語意——同樣是兩千個
像素，可能是一整塊圖塊畫錯，也可能是十幾格各差幾點。**格數**直接回答
「有幾格畫得跟原版不一樣」，而且門檻訂得起來。

已知會差的格（不算 bug，寫在 docs/spec/ui-layout.md）：

  * 最底下一列 —— remake 的工具帶比原版高三個像素，底列被蓋掉；
  * 游標所在的 3×3 外框 —— 兩邊選的工具不同，框的顏色與位置不一樣。

用法：tools/shot_diff_cells.py <原版.png> <remake.png> [--min-hit N]
"""
import argparse
import sys

from PIL import Image

# 兩邊的地圖區原點（原版座標）。x=64、y=55 是圖塊的第一格；
# y=54 那一列是地圖區的白色外框。DOSBox 的截圖要再加上遊戲區位移 (192,184)。
VIEW_X, VIEW_Y = 64, 55
DOS_OFF_X, DOS_OFF_Y = 192, 184
# 兩邊都先關閉 City Form，所以完整 32×16 編輯區都能比較。
MAX_X = 576
COLS, ROWS, SIZE = 32, 16, 16


EGA = {0x00, 0x55, 0xaa, 0xff}


def check_palette(im, path):
    """截圖裡的顏色必須本來就是 EGA 十六色。

    ⚠ 這道檢查是必要的，不是保險。`norm()` 會把任何顏色四捨五入到 EGA 四階
    ——包括**還在淡入、根本還沒畫完**的那一張（實測抓到過整片
    `(25,24,34)`）。四捨五入之後它會變成純黑，然後和原版比出「全部不同」
    或更糟的「碰巧相同」。判準不能建立在會把壞資料變成好資料的前處理上。
    """
    bad = sum(1 for r, g, b in im.getdata()
              if r not in EGA or g not in EGA or b not in EGA)
    if bad > im.size[0] * im.size[1] // 100:
        raise SystemExit(
            f"{path}：{bad} 個像素不是 EGA 十六色，這張截圖八成是在畫面還沒"
            f"畫完的時候拍的。重拍，不要拿去比。")


def norm(im):
    """把顏色正規化到 EGA 的四階，兩邊才比得了。"""
    px = im.load()
    for y in range(im.size[1]):
        for x in range(im.size[0]):
            r, g, b = px[x, y]
            px[x, y] = (round(r / 85) * 85, round(g / 85) * 85, round(b / 85) * 85)
    return im


def load(path, scale, ox, oy):
    im = Image.open(path).convert("RGB")
    if scale != 1:
        im = im.resize((im.size[0] // scale, im.size[1] // scale), Image.NEAREST)
    check_palette(im, path)
    return norm(im), ox, oy


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("dos")
    ap.add_argument("remake")
    ap.add_argument("--min-hit", type=int, default=0)
    a = ap.parse_args()

    d, dx, dy = load(a.dos, 1, DOS_OFF_X + VIEW_X, DOS_OFF_Y + VIEW_Y)
    r, rx, ry = load(a.remake, 3, VIEW_X, VIEW_Y)

    same, diff, bad = 0, 0, []
    for j in range(ROWS):
        for i in range(COLS):
            if VIEW_X + SIZE * i + SIZE > MAX_X:
                continue
            box_d = (dx + SIZE * i, dy + SIZE * j, dx + SIZE * i + SIZE, dy + SIZE * j + SIZE)
            box_r = (rx + SIZE * i, ry + SIZE * j, rx + SIZE * i + SIZE, ry + SIZE * j + SIZE)
            if box_d[2] > d.size[0] or box_d[3] > d.size[1]:
                continue
            if box_r[2] > r.size[0] or box_r[3] > r.size[1]:
                continue
            if d.crop(box_d).tobytes() == r.crop(box_r).tobytes():
                same += 1
            else:
                diff += 1
                bad.append((i, j))
    print(f"完整編輯視窗格：逐位元相同 {same}，不同 {diff}")
    if bad:
        print(f"  不同的格 {bad[:16]}{' …' if len(bad) > 16 else ''}")
    if a.min_hit and same < a.min_hit:
        print(f"  FAIL：門檻 {a.min_hit}")
        return 1
    if a.min_hit:
        print("  pass")
    return 0


if __name__ == "__main__":
    sys.exit(main())
