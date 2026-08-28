#!/usr/bin/env python3
"""把 Noto Sans CJK TC 烘成 24×24 的點陣字型圖集。

為什麼要烘成點陣而不是執行期讀 TTF：

  1. 發行包不必帶 16 MB 的 TTF，也不必要求玩家自備字型。
  2. 點陣在整數倍縮放下是銳利的。老遊戲的底圖用最近鄰放大，
     字如果走向量渲染會出現兩種質感混在同一畫面。
  3. 字集固定 → 產物可重現，diff 看得懂。

字集從 translations/ 與 internal/ui/ 的原始碼掃出來，只烘真的會用到的字。
文本改了就重跑一次；產物進版控，測試會擋住手改。

用法（在 docker 裡跑，見 tools/font.sh）：
    tools/build_font.py <輸出目錄>
"""
import json
import os
import re
import sys

from PIL import Image, ImageDraw, ImageFont

SIZE = 24          # 方塊字的格子邊長
ASCII_W = SIZE // 2  # 半形字寬
TTC = "/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc"
COLS = 64          # 圖集每列幾個字

ROOT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..")
SCAN_DIRS = ["translations", "internal/i18n", "internal/ui", "docs/manual-cht"]


def pick_tc_face():
    """從 ttc 裡挑出繁體那一個字面。

    ⚠ ttc 的 index 順序不保證，不能寫死。JP／KR／SC／TC／HK 五個字面
    共用大部分字形，挑錯不會壞掉——但「戶」「錄」這類有地區差異的字
    會出現簡體或日文的寫法，而且**看起來只是字型不同**，很難察覺。
    """
    for i in range(16):
        try:
            f = ImageFont.truetype(TTC, SIZE, index=i)
        except OSError:
            break
        name = " ".join(str(x) for x in f.getname())
        if "TC" in name:
            return i, name
    raise SystemExit("在 %s 裡找不到繁體字面" % TTC)


def collect_chars():
    """掃出所有要烘的字元。"""
    chars = set()
    # 半形：可列印的 ASCII
    for c in range(0x20, 0x7F):
        chars.add(chr(c))
    # 全形：從文本掃
    for d in SCAN_DIRS:
        p = os.path.join(ROOT, d)
        if not os.path.isdir(p):
            continue
        for dirpath, _, files in os.walk(p):
            for fn in files:
                if not fn.endswith((".md", ".toml", ".go")):
                    continue
                with open(os.path.join(dirpath, fn), encoding="utf-8") as fh:
                    for ch in fh.read():
                        if ord(ch) <= 0x7F:
                            continue
                        # ⚠ 全形空格 U+3000 的 str.isprintable() 是 False
                        # （Python 把 ASCII 以外的分隔字元都算不可列印），
                        # 但版面上真的會用到它。漏掉的話量測寬度會少一格，
                        # 整行文字往左縮，而且看起來只像「排版有點怪」。
                        if ch == "\u3000" or ch.isprintable():
                            chars.add(ch)
    return sorted(chars)


def is_wide(ch):
    """全形判定。CJK、全形標點、注音都算一格 24 寬。"""
    o = ord(ch)
    return (0x1100 <= o <= 0x115F or 0x2E80 <= o <= 0xA4CF or
            0xAC00 <= o <= 0xD7A3 or 0xF900 <= o <= 0xFAFF or
            0xFE30 <= o <= 0xFE6F or 0xFF00 <= o <= 0xFF60 or
            0xFFE0 <= o <= 0xFFE6 or 0x3000 <= o <= 0x303F)


def main():
    out = sys.argv[1] if len(sys.argv) > 1 else os.path.join(ROOT, "internal/textfont/assets")
    idx, name = pick_tc_face()
    font = ImageFont.truetype(TTC, SIZE - 4, index=idx)
    chars = collect_chars()
    rows = (len(chars) + COLS - 1) // COLS
    img = Image.new("L", (COLS * SIZE, rows * SIZE), 0)
    d = ImageDraw.Draw(img)
    meta = {"size": SIZE, "cols": COLS, "face": name, "glyphs": {}}
    for i, ch in enumerate(chars):
        x, y = (i % COLS) * SIZE, (i // COLS) * SIZE
        w = SIZE if is_wide(ch) else ASCII_W
        bbox = d.textbbox((0, 0), ch, font=font)
        gx = x + (w - (bbox[2] - bbox[0])) // 2 - bbox[0]
        gy = y + (SIZE - (bbox[3] - bbox[1])) // 2 - bbox[1]
        d.text((gx, gy), ch, font=font, fill=255)
        meta["glyphs"][ch] = [i, w]
    img.save(os.path.join(out, "font24.png"))
    with open(os.path.join(out, "font24.json"), "w", encoding="utf-8") as fh:
        json.dump(meta, fh, ensure_ascii=False, sort_keys=True, indent=0)
    print("字面 %s（index %d）" % (name, idx))
    print("烘了 %d 個字，圖集 %d×%d" % (len(chars), img.width, img.height))


if __name__ == "__main__":
    main()
