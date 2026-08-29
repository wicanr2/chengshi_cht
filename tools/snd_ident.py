"""從 DOSBox 錄下來的原始音訊裡切出聲音事件，再比對已知的八段音效。

用法：
    python3 tools/snd_ident.py <錄音.raw> <時間表.marks> <參考 WAV 目錄>

錄音是 SDL 的 disk 驅動倒出來的，s16le 立體聲 22050 Hz，沒有檔頭。
時間表由 tools/dosbox_inner.sh 產生，每一行是「秒數 動作」。

先把錄音切成一段一段的聲音事件（用音量門檻），對齊到時間表上最近的動作，
再拿每一段去比對八個參考音效——參考的取樣率還沒有證據，所以逐一假設。
"""

import array
import glob
import math
import os
import sys
import wave

RATE = 22050
FRAME = 220  # 10 毫秒
RATES = [4000, 5000, 6000, 7000, 8000, 9000, 10000, 11025, 12000, 14000, 16000, 22050]


def load_raw(path):
    a = array.array("h")
    with open(path, "rb") as f:
        a.frombytes(f.read())
    if sys.byteorder == "big":
        a.byteswap()
    return [(a[i] + a[i + 1]) / 2.0 for i in range(0, len(a) - 1, 2)]


def envelope(s, frame=FRAME):
    out = []
    for i in range(0, len(s) - frame + 1, frame):
        c = s[i : i + frame]
        out.append(math.sqrt(sum(v * v for v in c) / frame))
    return out


def bursts(env, floor, min_frames=5, gap=25):
    """把超過門檻的區段合併成事件。gap 是允許的中間空檔（幀）。"""
    out, cur = [], None
    quiet = 0
    for i, v in enumerate(env):
        if v > floor:
            if cur is None:
                cur = i
            quiet = 0
        elif cur is not None:
            quiet += 1
            if quiet > gap:
                if i - quiet - cur >= min_frames:
                    out.append((cur, i - quiet))
                cur = None
    if cur is not None and len(env) - cur >= min_frames:
        out.append((cur, len(env)))
    return out


def read_ref(path):
    with wave.open(path) as w:
        n, r, width = w.getnframes(), w.getframerate(), w.getsampwidth()
        raw = w.readframes(n)
    if width == 1:
        return [b - 128.0 for b in raw]
    return [int.from_bytes(raw[i : i + 2], "little", signed=True) / 256.0
            for i in range(0, len(raw), 2)]


def resample_env(env, src_hz, dst_hz):
    """把包絡線從「每幀 10 毫秒 @ src 取樣率」換算到 dst 的時間軸。"""
    ratio = dst_hz / src_hz
    n = max(1, int(len(env) / ratio))
    return [env[min(len(env) - 1, int(i * ratio))] for i in range(n)]


def corr(a, b):
    if len(a) > len(b):
        a, b = b, a
    if len(a) < 3:
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
    raw, marks_path, ref_dir = sys.argv[1], sys.argv[2], sys.argv[3]
    mono = load_raw(raw)
    env = envelope(mono)
    peak = max(env) if env else 0
    floor = peak * 0.06
    print(f"錄音 {len(mono) / RATE:.1f} 秒，峰值 {peak:.0f}，門檻 {floor:.0f}")

    marks = []
    for line in open(marks_path, encoding="utf-8"):
        line = line.strip()
        if not line:
            continue
        t, rest = line.split(None, 1)
        marks.append((float(t), rest.strip()))

    refs = []
    for f in sorted(glob.glob(os.path.join(ref_dir, "*.wav")),
                    key=lambda p: int(os.path.basename(p).split(".")[0])):
        s = read_ref(f)
        # 參考音效倒出來時假設 8000 Hz，這裡只需要取樣值本身
        refs.append((os.path.basename(f)[:-4], s))

    for a, b in bursts(env, floor):
        t0, t1 = a * FRAME / RATE, b * FRAME / RATE
        near = min(marks, key=lambda m: abs(m[0] - t0)) if marks else (0, "")
        seg = env[a:b]
        m = max(seg)
        seg = [v / m for v in seg] if m else seg
        hits = []
        for name, s in refs:
            for r in RATES:
                # 參考的包絡線：每 10 毫秒（以假設的取樣率算）
                f = max(1, int(r * 0.01))
                e = envelope(s, f)
                mm = max(e) if e else 0
                if not mm:
                    continue
                e = [v / mm for v in e]
                if not (0.5 <= (len(s) / r) / (t1 - t0) <= 2.0):
                    continue
                hits.append((corr(seg, e), name, r))
        hits.sort(reverse=True)
        top = "  ".join(f"{n}@{r}={c:.2f}" for c, n, r in hits[:3]) or "（沒有長度相近的參考）"
        print(f"{t0:7.2f}–{t1:6.2f}s ({t1 - t0:.2f}s)  近 [{near[1]}]  {top}")


if __name__ == "__main__":
    main()
