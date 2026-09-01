#!/usr/bin/env python3
"""比兩張截圖的解碼像素是否完全相同，輸出可重跑的 JSON 收據。

用途是**同源證明**：發行包裡那顆二進位畫出來的畫面，與工作樹原始碼建出來的
畫面，是不是同一張。相同就代表既有的原版對拍結論可以整批繼承到發行包，
不必為了包裝層再跟原版重吵一次像素。

比的是解碼後的 RGB，不是檔案雜湊——PNG 的中繼資料（時間戳、產生器字串）
會讓兩個內容相同的檔案雜湊不同。
"""

import argparse
import hashlib
import json
import sys
from pathlib import Path

from PIL import Image, ImageChops


def sha256(path):
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(1 << 20), b""):
            h.update(chunk)
    return h.hexdigest()


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("a")
    ap.add_argument("b")
    ap.add_argument("--label", default="")
    ap.add_argument("--out", default="")
    args = ap.parse_args()

    ia = Image.open(args.a).convert("RGB")
    ib = Image.open(args.b).convert("RGB")
    rec = {
        "schema": 1,
        "label": args.label,
        "a": {"path": args.a, "sha256": sha256(args.a), "size": list(ia.size)},
        "b": {"path": args.b, "sha256": sha256(args.b), "size": list(ib.size)},
    }
    if ia.size != ib.size:
        rec["identical"] = False
        rec["reason"] = "尺寸不同"
        print(json.dumps(rec, ensure_ascii=False))
        return 1

    diff = ImageChops.difference(ia, ib)
    box = diff.getbbox()
    total = ia.size[0] * ia.size[1]
    if box is None:
        differing = 0
    else:
        # 逐像素數，不用直方圖近似：三個通道任一不同就算一個像素。
        differing = sum(1 for px in diff.convert("L").point(lambda v: 255 if v else 0).getdata() if px)
    rec["pixels_total"] = total
    rec["pixels_differing"] = differing
    rec["identical"] = differing == 0
    rec["diff_bbox"] = list(box) if box else None
    if args.out:
        out = Path(args.out)
        out.parent.mkdir(parents=True, exist_ok=True)
        out.write_text(json.dumps(rec, ensure_ascii=False, indent=1), encoding="utf-8")
        if box:
            diff.point(lambda v: 255 if v else 0).save(out.with_suffix(".mask.png"))
    print(json.dumps(rec, ensure_ascii=False))
    return 0 if rec["identical"] else 1


if __name__ == "__main__":
    sys.exit(main())
