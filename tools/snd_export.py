"""把 DOS 版的音效檔拆成 WAV。

用法：
    python3 tools/snd_export.py <解壓過的 .PSF 或 .V4> <輸出目錄> [取樣率]

檔案版面（見 docs/formats/05-psf-sound.md）：
    u16  之後的位元組數（＝檔案大小 − 2）
    重複 8 次：
        u16  這段的位元組數
        位元組  4 位元無號 PCM，一個位元組兩個取樣，中心值 8

取樣率還沒有一手證據，預設 11025，可以從命令列改。
"""

import os
import struct
import sys
import wave


def segments(data):
    """把整份檔案拆成一段一段的原始位元組。"""
    n = len(data)
    total = int.from_bytes(data[0:2], "little")
    if total != n - 2:
        raise ValueError(f"檔頭宣告 {total} 位元組，實際 {n - 2}")
    out, off = [], 2
    while off + 2 <= n:
        ln = int.from_bytes(data[off : off + 2], "little")
        if ln == 0 or off + 2 + ln > n:
            break
        out.append(data[off + 2 : off + 2 + ln])
        off += 2 + ln
    if off != n:
        raise ValueError(f"走到 {off} 就對不上了，檔案有 {n}")
    return out


def unpack4(raw, high_first=True):
    """4 位元 → 8 位元無號。原始值 0–15 直接乘 17 攤到 0–255。"""
    out = bytearray()
    for b in raw:
        hi, lo = b >> 4, b & 0x0F
        a, c = (hi, lo) if high_first else (lo, hi)
        out.append(a * 17)
        out.append(c * 17)
    return bytes(out)


def roughness(pcm):
    """相鄰取樣的平均絕對差。拆錯 nibble 順序會把波形打散，這個值會變大。"""
    if len(pcm) < 2:
        return 0.0
    return sum(abs(pcm[i] - pcm[i - 1]) for i in range(1, len(pcm))) / (len(pcm) - 1)


def main():
    data = open(sys.argv[1], "rb").read()
    outdir = sys.argv[2]
    rate = int(sys.argv[3]) if len(sys.argv) > 3 else 11025
    os.makedirs(outdir, exist_ok=True)

    for i, seg in enumerate(segments(data)):
        hi = unpack4(seg, True)
        lo = unpack4(seg, False)
        # 兩種 nibble 順序都試，取波形比較平滑的那個。
        pcm, order = (hi, "hi") if roughness(hi) <= roughness(lo) else (lo, "lo")
        path = os.path.join(outdir, f"{i}.wav")
        with wave.open(path, "wb") as w:
            w.setnchannels(1)
            w.setsampwidth(1)
            w.setframerate(rate)
            w.writeframes(pcm)
        print(
            f"{i}  {len(seg):6d} 位元組  {len(pcm) / rate:5.2f} 秒  "
            f"nibble={order}  粗糙度 hi={roughness(hi):.1f} lo={roughness(lo):.1f}"
        )


if __name__ == "__main__":
    main()
