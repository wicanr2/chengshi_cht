#!/usr/bin/env python3
# 從 SIMCITY.EXE 抽出符號表：名字 → 模組:位移。
#
# 這份表是**原版自己帶的**，不是我們推測的。它在載入器區（明文，0x1000
# 起），紀錄格式是：
#
#     [4 位元組 far 指標][種類 word][模組 word][位移 word][…][長度][名字][00]
#     種類 0x0003 = 程式、0x0103 = 資料
#
# 用途：DOS 版有東西要查的時候，先看它在哪個模組。目前唯一還沒解的
# DOS 專屬問題是音效——八段 PCM 各對應哪個事件（docs/re/16-dos-oracle.md §4）。
# 音效的三支常式都在**模組 0x23**。
#
# ⚠ 只有 30 個符號，不是完整的符號表；看起來是載入器要修正的那一批。
# ⚠ 模組編號還沒對應到解壓後映像的位址，所以**還讀不到那些常式的內容**。
#
#   python3 tools/dos_symbols.py "workplace/dos110/SIMCITY 1.10/SIMCITY.EXE"
import re, struct, sys

KIND = {0x0003: "程式", 0x0103: "資料"}


def symbols(path):
    d = open(path, "rb").read()
    out = {}
    for m in re.finditer(rb"(.)(_[A-Za-z][A-Za-z0-9_]{1,24})\x00", d):
        if m.group(1)[0] != len(m.group(2)):
            continue
        tail = d[m.end():m.end() + 10]
        if len(tail) < 10:
            continue
        _, kind, mod, off = struct.unpack("<IHHH", tail)
        if kind not in KIND:
            continue
        out[m.group(2).decode()] = (kind, mod, off)
    return out


def main(path):
    syms = symbols(path)
    print(f"共 {len(syms)} 個符號")
    mods = {}
    for name, (kind, mod, off) in syms.items():
        mods.setdefault(mod, []).append((off, name, kind))
    for mod in sorted(mods):
        print(f"模組 0x{mod:02X}")
        for off, name, kind in sorted(mods[mod]):
            print(f"    {off:#06x}  {KIND[kind]}  {name}")


if __name__ == "__main__":
    main(sys.argv[1])
