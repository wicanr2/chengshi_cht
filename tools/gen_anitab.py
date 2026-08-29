#!/usr/bin/env python3
"""從 Micropolis 的 headers/animtab.h 重產 internal/sim/anitab.go。

`aniTile[1024]` 是動畫的下一格表：`animateTiles()` 把每一格帶 ANIMBIT 的
圖塊換成 `aniTile[圖塊編號]`，跑完一圈就是一個動畫循環（火在燒、煙在冒、
車在跑、雷達在轉）。

為什麼要用工具而不是手抄：一千零二十四筆，而且原始檔裡夾著 `#if 0`／
`#else` 兩個版本的路面車流表——手抄會抄錯，抄錯之後測試會把錯的釘住。
`#if 0` 那一段是**沒有編進去的舊版**，要取 `#else` 那一半。

用法：tools/gen_anitab.py [封存路徑] > internal/sim/anitab.go
"""
import hashlib
import os
import re
import sys

DEFAULT = os.path.join(os.path.dirname(__file__), "..", "workplace", "ref",
                       "micropolis", "micropolis-activity", "src", "sim",
                       "headers", "animtab.h")


def main():
    path = sys.argv[1] if len(sys.argv) > 1 else DEFAULT
    raw = open(path, "rb").read()
    digest = hashlib.sha256(raw).hexdigest()
    text = raw.decode("latin-1")

    # 只取第一個陣列（aniTile）。第二個陣列 aniSynch 只在 g_ani.c 的
    # `#if 0` 裡用到，沒有編進去。
    start = text.index("short aniTile[")
    body = text[text.index("{", start) + 1:]
    body = body[:body.index("};")]

    # 拆掉註解，並處理 #if 0 / #else / #endif：#if 0 那一段丟掉。
    body = re.sub(r"/\*.*?\*/", " ", body, flags=re.S)
    out, skip = [], False
    for line in body.splitlines():
        st = line.strip()
        if st.startswith("#if 0"):
            skip = True
            continue
        if st.startswith("#else"):
            skip = False
            continue
        if st.startswith("#endif"):
            skip = False
            continue
        if st.startswith("#"):
            raise SystemExit(f"沒預期的前置處理指令：{st}")
        if not skip:
            out.append(line)

    vals = [int(t, 0) for t in re.findall(r"0x[0-9a-fA-F]+|\d+", "\n".join(out))]
    # ⚠ 原始檔只列了 956 筆，宣告卻是 [1024]——**其餘由 C 補 0**。
    # 那 68 筆不是資料遺失，是語言規則；照補 0，不要改成「維持原圖塊」。
    # （實際上也用不到：TILE_COUNT 是 960。）
    if not 900 <= len(vals) <= 1024:
        raise SystemExit(f"取出 {len(vals)} 筆，不在合理範圍")
    listed = len(vals)
    vals += [0] * (1024 - listed)

    w = sys.stdout.write
    w("// 產生檔案，不要手改。來源：Micropolis headers/animtab.h\n")
    w(f"// SHA-256: {digest}\n")
    w("// 重產：tools/gen_anitab.py > internal/sim/anitab.go\n")
    w("\npackage sim\n\n")
    w("// aniTile 是動畫的下一格表。`AnimateTiles` 把每一格帶 ANIMBIT 的圖塊\n")
    w("// 換成 aniTile[圖塊編號]，走完一圈就是一個動畫循環。animtab.h\n")
    w("//\n")
    w("// ⚠ 原始檔裡路面車流那一段有兩個版本夾在 `#if 0`／`#else` 裡。\n")
    w("// 這裡取的是**實際編進去的那一半**（`#else` 之後）。\n")
    w(f"// 原始檔實際列出 {listed} 筆，其餘由 C 補 0（照補）。\n")
    w("var aniTile = [1024]uint16{\n")
    for i in range(0, 1024, 8):
        w("\t" + " ".join(f"{v}," for v in vals[i:i + 8]) + "\n")
    w("}\n")


main()
