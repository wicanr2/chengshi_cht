#!/usr/bin/env python3
"""把原版 `.PTF` 裡**新出現的鍵**補進 TSV，已經有的一列都不動。

這支取代舊的 `merge.py`。舊的做法是「python 表 → 整檔重寫 toml」，
所以任何直接編在產出檔裡的譯文都會被洗掉——實際發生過：asia 的
「水車」變回「核能發電廠」、「海嘯」變回「水災」，六筆，
而且要等 `TestStyleFilesDoNotInheritBaseWording` 才抓得到。

現在 **TSV 就是唯一真相**，這支只做兩件事：

  1. 原版檔案裡有、TSV 沒有的鍵 → 補一列空的（附上原文長度）
  2. 原文長度變了 → 更新 `len` 欄

譯文欄一個字都不碰。

⚠ **預設是空跑**，只印出差在哪，不動檔案。原因是原版 `.PTF` 的段落
不是每一筆都是給玩家看的字串——第 0 段的奇數索引是兩個位元組的屬性、
第 2 段每一筆前面有三個位元組的前綴（docs/formats/04-ptf-messages.md）。
照單全收會在唯一真相裡塞進一百多列不是文字的東西。要真的寫入
加 `--write`，而且寫完要自己看一遍新增的是不是真的文字。

用法（在 docker 裡跑，見 tools/i18n.sh）：
    tools/i18n/skeleton.py <解開的 SIMCITY 1.10 目錄> internal/i18n/messages [--write]
"""
import os
import sys

# 檔名 → 原版 `.PTF`
FILES = {
    "message": "MESSAGE.PTF",
    "asia_msg": "ASIA_MSG.PTF",
    "medi_msg": "MEDI_MSG.PTF",
    "west_msg": "WEST_MSG.PTF",
    "fusa_msg": "FUSA_MSG.PTF",
    "feur_msg": "FEUR_MSG.PTF",
    "moon_msg": "MOON_MSG.PTF",
}


def lzss(src):
    """DOS 版共用的 LZSS（docs/formats/02-dos-lzss.md）。"""
    out = bytearray()
    win = bytearray(4096)
    p = 0
    r = 4096 - 18
    while p < len(src):
        flags = src[p]
        p += 1
        for b in range(8):
            if p >= len(src):
                return bytes(out)
            if flags & (1 << b):
                c = src[p]
                p += 1
                out.append(c)
                win[r] = c
                r = (r + 1) & 0xFFF
            else:
                if p + 1 >= len(src):
                    return bytes(out)
                a, bb = src[p], src[p + 1]
                p += 2
                pos = a | ((bb & 0xF0) << 4)
                ln = (bb & 0x0F) + 3
                for k in range(ln):
                    c = win[(pos + k) & 0xFFF]
                    out.append(c)
                    win[r] = c
                    r = (r + 1) & 0xFFF
    return bytes(out)


def sections(data):
    """切段落，回傳 [(段, [字串…])]。版面見 docs/formats/04-ptf-messages.md。"""
    out = []
    p = 0
    while p + 4 <= len(data):
        size = data[p] | (data[p + 1] << 8)
        if size == 0:
            break
        body = data[p + 4:p + 4 + size]
        strs = []
        q = 0
        while q < len(body):
            z = body.find(b"\x00", q)
            if z < 0:
                z = len(body)
            strs.append(body[q:z])
            q = z + 1
        out.append(strs)
        p += 4 + size
    return out


def find(dirpath, name):
    for e in os.listdir(dirpath):
        if e.lower() == name.lower():
            return os.path.join(dirpath, e)
    return None


def main():
    args = [a for a in sys.argv[1:] if not a.startswith("--")]
    write = "--write" in sys.argv
    data_dir = os.path.join(args[0], "DATA")
    out_dir = args[1] if len(args) > 1 else "internal/i18n/messages"
    for name, ptf in sorted(FILES.items()):
        tsv = os.path.join(out_dir, name + ".tsv")
        src = find(data_dir, ptf)
        if src is None or not os.path.exists(tsv):
            print("%-14s 跳過（找不到 %s）" % (name, ptf))
            continue
        secs = sections(lzss(open(src, "rb").read()))
        want = {}
        for si, strs in enumerate(secs):
            for i, b in enumerate(strs):
                if b:
                    want["%d.%d" % (si, i)] = len(b)

        lines = open(tsv, encoding="utf-8").read().split("\n")
        head = lines[0].split("\t")
        seen = set()
        added = relen = 0
        for i in range(1, len(lines)):
            if not lines[i]:
                continue
            f = lines[i].split("\t")
            seen.add(f[0])
            if f[0] in want and f[1] != str(want[f[0]]):
                relen += 1
                if write:
                    f[1] = str(want[f[0]])
                    lines[i] = "\t".join(f)
        tail = [k for k in want if k not in seen]
        tail.sort(key=lambda k: tuple(int(x) for x in k.split(".")))
        body = [ln for ln in lines[1:] if ln]
        for k in tail:
            added += 1
            if write:
                body.append("\t".join([k, str(want[k])] + [""] * (len(head) - 2)))
        if write:
            open(tsv, "w", encoding="utf-8").write(
                "\n".join([lines[0]] + body) + "\n")
        print("%-14s %s新增 %d 列，長度不同 %d 列" % (
            name, "" if write else "（空跑）", added, relen))


if __name__ == "__main__":
    main()
