#!/usr/bin/env python3
# 從解開的 SIMCITY.EXE 映像抽出硬編碼字串。
#
# CLAUDE.md §3.2 列了中文化的三個文字來源：`.PTF` 訊息檔、X11 版的 Tcl
# 腳本、**DOS 執行檔內硬編碼的選單／按鈕／數字格式**。第三個一直拿不到，
# 因為執行檔是打包的——現在解得開了（tools/unpack_simcity_exe.py）。
#
#   python3 tools/unpack_simcity_exe.py "…/SIMCITY.EXE" workplace/ida/image.bin
#   python3 tools/dos_exe_strings.py workplace/ida/image.bin
#
# ⚠ 位移是**解壓後映像**的線性位移，不是檔案位移，也不是執行時的
# segment:offset。要引用某一條字串時連同這個位移一起寫，並註明是哪一份
# 映像（本專案的 SIMCITY.EXE SHA-256 66457cc4…）。
import re, sys

MIN = 4
# 玩家看得到的字串大多有這些特徵：字母佔多數、只用一般標點。
OK = set(" .\\%_-:,!?$()[]/'&+#=;\"")


def strings(img, minlen=MIN):
    out = []
    for m in re.finditer(rb"[ -~]{%d,}" % minlen, img):
        t = m.group(0).decode("latin1")
        if not re.search(r"[A-Za-z]{4,}", t):
            continue
        if sum(c.isalpha() or c in OK for c in t) / len(t) < 0.85:
            continue
        out.append((m.start(), t))
    return out


def main(path):
    img = open(path, "rb").read()
    ss = strings(img)
    print(f"# {len(ss)} 條（來源 {path}，{len(img)} 位元組）")
    for off, t in ss:
        print(f"{off:#08x}\t{t}")


if __name__ == "__main__":
    main(sys.argv[1] if len(sys.argv) > 1 else "workplace/ida/image.bin")
