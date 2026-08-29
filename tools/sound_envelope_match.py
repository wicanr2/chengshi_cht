#!/usr/bin/env python3
# 拿 X11 版的具名音效比對 DOS 的八段：**比波形包絡，不比長度**。
#
# 為什麼不比長度：`SOUNDDAT.V4` 的八段全部補齊到 64 的倍數，
# 所以段長不是錄音的實際長度（docs/formats/05-psf-sound.md §二）。
# 包絡先正規化成固定點數，長度與取樣率就被除掉了，剩下形狀。
#
# 兩邊都轉成「N 點的能量包絡」再算相關係數：
#   DOS  4 位元 PCM（0–15，中心 8）→ |x−8|
#   X11  8 位元 µ-law → 解碼成線性 → |x|
#
# ⚠ 就算對上也只到**強證據**：X11（1993）與 DOS（1991）是兩份不同年份的
# 錄音，不保證同一段素材。這支腳本只回報排名，不下結論。
#
#   python3 tools/sound_envelope_match.py
import glob, os, struct, wave, math

N = 48  # 包絡點數


def envelope(vals):
    """把一串已取絕對值的能量攤成 N 點，再正規化。"""
    if len(vals) < N:
        return None
    step = len(vals) / N
    env = []
    for i in range(N):
        a, b = int(i * step), int((i + 1) * step)
        seg = vals[a:b] or vals[a:a + 1]
        env.append(sum(seg) / len(seg))
    m = max(env)
    return [e / m for e in env] if m > 0 else None


def corr(a, b):
    ma, mb = sum(a) / len(a), sum(b) / len(b)
    va = sum((x - ma) ** 2 for x in a) ** 0.5
    vb = sum((x - mb) ** 2 for x in b) ** 0.5
    if va == 0 or vb == 0:
        return 0.0
    return sum((x - ma) * (y - mb) for x, y in zip(a, b)) / (va * vb)


ULAW = []
for i in range(256):
    u = ~i & 0xFF
    t = (((u & 0x0F) << 3) + 0x84) << ((u & 0x70) >> 4)
    t -= 0x84
    ULAW.append(-t if u & 0x80 else t)


def au_env(path):
    d = open(path, "rb").read()
    off, size, enc, rate, ch = struct.unpack(">IIIII", d[4:24])
    if size == 0xFFFFFFFF:
        size = len(d) - off
    if enc != 1:
        return None
    return envelope([abs(ULAW[b]) for b in d[off:off + size]])


def wav_env(path):
    with wave.open(path, "rb") as w:
        n, sw = w.getnframes(), w.getsampwidth()
        raw = w.readframes(n)
    if sw == 1:
        return envelope([abs(b - 128) for b in raw])
    vals = struct.unpack(f"<{len(raw)//2}h", raw[:len(raw) // 2 * 2])
    return envelope([abs(v) for v in vals])


def main():
    aus = {}
    for p in sorted(glob.glob("workplace/x11/sun/SimCity/res/*.au")):
        e = au_env(p)
        if e:
            aus[os.path.basename(p)[:-3]] = e
    if not aus:
        print("找不到 X11 的 .au —— 先解開 Rare simcity.zip 到 workplace/x11/")
        return
    print(f"X11 具名音效 {len(aus)} 個\n")
    # 只看這支腳本的姊妹指令倒出來的那八組：
    #   tools/go.sh run ./cmd/simtool sound -file <PSF|V4> -out workplace/snd/<名字>
    groups = ["base"] + ["ASIA", "MEDI", "WEST", "FEUR", "FUSA", "MOON"]
    for name in groups:
        group = os.path.join("workplace", "snd", name)
        if not os.path.isdir(group):
            continue
        print(f"== {os.path.basename(group)}")
        for p in sorted(glob.glob(f"{group}/*.wav")):
            e = wav_env(p)
            if not e:
                continue
            rank = sorted(((corr(e, v), k) for k, v in aus.items()), reverse=True)
            top = "  ".join(f"{k}={c:.2f}" for c, k in rank[:3])
            print(f"   段{os.path.basename(p)[0]}  {top}")


if __name__ == "__main__":
    main()
