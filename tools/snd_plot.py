"""把音效畫成波形圖，用來眼睛判斷這是什麼聲音。

用法：python3 tools/snd_plot.py <WAV 目錄> <輸出 PNG>

判不了「這是警笛還是喇叭」的時候，波形的包絡線常常就給答案：
爆炸是一次陡升慢降、喇叭是兩段、警笛是規律起伏。
"""

import glob
import os
import sys
import wave

from PIL import Image, ImageDraw

W, ROW, PAD = 1000, 90, 26


def main():
    files = sorted(glob.glob(os.path.join(sys.argv[1], "*.wav")),
                   key=lambda p: int(os.path.basename(p).split(".")[0]))
    img = Image.new("RGB", (W + 2 * PAD, len(files) * ROW + 2 * PAD), (18, 20, 26))
    d = ImageDraw.Draw(img)
    for i, f in enumerate(files):
        with wave.open(f) as w:
            n, rate = w.getnframes(), w.getframerate()
            pcm = w.readframes(n)
        top = PAD + i * ROW
        mid = top + ROW // 2
        d.line([(PAD, mid), (PAD + W, mid)], fill=(60, 64, 74))
        step = max(1, n // W)
        for x in range(W):
            chunk = pcm[x * step : (x + 1) * step]
            if not chunk:
                break
            lo, hi = min(chunk) - 128, max(chunk) - 128
            d.line([(PAD + x, mid - hi * (ROW - 24) // 256),
                    (PAD + x, mid - lo * (ROW - 24) // 256)], fill=(120, 200, 150))
        d.text((PAD + 4, top + 2),
               f"{os.path.basename(f)}  {n} samples  {n / rate:.2f}s @ {rate}Hz",
               fill=(210, 214, 222))
    img.save(sys.argv[2])
    print("寫入", sys.argv[2])


if __name__ == "__main__":
    main()
