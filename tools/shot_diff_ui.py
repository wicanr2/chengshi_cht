#!/usr/bin/env python3
"""產生 DOS 原版與 remake 的同畫布 UI 差分收據。

DOSBox 擷取是 1024×768，遊戲畫布固定在 (192,184) 的 640×350；remake 是
1920×1050 的三倍畫布。這支工具只做可重播的幾何正規化與逐像素統計，不會
把不同狀態或不同語言的畫面冒稱為 pixel-perfect。
"""

import argparse
import hashlib
import json
from pathlib import Path

from PIL import Image, ImageChops


ORIG_W, ORIG_H = 640, 350
DOS_BOX = (192, 184, 192 + ORIG_W, 184 + ORIG_H)
EGA_LEVELS = (0x00, 0x55, 0xAA, 0xFF)

PROFILES = {
    "ppf": {
        "full": (0, 0, 640, 350),
    },
    "base": {
        "full": (0, 0, 640, 350),
        "menu_bar": (0, 0, 640, 18),
        "edit_window": (5, 21, 580, 325),
        "city_form": (240, 21, 640, 350),
        "city_form_map": (274, 44, 635, 344),
    },
    "graphs": {
        "graphs_window": (240, 103, 544, 228),
        "graphs_plot": (303, 119, 539, 208),
    },
    "budget": {
        "budget_window": (171, 27, 456, 336),
    },
    "eval": {
        "eval_window": (39, 70, 552, 280),
    },
    "query": {
        "query_panel": (8, 208, 176, 322),
    },
    "newcity": {
        "dialog": (240, 70, 400, 280),
        "desktop_outside_dialog": (0, 18, 240, 350),
    },
    "brief": {
        "dialog": (168, 85, 472, 251),
        "client": (172, 89, 468, 247),
    },
}


def sha256(path):
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(1024 * 1024), b""):
            h.update(chunk)
    return h.hexdigest()


def nearest_ega(v):
    return min(EGA_LEVELS, key=lambda n: abs(n - v))


def normalize_ega(im):
    return im.point([nearest_ega(i) for i in range(256)] * 3)


def load_dos(path):
    im = Image.open(path).convert("RGB")
    if im.size == (ORIG_W, ORIG_H):
        return normalize_ega(im)
    if im.width < DOS_BOX[2] or im.height < DOS_BOX[3]:
        raise SystemExit(f"DOS 截圖尺寸不含固定遊戲畫布：{im.size}")
    return normalize_ega(im.crop(DOS_BOX))


def load_remake(path):
    im = Image.open(path).convert("RGB")
    if im.size == (ORIG_W, ORIG_H):
        return normalize_ega(im)
    if im.size != (ORIG_W * 3, ORIG_H * 3):
        raise SystemExit(f"remake 截圖不是 640×350 或三倍畫布：{im.size}")
    return normalize_ega(im.resize((ORIG_W, ORIG_H), Image.Resampling.NEAREST))


def region_stats(dos, remake, box):
    a, b = dos.crop(box), remake.crop(box)
    total = a.width * a.height
    same = sum(pa == pb for pa, pb in zip(a.getdata(), b.getdata()))
    diff = ImageChops.difference(a, b)
    return {
        "box": list(box),
        "same_pixels": same,
        "different_pixels": total - same,
        "total_pixels": total,
        "same_percent": round(same * 100 / total, 6),
        "difference_bbox": list(diff.getbbox()) if diff.getbbox() else None,
    }


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("dos")
    ap.add_argument("remake")
    ap.add_argument("--profile", choices=PROFILES, required=True)
    ap.add_argument("--state", required=True,
                    help="同狀態標籤；例如 scenario=1,cam=0,0,layer=0")
    ap.add_argument("--classification", choices=("exact-state", "nearby-state", "layout-only"),
                    required=True)
    ap.add_argument("--out", required=True, help="收據輸出目錄")
    args = ap.parse_args()

    dos_path, remake_path = Path(args.dos), Path(args.remake)
    out = Path(args.out)
    out.mkdir(parents=True, exist_ok=True)
    dos, remake = load_dos(dos_path), load_remake(remake_path)
    diff = ImageChops.difference(dos, remake)
    # 黑底上的原始差值不夠醒目，非零像素改成紅色。
    mask = diff.convert("L").point(lambda v: 255 if v else 0)
    marked = Image.new("RGB", dos.size, (0, 0, 0))
    marked.paste((255, 0, 0), mask=mask)
    dos.save(out / "dos-normalized.png")
    remake.save(out / "remake-normalized.png")
    marked.save(out / "difference-mask.png")

    report = {
        "schema": 1,
        "classification": args.classification,
        "state": args.state,
        "profile": args.profile,
        "canvas": [ORIG_W, ORIG_H],
        "normalization": "DOS crop (192,184,640,350); remake nearest-neighbor 3x→1x; nearest EGA level",
        "sources": {
            "dos": {"path": str(dos_path), "sha256": sha256(dos_path)},
            "remake": {"path": str(remake_path), "sha256": sha256(remake_path)},
        },
        "regions": {
            name: region_stats(dos, remake, box)
            for name, box in PROFILES[args.profile].items()
        },
    }
    (out / "report.json").write_text(
        json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    for name, stats in report["regions"].items():
        print(f"{name}: {stats['same_pixels']}/{stats['total_pixels']} "
              f"({stats['same_percent']:.3f}%)，差 {stats['different_pixels']}")


if __name__ == "__main__":
    main()
