"""把 DOSBox 錄下來的音訊畫成時間軸，動作標在上面。

用法：python3 tools/snd_timeline.py <錄音.raw> <時間表.marks> <輸出 PNG>

看不到波形就只能亂調門檻。先畫出來，才知道哪一段是動作聲、哪一段是底噪。
"""

import array
import math
import sys

from PIL import Image, ImageDraw

RATE, FRAME = 22050, 220  # 10 毫秒一格
W, H, PAD = 1600, 260, 40


def main():
    a = array.array("h")
    with open(sys.argv[1], "rb") as f:
        a.frombytes(f.read())
    if sys.byteorder == "big":
        a.byteswap()
    mono = [(a[i] + a[i + 1]) / 2.0 for i in range(0, len(a) - 1, 2)]
    env = [math.sqrt(sum(v * v for v in mono[i : i + FRAME]) / FRAME)
           for i in range(0, len(mono) - FRAME + 1, FRAME)]
    dur = len(mono) / RATE
    peak = max(env) or 1

    img = Image.new("RGB", (W + 2 * PAD, H + 2 * PAD), (18, 20, 26))
    d = ImageDraw.Draw(img)
    base = PAD + H
    for x in range(W):
        i = int(x * len(env) / W)
        j = max(i + 1, int((x + 1) * len(env) / W))
        v = max(env[i:j]) if i < len(env) else 0
        d.line([(PAD + x, base), (PAD + x, base - v / peak * H)], fill=(120, 200, 150))
    for line in open(sys.argv[2], encoding="utf-8"):
        line = line.strip()
        if not line:
            continue
        t, rest = line.split(None, 1)
        x = PAD + int(float(t) / dur * W)
        d.line([(x, PAD - 12), (x, base)], fill=(220, 120, 90))
        # PIL 的預設點陣字沒有中文字模，標籤只留得下 ASCII。
        label = "".join(ch for ch in rest.split()[0] if ord(ch) < 128)[:12]
        d.text((x + 2, PAD - 26 + (int(float(t)) % 3) * 8), label or "*",
               fill=(220, 170, 140))
    d.text((PAD, base + 8), f"{dur:.1f}s  peak={peak:.0f}", fill=(200, 204, 212))
    img.save(sys.argv[3])
    print("寫入", sys.argv[3])


if __name__ == "__main__":
    main()
