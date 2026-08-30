"""用頻譜形狀反推 DOS 音效的取樣率。

用法：
    python3 tools/snd_rate_fit.py <段.wav> <對應的.au>       # 量一對
    python3 tools/snd_rate_fit.py --control <對應的.au>      # 正對照

原理只有一句：**同一份錄音，用不同取樣率播，頻譜會沿頻率軸整個縮放。**
X11 版的 `.au` 是 8000 Hz µ-law，取樣率已知；DOS 那份未知。若兩邊真的是
同一份素材，那麼

    ν_dos = ν_x11 × 8000 / R

對所有頻率成立（ν 是正規化頻率）。所以把 X11 的對數頻譜沿頻率軸拉伸一個
倍率去對 DOS 的，相關係數最高的那個倍率就給出 R。

**為什麼要另外做這件事**——`docs/re/16-dos-oracle.md` §五之五 已經用「總長度比」
量過一次。長度比的弱點是它假設兩邊剪裁一致：頭尾多剪或少剪幾個取樣，比值
就跟著動，而 `SOUNDDAT.V4` 的段長還補齊到 64 的倍數。頻譜形狀完全不受剪裁
影響（只要留得下夠多的音框），所以這是一個**對不同誤差敏感**的獨立方法，
不是把同一件事再算一次。

正對照是必要的，不是儀式：把 X11 那份人工降頻到已知的取樣率、加上 DOS 那種
4 位元量化，再餵給同一支估計器，看它還不還得回真值。還不回來就代表這支估計器
的輸出不能拿來當證據——先證明尺會量，再拿它去量未知的東西。
"""

import cmath
import math
import struct
import sys
import wave

X11_RATE = 8000  # `.au` 的取樣率，檔頭自述
NFFT = 512
RATE_LO, RATE_HI, RATE_STEP = 3000, 11000, 5


def load_au(path):
    """Sun `.au`：大端檔頭 ＋ 8 位元 µ-law。"""
    d = open(path, "rb").read()
    off = struct.unpack(">I", d[4:8])[0]
    size = struct.unpack(">I", d[8:12])[0]
    raw = d[off:] if size == 0xFFFFFFFF else d[off : off + size]
    out = []
    for b in raw:
        u = ~b & 0xFF
        exp, man = (u >> 4) & 7, u & 0x0F
        mag = (((man << 1) + 33) << exp) - 33
        out.append(-mag if u & 0x80 else mag)
    return out


def load_wav(path):
    """`tools/snd_export.py` 吐的 8 位元無號單聲道。檔頭裡的取樣率是佔位值，不要信。"""
    w = wave.open(path)
    raw = w.readframes(w.getnframes())
    w.close()
    return [b - 128 for b in raw]


def fft(a):
    n = len(a)
    if n == 1:
        return a
    ev, od = fft(a[0::2]), fft(a[1::2])
    out = [0j] * n
    for k in range(n // 2):
        t = cmath.exp(-2j * math.pi * k / n) * od[k]
        out[k], out[k + n // 2] = ev[k] + t, ev[k] - t
    return out


def logspec(x):
    """疊出平均對數頻譜，減掉平均值（只比形狀，不比音量）。"""
    m = sum(x) / len(x)
    x = [v - m for v in x]
    win = [0.5 - 0.5 * math.cos(2 * math.pi * i / NFFT) for i in range(NFFT)]
    acc = [0.0] * (NFFT // 2 + 1)
    n = 0
    for i in range(0, len(x) - NFFT, NFFT // 2):
        f = fft([x[i + j] * win[j] for j in range(NFFT)])
        for k in range(len(acc)):
            acc[k] += abs(f[k])
        n += 1
    if n == 0:
        return None
    s = [math.log10(v / n + 1e-6) for v in acc]
    m = sum(s) / len(s)
    return [v - m for v in s]


def _interp(ys, pos):
    i = int(pos)
    if i >= len(ys) - 1:
        return ys[-1]
    f = pos - i
    return ys[i] * (1 - f) + ys[i + 1] * f


def warp_curve(sd, sx):
    """回傳 [(取樣率, 相關係數)]：把 X11 頻譜縮放 R/8000 去對 DOS 頻譜。"""
    nd, nx = len(sd) - 1, len(sx) - 1
    nu = [i / nd * 0.5 for i in range(len(sd))]
    out = []
    for r10 in range(RATE_LO, RATE_HI + 1, RATE_STEP):
        idx = [(v * r10 / X11_RATE) for v in nu]
        keep = [i for i, v in enumerate(idx) if v <= 0.5]
        if len(keep) < len(sd) // 3:
            continue
        a = [sd[i] for i in keep]
        b = [_interp(sx, idx[i] / 0.5 * nx) for i in keep]
        ma, mb = sum(a) / len(a), sum(b) / len(b)
        a = [v - ma for v in a]
        b = [v - mb for v in b]
        num = sum(p * q for p, q in zip(a, b))
        den = math.sqrt(sum(p * p for p in a) * sum(q * q for q in b))
        out.append((r10, num / den if den else 0.0))
    return out


def report(sd, sx, label):
    c = warp_curve(sd, sx)
    r, best = max(c, key=lambda t: t[1])
    band = [rr for rr, v in c if v >= best - 0.01]
    print(f"{label}: {r} Hz（r={best:.3f}），峰值 0.01 內 {min(band)}–{max(band)}")
    return r, best


def resample(x, rate):
    """線性內插到指定取樣率，再套 DOS 那種 4 位元量化 —— 正對照要跟實測一樣粗。"""
    n = int(len(x) * rate / X11_RATE)
    y = [_interp(x, i * (len(x) - 1) / (n - 1)) for i in range(n)]
    step = max(abs(v) for v in y) / 7.0
    return [round(v / step) * step for v in y]


def main(argv):
    if len(argv) == 3 and argv[1] == "--control":
        x = load_au(argv[2])
        sx = logspec(x)
        for truth in (4800, 5400, 6000, 7000):
            report(logspec(resample(x, truth)), sx, f"  真值 {truth} → 估")
        return 0
    if len(argv) != 3:
        print(__doc__)
        return 2
    report(logspec(load_wav(argv[1])), logspec(load_au(argv[2])), "估計")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
