#!/usr/bin/env python3
"""畫台北、台中、台南三張 120×100 的粗胚地形，給 tools/citymap 轉成城市檔。

**這是風格化的地形，不是測繪資料。** 120×100 格要放下一座城市的水系與地勢，
比例必然是取捨過的：河道加寬到看得見、丘陵化成連續的林地、海岸線簡化成
幾段折線。目標是「玩起來認得出這是哪裡」，不是地圖精度。

粗胚只有三種字元（水／陸／林），岸線與林緣的圖塊交給引擎自己的
smoothRiver／smoothTrees 去長——那是原版 s_gen.c 的邊界規則。

形狀全部用**固定種子的值雜訊**推出來，不用亂數：同一份腳本每次畫出同一張圖。
直線與正圓在畫面上一眼就看得出是機器畫的，所以海岸、河道與丘陵邊緣都由
雜訊擾動；河道另外加一條低頻的蜿蜒，那是河谷的形狀，不是雜訊。
"""

import math
import os

W, H = 120, 100
LAND, WATER, TREE = ".", "~", "T"


# ── 值雜訊 ──────────────────────────────────────────────────────────────
def _hash(seed, i, j):
    h = (seed * 374761393 + i * 668265263 + j * 2147483647) & 0xFFFFFFFF
    h = (h ^ (h >> 13)) * 1274126177 & 0xFFFFFFFF
    h ^= h >> 16
    return h / 0xFFFFFFFF


def _smooth(t):
    return t * t * (3 - 2 * t)


def vnoise(seed, x, y):
    """二維值雜訊，回傳 0..1。格點取雜湊，格內用 smoothstep 內插。"""
    i, j = math.floor(x), math.floor(y)
    fx, fy = _smooth(x - i), _smooth(y - j)
    a = _hash(seed, i, j)
    b = _hash(seed, i + 1, j)
    c = _hash(seed, i, j + 1)
    d = _hash(seed, i + 1, j + 1)
    return (a * (1 - fx) + b * fx) * (1 - fy) + (c * (1 - fx) + d * fx) * fy


def fbm(seed, x, y, octaves=4, scale=0.06):
    """疊幾層雜訊，低頻決定大形狀、高頻決定邊緣毛邊。回傳 -1..1。"""
    total, amp, norm, f = 0.0, 1.0, 0.0, scale
    for o in range(octaves):
        total += (vnoise(seed + o * 101, x * f, y * f) * 2 - 1) * amp
        norm += amp
        amp *= 0.5
        f *= 2.0
    return total / norm


# ── 畫布基元 ────────────────────────────────────────────────────────────
def blank():
    return [[LAND] * W for _ in range(H)]


def put(g, x, y, ch):
    if 0 <= x < W and 0 <= y < H:
        g[y][x] = ch


def disk(g, cx, cy, r, ch):
    r2 = r * r
    for y in range(int(cy - r) - 1, int(cy + r) + 2):
        for x in range(int(cx - r) - 1, int(cx + r) + 2):
            if (x - cx) ** 2 + (y - cy) ** 2 <= r2:
                put(g, x, y, ch)


def blob(g, seed, cx, cy, rx, ry, ch, rough=0.35):
    """邊緣被雜訊咬過的橢圓。正圓一眼就看得出是機器畫的，所以不留正圓。"""
    for y in range(max(0, int(cy - ry * 1.6)), min(H, int(cy + ry * 1.6) + 1)):
        for x in range(max(0, int(cx - rx * 1.6)), min(W, int(cx + rx * 1.6) + 1)):
            d = math.hypot((x - cx) / rx, (y - cy) / ry)
            if d <= 1.0 + fbm(seed, x, y, 3, 0.14) * rough:
                g[y][x] = ch


def river(g, seed, pts, width, ch=WATER, meander=3.0, wobble=0.9):
    """沿控制點畫一條會蜿蜒的河。

    pts 是 (x, y, 寬度倍率)。除了高頻雜訊之外，另外加一條低頻的側向位移
    當作河谷的彎曲——只有高頻的話河道看起來是抖動的直線，不是彎的。
    """
    # 先把控制點內插成密集路徑。
    path = []
    for i in range(len(pts) - 1):
        (x0, y0, w0), (x1, y1, w1) = pts[i], pts[i + 1]
        steps = max(1, int(math.hypot(x1 - x0, y1 - y0) * 3))
        for s in range(steps):
            t = s / steps
            path.append((x0 + (x1 - x0) * t, y0 + (y1 - y0) * t, w0 + (w1 - w0) * t))
    path.append(pts[-1])

    n = len(path)
    for k, (x, y, wr) in enumerate(path):
        # 路徑切線 → 法線，位移沿法線加，河才會左右擺而不是變粗。
        px, py, _ = path[max(0, k - 1)]
        nx, ny, _ = path[min(n - 1, k + 1)]
        tx, ty = nx - px, ny - py
        L = math.hypot(tx, ty) or 1.0
        ox, oy = -ty / L, tx / L
        # 低頻蜿蜒 ＋ 高頻毛邊。兩端收斂到 0，免得河口與源頭飄離控制點。
        #
        # ⚠ 蜿蜒用**沿路徑的低頻雜訊**，不用正弦。正弦的波長固定，
        # 幾條河擺在一起就變成等距的波浪紋，一眼看得出是機器畫的。
        edge = min(1.0, min(k, n - 1 - k) / 12.0)
        s = fbm(seed + 17, k * 0.035, seed * 0.37, 3, 1.0) * meander * 2.2 * edge
        s += fbm(seed + 3, x, y, 3, 0.09) * wobble
        r = width * wr * (1.0 + fbm(seed + 9, x, y, 2, 0.2) * 0.18)
        disk(g, x + ox * s, y + oy * s, r, ch)


def coast(g, seed, profile, amp=4.0, edge="w"):
    """把海填進去。profile 是 (y, x) 控制點；岸線再被雜訊推移。"""
    ys = [p[0] for p in profile]
    xs = [p[1] for p in profile]
    for y in range(H):
        if y <= ys[0]:
            cx = xs[0]
        elif y >= ys[-1]:
            cx = xs[-1]
        else:
            for i in range(len(ys) - 1):
                if ys[i] <= y <= ys[i + 1]:
                    t = _smooth((y - ys[i]) / (ys[i + 1] - ys[i]))
                    cx = xs[i] + (xs[i + 1] - xs[i]) * t
                    break
        cx += fbm(seed, 0, y, 4, 0.11) * amp
        for x in range(W):
            if (edge == "w" and x < cx) or (edge == "e" and x > cx):
                g[y][x] = WATER


def range_hills(g, seed, spine, halfwidth, thresh=0.0, ch=TREE):
    """沿一條稜線長出山脈：離稜線越近越可能是林地，邊界由雜訊決定。

    spine 是 (x, y) 控制點；halfwidth 是山脈的半寬。用「距離場 ＋ 雜訊」
    而不是實心橢圓，山才有稜有谷，不是一坨。
    """
    pts = []
    for i in range(len(spine) - 1):
        (x0, y0), (x1, y1) = spine[i], spine[i + 1]
        steps = max(1, int(math.hypot(x1 - x0, y1 - y0)))
        for s in range(steps):
            t = s / steps
            pts.append((x0 + (x1 - x0) * t, y0 + (y1 - y0) * t))
    pts.append(spine[-1])

    for y in range(H):
        for x in range(W):
            if g[y][x] != LAND:
                continue
            d = min(math.hypot(x - px, y - py) for px, py in pts)
            # 1 在稜線上、0 在山腳外。
            v = 1.0 - d / halfwidth
            if v + fbm(seed, x, y, 4, 0.10) * 0.55 > thresh + 0.35:
                g[y][x] = ch


def scatter_woods(g, seed, density=0.22):
    """平地上零星的樹叢。全平的平原看起來像空白，原版的地圖也不會這樣。"""
    for y in range(H):
        for x in range(W):
            if g[y][x] != LAND:
                continue
            if fbm(seed, x, y, 3, 0.13) > 1.0 - density * 2:
                g[y][x] = TREE


def dump(g, path, header):
    with open(path, "w", encoding="utf-8") as f:
        for line in header.strip().split("\n"):
            f.write("# " + line + "\n")
        for row in g:
            f.write("".join(row) + "\n")
    land = sum(r.count(LAND) for r in g)
    water = sum(r.count(WATER) for r in g)
    tree = sum(r.count(TREE) for r in g)
    print("寫出 %s：陸 %d／水 %d／林 %d" % (path, land, water, tree))


# ── 台北 ────────────────────────────────────────────────────────────────
# 盆地被山圍著，三條河在西北會合之後出海：基隆河自東蜿蜒進來、
# 新店溪自南上來、大漢溪自西南上來，匯流成淡水河往西北流。
def taipei():
    g = blank()
    # 圍住盆地的四段山：北邊大屯與陽明山、東邊內湖南港、南邊木柵新店、
    # 西南林口台地。用稜線而不是圓塊，山才有走向。
    range_hills(g, 41, [(6, 14), (30, 4), (60, 6), (84, 14)], 15)      # 北
    range_hills(g, 42, [(104, 16), (114, 40), (108, 66)], 14)          # 東
    range_hills(g, 43, [(96, 88), (68, 96), (40, 92)], 13)             # 南
    range_hills(g, 44, [(4, 52), (10, 74), (26, 92)], 12)              # 西南
    range_hills(g, 45, [(88, 40), (96, 52)], 7)                        # 南港小丘
    scatter_woods(g, 46, 0.10)

    # 淡水河口與出海口：西北角。
    coast(g, 47, [(0, 22), (10, 14), (20, 4), (30, -30)], amp=3.0)
    # 淡水河本流：關渡 → 江子翠
    river(g, 48, [(22, 12, 1.3), (30, 22, 1.2), (37, 33, 1.1), (43, 45, 1.0),
                  (46, 54, 0.95)], 2.6, meander=1.6)
    # 基隆河：東邊進來，經南港、內湖、大直、圓山
    river(g, 49, [(108, 40, 0.6), (96, 34, 0.7), (84, 38, 0.75), (72, 32, 0.8),
                  (60, 28, 0.85), (48, 30, 0.9), (38, 32, 1.0)], 2.0, meander=2.6)
    # 新店溪：南邊上來，經景美、公館、萬華
    river(g, 50, [(70, 88, 0.6), (64, 78, 0.7), (58, 68, 0.8), (52, 60, 0.9),
                  (47, 55, 1.0)], 2.0, meander=2.2)
    # 大漢溪：西南上來
    river(g, 51, [(28, 90, 0.6), (33, 78, 0.7), (39, 66, 0.8), (44, 58, 0.9),
                  (46, 54, 1.0)], 1.9, meander=2.0)
    return g


# ── 台中 ────────────────────────────────────────────────────────────────
# 西為台灣海峽與台中港，海岸與盆地之間隔著大肚台地；
# 大甲溪在北、大肚溪在南，東側為中央山脈山腳。
def taichung():
    g = blank()
    coast(g, 61, [(0, 18), (24, 14), (48, 11), (72, 15), (99, 21)], amp=4.5)
    # 台中港：岸線挖進來的港灣與外堤。
    blob(g, 62, 14, 46, 8, 6, WATER, rough=0.3)
    river(g, 63, [(8, 46, 1.0), (22, 45, 0.7), (30, 46, 0.5)], 2.4, meander=0.8)
    # 大肚台地：南北向的長稜線。
    range_hills(g, 64, [(31, 22), (28, 48), (33, 74)], 9)
    # 東側山腳與北緣。
    range_hills(g, 65, [(112, 12), (116, 44), (110, 76), (114, 96)], 16)
    range_hills(g, 66, [(78, 6), (98, 10)], 11)
    range_hills(g, 67, [(80, 96), (104, 92)], 12)
    scatter_woods(g, 68, 0.12)
    # 大甲溪（北）與大肚溪（南）：由東往西出海。
    river(g, 69, [(110, 22, 0.6), (92, 25, 0.7), (72, 21, 0.8), (52, 25, 0.9),
                  (32, 22, 1.0), (14, 25, 1.1)], 2.2, meander=3.2)
    river(g, 70, [(112, 78, 0.6), (94, 82, 0.7), (74, 78, 0.8), (54, 82, 0.95),
                  (34, 79, 1.1), (14, 82, 1.2)], 2.2, meander=3.4)
    # 盆地裡的筏子溪與旱溪。
    river(g, 71, [(62, 28, 0.5), (59, 44, 0.55), (56, 60, 0.6), (52, 76, 0.6)],
          1.5, meander=2.0)
    river(g, 72, [(80, 30, 0.5), (77, 46, 0.5), (72, 62, 0.55), (66, 78, 0.6)],
          1.4, meander=2.0)
    return g


# ── 台南 ────────────────────────────────────────────────────────────────
# 西岸是沙洲與潟湖（台江內海殘跡）與安平港；曾文溪、鹽水溪、二仁溪
# 由東往西入海；東側是新化丘陵，其餘是嘉南平原。
def tainan():
    g = blank()
    coast(g, 81, [(0, 16), (28, 12), (52, 10), (76, 13), (99, 18)], amp=3.5)
    # 潟湖：沿岸一串半封閉的水域，形狀彼此不同。
    for n, (cx, cy, rx, ry) in enumerate([(23, 18, 6, 5), (21, 32, 7, 6),
                                          (24, 47, 5, 7), (20, 60, 7, 5),
                                          (24, 73, 5, 6), (21, 86, 6, 5)]):
        blob(g, 82 + n, cx, cy, rx, ry, WATER, rough=0.45)
    # 沙洲：潟湖外側那一排（七股、四草）。
    for n, (cx, cy, rx, ry) in enumerate([(15, 24, 3, 5), (14, 40, 3, 6),
                                          (16, 66, 3, 5), (14, 80, 3, 5)]):
        blob(g, 90 + n, cx, cy, rx, ry, LAND, rough=0.4)
    # 安平港水道。
    river(g, 94, [(9, 53, 1.0), (20, 52, 0.8), (30, 54, 0.6)], 2.4, meander=0.7)
    # 三條溪。
    river(g, 95, [(114, 20, 0.6), (96, 23, 0.7), (76, 19, 0.8), (56, 23, 0.9),
                  (38, 20, 1.0), (24, 22, 1.1)], 2.0, meander=3.0)
    river(g, 96, [(106, 52, 0.5), (88, 49, 0.6), (70, 53, 0.7), (52, 50, 0.8),
                  (38, 53, 0.9), (28, 52, 1.0)], 1.8, meander=2.6)
    river(g, 97, [(112, 86, 0.6), (94, 83, 0.7), (74, 87, 0.8), (56, 83, 0.9),
                  (38, 86, 1.0), (24, 84, 1.1)], 2.0, meander=3.0)
    # 新化丘陵：東側。
    range_hills(g, 98, [(112, 24), (116, 48), (110, 72)], 15)
    scatter_woods(g, 99, 0.13)
    return g


def main():
    out = os.path.join(os.path.dirname(os.path.abspath(__file__)), "maps")
    os.makedirs(out, exist_ok=True)
    for name, fn, note in [
        ("taipei", taipei,
         "台北盆地。三條河在西北會合成淡水河出海：基隆河自東、新店溪自南、\n"
         "大漢溪自西南。四周是大屯、內湖南港、木柵新店與林口台地。\n"
         "風格化地形，不是測繪資料。由 tools/gen_city_masks.py 產生。"),
        ("taichung", taichung,
         "台中盆地。西為台灣海峽與台中港，海岸與盆地之間隔著大肚台地；\n"
         "大甲溪在北、大肚溪在南，東側為中央山脈山腳。\n"
         "風格化地形，不是測繪資料。由 tools/gen_city_masks.py 產生。"),
        ("tainan", tainan,
         "台南。西岸是沙洲與潟湖（台江內海殘跡）與安平港；\n"
         "曾文溪、鹽水溪、二仁溪由東往西入海；東側是新化丘陵。\n"
         "風格化地形，不是測繪資料。由 tools/gen_city_masks.py 產生。"),
    ]:
        dump(fn(), os.path.join(out, name + ".txt"), note)


if __name__ == "__main__":
    main()
