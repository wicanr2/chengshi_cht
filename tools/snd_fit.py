"""拿 DOSBox 錄下來的實跑波形，反推「哪一段音效、什麼取樣率」。

用法：
    python3 tools/snd_fit.py <錄音.raw> <音效檔.PSF 或 .V4>

錄音是 SDL disk 驅動吐的 22050 Hz、16 位元、立體聲原始資料。
先用每 5 毫秒的標準差切出事件（音量不能當判準——DOSBox 播完會把喇叭
停在直流準位，看起來一直很大聲），再對每一個事件做兩件事：

1. **長度**：事件長度 × 取樣率 應該等於某一段的取樣數。
2. **波形**：把該段用假設的取樣率重取樣到 22050，跟錄到的波形算相關。

兩件事共用同一個取樣率未知數，所以八段 × 一格取樣率掃過去，
只有真的對上的組合會同時讓長度和波形都吻合。這比單看包絡線強得多。
"""

import struct
import sys

REC_RATE = 22050
FRAME_MS = 5.0


def read_raw(path):
    """22050 Hz、16 位元 LE、立體聲 → 單聲道 float list。"""
    d = open(path, "rb").read()
    n = len(d) // 4
    out = [0.0] * n
    for i in range(n):
        l, r = struct.unpack_from("<hh", d, i * 4)
        out[i] = (l + r) / 2.0
    return out


def lzss(src):
    """與 internal/assets/lzss.go 同一套參數。"""
    out = bytearray()
    win = bytearray([0x20] * 4096)
    r, i = 4078, 0
    while i < len(src):
        flags = src[i]
        i += 1
        for b in range(8):
            if i >= len(src):
                break
            if flags & (1 << b):
                c = src[i]
                i += 1
                out.append(c)
                win[r] = c
                r = (r + 1) % 4096
                continue
            if i + 1 >= len(src):
                return bytes(out)
            b1, b2 = src[i], src[i + 1]
            i += 2
            off = b1 | ((b2 & 0xF0) << 4)
            ln = (b2 & 0x0F) + 3
            for k in range(ln):
                c = win[(off + k) % 4096]
                out.append(c)
                win[r] = c
                r = (r + 1) % 4096
    return bytes(out)


def load_segments(path):
    """回傳八段，每段是 −1..1 的 float list（4 位元 PCM，高位 nibble 在前）。"""
    raw = open(path, "rb").read()
    d = raw if path.upper().endswith(".V4") else lzss(raw)
    off, segs = 2, []
    while off + 2 <= len(d):
        n = struct.unpack_from("<H", d, off)[0]
        off += 2
        if off + n > len(d):
            break
        body = d[off:off + n]
        off += n
        s = []
        for byte in body:
            s.append(((byte >> 4) - 8) / 8.0)
            s.append(((byte & 0x0F) - 8) / 8.0)
        segs.append(s)
    return segs


def envelope(x, hop):
    """每 hop 個取樣的標準差。"""
    out = []
    for i in range(0, len(x) - hop, hop):
        w = x[i:i + hop]
        m = sum(w) / len(w)
        out.append((sum((v - m) ** 2 for v in w) / len(w)) ** 0.5)
    return out


def find_events(x, floor_mult=6.0, min_frames=3, gap_frames=8):
    """用每 5 毫秒的標準差切事件，回傳 [(起, 迄)] 的取樣索引。"""
    hop = int(REC_RATE * FRAME_MS / 1000)
    env = envelope(x, hop)
    quiet = sorted(env)[len(env) // 10] + 1e-9
    thr = max(quiet * floor_mult, max(env) * 0.02)
    evs, run = [], None
    silent = 0
    for i, v in enumerate(env):
        if v > thr:
            if run is None:
                run = i
            silent = 0
        elif run is not None:
            silent += 1
            if silent >= gap_frames:
                if i - silent - run >= min_frames:
                    evs.append((run * hop, (i - silent) * hop))
                run = None
    if run is not None and len(env) - run >= min_frames:
        evs.append((run * hop, len(env) * hop))
    return evs


def resample(seg, rate):
    """把取樣率 rate 的 seg 線性內插到 REC_RATE。"""
    n = int(len(seg) * REC_RATE / rate)
    step = rate / REC_RATE
    out = [0.0] * n
    for i in range(n):
        p = i * step
        j = int(p)
        if j + 1 < len(seg):
            f = p - j
            out[i] = seg[j] * (1 - f) + seg[j + 1] * f
        elif j < len(seg):
            out[i] = seg[j]
    return out


def corr(a, b):
    n = min(len(a), len(b))
    if n < 8:
        return 0.0
    a, b = a[:n], b[:n]
    ma, mb = sum(a) / n, sum(b) / n
    sa = sum((v - ma) ** 2 for v in a) ** 0.5
    sb = sum((v - mb) ** 2 for v in b) ** 0.5
    if sa == 0 or sb == 0:
        return 0.0
    return sum((a[i] - ma) * (b[i] - mb) for i in range(n)) / (sa * sb)


def main():
    rec, psf = sys.argv[1], sys.argv[2]
    x = read_raw(rec)
    segs = load_segments(psf)
    print(f"錄音 {len(x)/REC_RATE:.1f} 秒；{len(segs)} 段，取樣數 "
          f"{[len(s) for s in segs]}")
    evs = find_events(x)
    print(f"切出 {len(evs)} 個事件")
    hop = int(REC_RATE * FRAME_MS / 1000)
    for k, (a, b) in enumerate(evs):
        dur = (b - a) / REC_RATE
        clip = x[a:b]
        cenv = envelope(clip, hop)
        best = []
        for si, seg in enumerate(segs):
            rate = len(seg) / dur          # 長度先把取樣率釘死
            rs = resample(seg, rate)
            e = envelope(rs, hop)
            best.append((corr(cenv, e), si, rate))
        best.sort(reverse=True)
        top = "  ".join(f"段{si} r={rate:.0f} ρ={c:.2f}" for c, si, rate in best[:3])
        print(f"事件 {k:2d}  {a/REC_RATE:8.3f}s  長 {dur:.3f}s   {top}")


main()
