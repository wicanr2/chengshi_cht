#!/usr/bin/env python3
"""把某一個語言欄**空著的格子**填起來。

⚠ **只填空的**。已經有值的格子一律不動——先前那支 merge 腳本會整檔重寫，
把直接編在產出檔裡的譯文洗掉（asia 的「水車」變回「核能發電廠」、
「海嘯」變回「水災」，六筆），這裡不重蹈覆轍。TSV 是唯一真相。

用法：
    tools/i18n/fill_lang.py zh_hans internal/i18n/messages   # 由繁體字面轉換
"""
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from hant2hans import convert as hant2hans  # noqa: E402

FILLERS = {"zh_hans": ("zh_hant", hant2hans)}


def main():
    lang = sys.argv[1]
    d = sys.argv[2] if len(sys.argv) > 2 else "internal/i18n/messages"
    src, fn = FILLERS[lang]
    for name in sorted(os.listdir(d)):
        if not name.endswith(".tsv"):
            continue
        p = os.path.join(d, name)
        lines = open(p, encoding="utf-8").read().split("\n")
        head = lines[0].split("\t")
        ci, si = head.index(lang), head.index(src)
        n = 0
        for i in range(1, len(lines)):
            if not lines[i]:
                continue
            f = lines[i].split("\t")
            while len(f) < len(head):
                f.append("")
            if f[ci] == "" and f[si] != "":
                f[ci] = fn(f[si])
                n += 1
            lines[i] = "\t".join(f)
        open(p, "w", encoding="utf-8").write("\n".join(lines))
        print("%-14s 補了 %3d 格 %s" % (name, n, lang))


if __name__ == "__main__":
    main()
