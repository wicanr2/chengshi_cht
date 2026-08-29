#!/usr/bin/env python3
# 印出 `.PTF` 第 0 段（狀態訊息）每一則的類別。
#
# 類別是原版拿來決定要不要播警笛的欄位（類別 6／7 → 段 3），
# 語意見 docs/formats/04-ptf-messages.md §二、docs/re/16-dos-oracle.md §五之四。
#
# ⚠ 類別屬於**前一則**：第 n 則的文字在字串 2n、類別在字串 2n+1。
# 這是整份檔案唯一長這樣的段落，別把它推廣到其他段。
#
#   python3 tools/ptf_msg_class.py "workplace/dos110/SIMCITY 1.10/DATA/MESSAGE.PTF"
import struct
import sys

RING, INIT, RPOS = 4096, 0x20, 4078
NAMES = {2: "建議", 3: "問題", 4: "急迫", 5: "通知",
         6: "災難", 7: "地震", 8: "壅塞", 9: "工具錯誤"}


def unlzss(src):
    win = bytearray([INIT]) * RING
    r, out, p = RPOS, bytearray(), 0
    while p < len(src):
        flags = src[p]
        p += 1
        for b in range(8):
            if p >= len(src):
                break
            if flags & (1 << b):
                c = src[p]
                p += 1
                out.append(c)
                win[r] = c
                r = (r + 1) % RING
            else:
                if p + 1 >= len(src):
                    return out
                b1, b2 = src[p], src[p + 1]
                p += 2
                off = b1 | ((b2 & 0xF0) << 4)
                for k in range((b2 & 0x0F) + 3):
                    c = win[(off + k) % RING]
                    out.append(c)
                    win[r] = c
                    r = (r + 1) % RING
    return out


def main(path):
    d = unlzss(open(path, "rb").read())
    size, count = struct.unpack_from("<HH", d, 0)
    parts = d[4:4 + size].split(b"\x00")
    print(f"{path}：宣告 {count} 則")
    for n in range(len(parts) // 2):
        attr = parts[2 * n + 1]
        cls = attr[1] if len(attr) >= 2 and attr[0] == 0xFE else 0
        text = parts[2 * n].decode("latin1")
        if not text.strip():
            continue
        print(f"  訊息 {n + 1:2d}  類別 {cls} {NAMES.get(cls, ''):4s}  {text}")


if __name__ == "__main__":
    main(sys.argv[1])
