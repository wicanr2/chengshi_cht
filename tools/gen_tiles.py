#!/usr/bin/env python3
"""從 Micropolis 的 headers/sim.h 重產 internal/sim/tiles.go。

為什麼要用工具而不是手抄：圖塊常數有 130 條，手抄一定會錯，而且錯了之後
測試會把錯的釘住。改語意先改這支工具再重產，不要手改產物。

用法：tools/gen_tiles.py [封存路徑] > internal/sim/tiles.go
"""
import hashlib
import os
import re
import sys

DEFAULT = os.path.join(os.path.dirname(__file__), "..", "workplace", "ref",
                       "micropolis", "micropolis-activity", "src", "sim",
                       "headers", "sim.h")

# 圖塊編號區（sim.h 的 "Character Mapping" 註解之後到 TILE_COUNT 為止）
TILE_START = "/* Character Mapping */"
TILE_END = "TILE_COUNT"

FLAGS = ["PWRBIT", "CONDBIT", "BURNBIT", "BULLBIT", "ANIMBIT", "ZONEBIT",
         "ALLBITS", "LOMASK"]
SIZES = ["SimWidth", "SimHeight", "HISTLEN", "MISCHISTLEN"]

DEFINE = re.compile(r"^#define\s+([A-Za-z_][A-Za-z0-9_]*)\s+(\d+)\s*(?:/\*(.*?)\*/)?\s*$")


def main():
    path = sys.argv[1] if len(sys.argv) > 1 else DEFAULT
    with open(path, "r", errors="replace") as fh:
        text = fh.read()
    digest = hashlib.sha256(text.encode("utf-8", "replace")).hexdigest()

    lines = text.splitlines()
    try:
        i0 = next(i for i, l in enumerate(lines) if TILE_START in l)
        i1 = next(i for i, l in enumerate(lines) if TILE_END in l and l.startswith("#define"))
    except StopIteration:
        sys.exit("在 %s 裡找不到圖塊區的界線——標頭換版了？" % path)

    tiles = []
    for line in lines[i0:i1 + 1]:
        m = DEFINE.match(line.strip())
        if m:
            tiles.append((m.group(1), int(m.group(2)), (m.group(3) or "").strip()))

    flags, sizes = [], []
    for line in lines:
        m = DEFINE.match(line.strip())
        if not m:
            continue
        if m.group(1) in FLAGS:
            flags.append((m.group(1), int(m.group(2)), (m.group(3) or "").strip()))
        elif m.group(1) in SIZES:
            sizes.append((m.group(1), int(m.group(2)), (m.group(3) or "").strip()))

    if not tiles or len(flags) != len(FLAGS):
        sys.exit("抽出來的常數數量不對：tiles=%d flags=%d" % (len(tiles), len(flags)))

    w = sys.stdout.write
    w("// 本檔由 tools/gen_tiles.py 從 Micropolis 的 headers/sim.h 重產，不要手改。\n")
    w("// 來源 SHA-256：%s\n" % digest)
    w("// 證據：docs/re/03-map-and-tiles.md\n")
    w("//\n")
    w("// UNUSED_TRASH* 與帶 `bogus?` 註解的編號照抄，不要整理掉——\n")
    w("// 它們是原版的一部分，動了會讓對拍失準。\n")
    w("\npackage sim\n\n")

    w("// 世界尺寸與歷史統計長度。headers/sim.h\n")
    w("const (\n")
    for n, v, c in sizes:
        w("\t%s = %d%s\n" % (n, v, ("  // " + c) if c else ""))
    w(")\n\n")

    w("// 每一格的旗標位元。headers/sim.h\n")
    w("const (\n")
    for n, v, c in flags:
        w("\t%s = %d%s\n" % (n, v, ("  // " + c) if c else ""))
    w(")\n\n")

    w("// 圖塊編號。headers/sim.h 的 Character Mapping 區，逐條重產。\n")
    w("const (\n")
    for n, v, c in tiles:
        w("\t%s = %d%s\n" % (n, v, ("  // " + c) if c else ""))
    w(")\n")


if __name__ == "__main__":
    main()
