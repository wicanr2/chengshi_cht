"""把 DOS 的 8 段音效拿去比對 X11 版有名字的音效，猜它們是什麼聲音。

用法：python3 tools/snd_match.py <DOS 的 WAV 目錄> <X11 音效目錄>

原理：DOS 版與 X11 版是同一批錄音的不同編碼，所以**包絡線**（每 10 毫秒的
音量）應該對得上，即使位元深度、取樣率、雜訊都不同。DOS 的取樣率還沒有
證據，所以逐一假設幾個常見值，取相關性最高的組合。

輸出是**假說**，不是結論：包絡線像不代表就是同一個聲音，尤其短音效。
"""

import glob
import math
import os
import sys
import wave

FRAME_MS = 10.0
RATES = [5000, 6000, 7000, 8000, 9000, 10000, 11025, 12000, 14000, 16000, 22050]


def read_wav(path):
    """回傳 (取樣值 list，取樣率)。只吃 PCM，a-law 之類的跳過。"""
    with wave.open(path) as w:
        if w.getcomptype() != "NONE":
            raise ValueError("非 PCM")
        n, rate, width, ch = w.getnframes(), w.getframerate(), w.getsampwidth(), w.getnchannels()
        raw = w.readframes(n)
    out = []
    if width == 1:
        out = [b - 128 for b in raw[::ch]]
    elif width == 2:
        for i in range(0, len(raw), 2 * ch):
            v = int.from_bytes(raw[i : i + 2], "little", signed=True)
            out.append(v / 256.0)
    else:
        raise ValueError(f"{width * 8} 位元沒處理")
    return out, rate


def envelope(samples, rate):
    """每 FRAME_MS 毫秒的均方根音量，正規化成最大值 1。"""
    step = max(1, int(rate * FRAME_MS / 1000))
    env = []
    for i in range(0, len(samples) - step + 1, step):
        chunk = samples[i : i + step]
        env.append(math.sqrt(sum(v * v for v in chunk) / len(chunk)))
    m = max(env) if env else 0
    return [v / m for v in env] if m else env


def correlate(a, b):
    """兩條包絡線的最佳相關係數（允許平移）。短的那條在長的那條上滑。"""
    if len(a) > len(b):
        a, b = b, a
    if not a:
        return 0.0
    best = 0.0
    for lag in range(0, len(b) - len(a) + 1):
        w = b[lag : lag + len(a)]
        ma, mw = sum(a) / len(a), sum(w) / len(w)
        num = sum((x - ma) * (y - mw) for x, y in zip(a, w))
        da = math.sqrt(sum((x - ma) ** 2 for x in a))
        dw = math.sqrt(sum((y - mw) ** 2 for y in w))
        if da and dw:
            best = max(best, num / (da * dw))
    return best


def main():
    dos_dir, ref_dir = sys.argv[1], sys.argv[2]
    refs = []
    for f in sorted(glob.glob(os.path.join(ref_dir, "*.wav"))):
        try:
            s, r = read_wav(f)
            refs.append((os.path.basename(f)[:-4], envelope(s, r), len(s) / r))
        except Exception:
            continue
    print(f"參考音效 {len(refs)} 個")

    for f in sorted(glob.glob(os.path.join(dos_dir, "*.wav")),
                    key=lambda p: int(os.path.basename(p).split(".")[0])):
        s, _ = read_wav(f)
        hits = []
        for rate in RATES:
            env = envelope(s, rate)
            for name, renv, rdur in refs:
                # 長度差太多就不必比了：同一段錄音不會差兩倍以上
                dur = len(s) / rate
                if not (0.5 <= dur / rdur <= 2.0):
                    continue
                hits.append((correlate(env, renv), name, rate, dur))
        hits.sort(reverse=True)
        print(f"\n{os.path.basename(f)}")
        for c, name, rate, dur in hits[:5]:
            print(f"   {c:.3f}  {name:<18} 假設 {rate} Hz → {dur:.2f} 秒")


if __name__ == "__main__":
    main()
