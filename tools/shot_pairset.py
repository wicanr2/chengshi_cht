#!/usr/bin/env python3
"""成對重複量測：判斷兩顆二進位畫出來的畫面有沒有可分辨的差別。

為什麼不是直接比一張對一張：讀城市檔時 `LoadCitySeed` 拿的是
`sim.RandomSeed()`（`time.Now().UnixNano()`），而 `DoSimInit` 的那次
`MapScan` 會擲亂數——**同一顆二進位讀同一份檔，兩次啟動的世界就已經不同**。
`-seed` 只接到「開新城市」那條路，讀檔與載劇本都沒有。

所以判準是：**乙側與甲側的差距，不大於甲側或乙側自己跟自己的差距**。
自己跟自己是 0 的場景（沒有世界的畫面），判準自動收斂成逐像素相同。
"""

import argparse
import hashlib
import json
import sys
from pathlib import Path

from PIL import Image, ImageChops


def sha256(p):
    h = hashlib.sha256()
    with open(p, "rb") as f:
        for c in iter(lambda: f.read(1 << 20), b""):
            h.update(c)
    return h.hexdigest()


def load(p):
    return Image.open(p).convert("RGB")


def diff_count(a, b):
    if a.size != b.size:
        return None, None
    d = ImageChops.difference(a, b)
    box = d.getbbox()
    if box is None:
        return 0, None
    mask = d.convert("L").point(lambda v: 255 if v else 0)
    return sum(1 for px in mask.getdata() if px), list(box)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--a", nargs="+", required=True, help="甲側的重複截圖")
    ap.add_argument("--b", nargs="+", required=True, help="乙側的重複截圖")
    ap.add_argument("--label", required=True)
    ap.add_argument("--out", default="")
    args = ap.parse_args()

    A = [load(p) for p in args.a]
    B = [load(p) for p in args.b]
    rec = {
        "schema": 1,
        "label": args.label,
        "a_files": [{"path": p, "sha256": sha256(p)} for p in args.a],
        "b_files": [{"path": p, "sha256": sha256(p)} for p in args.b],
    }

    within_a = [diff_count(A[i], A[j])[0] for i in range(len(A)) for j in range(i + 1, len(A))]
    within_b = [diff_count(B[i], B[j])[0] for i in range(len(B)) for j in range(i + 1, len(B))]
    cross = []
    for i, x in enumerate(A):
        for j, y in enumerate(B):
            n, box = diff_count(x, y)
            cross.append({"a": i, "b": j, "differing": n, "bbox": box})

    noise = max(within_a + within_b) if (within_a + within_b) else 0
    cross_min = min(c["differing"] for c in cross)
    cross_max = max(c["differing"] for c in cross)
    rec["within_a"] = within_a
    rec["within_b"] = within_b
    rec["noise_floor"] = noise
    rec["cross"] = cross
    rec["cross_min"] = cross_min
    rec["cross_max"] = cross_max
    rec["pixels_total"] = A[0].size[0] * A[0].size[1]
    rec["verdict"] = "same" if cross_min <= noise else "different"

    if args.out:
        o = Path(args.out)
        o.parent.mkdir(parents=True, exist_ok=True)
        o.write_text(json.dumps(rec, ensure_ascii=False, indent=1), encoding="utf-8")
    print(json.dumps({k: rec[k] for k in
                      ("label", "noise_floor", "cross_min", "cross_max", "verdict")},
                     ensure_ascii=False))
    return 0 if rec["verdict"] == "same" else 1


if __name__ == "__main__":
    sys.exit(main())
