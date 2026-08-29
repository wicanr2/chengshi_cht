#!/usr/bin/env python3
# 解開 SIMCITY.EXE 的自解壓外殼，把程式碼倒出來。
#
# 為什麼要有它：DOS 版的規則只有反組譯答得了的那幾條（目前只有一條——
# 汙染權重，docs/re/18-dos-parity.md §6）。而這支執行檔進 IDA 只解得出
# 三個函式一個字串，因為它是打包過的。
#
# 檔案結構（SHA-256 66457cc4…，玩家自備的 1.10 破解版）：
#
#     0x000  MZ 檔頭。CS:IP 指向 0x1EA0:0 —— **那是破解程式的 stub**，
#            不是原版進入點。stub 掛 INT 21h，攔 AH=30h，在記憶體裡
#            搜一段遮罩樣式再蓋兩個位元組（防拷判斷一律判過）。
#     0x200  壓縮資料。[壓縮長度 word][0x8000 word] ＋ LZSS 位元流
#     0x1000 原版的自解壓載入器（stub 最後 retf 到 載入段+0xE0:0）
#     0x1EC00 破解 stub
#
# 壓縮法是 LZSS，與資料檔（docs/formats/02-dos-lzss.md）同一支，
# **多一道 `ror 1`**：環形緩衝存原值，輸出前每個位元組右旋一位。
# 出處是原版載入器自己的程式碼（`ror dl,1 / mov es:[di],dl / rol dl,1`）。
#
# ⚠ **這份解壓不完全正確。** 輸出裡會出現一串串 0x10——那是環形緩衝
# 的初值 0x20 旋轉後的樣子，代表某些回指讀到了還沒寫過的槽位。
# 原版填的是 0xFEE 個位元組而不是 4096，尾端 18 格是殘留記憶體；
# 這裡照 4096 填。**所以拿它當證據時要逐位元組交叉檢查**，
# 不要整段當成可信的反組譯。
#
#   python3 tools/unpack_simcity_exe.py "workplace/dos110/SIMCITY 1.10/SIMCITY.EXE" out.bin
import struct, sys

RING, INIT, RPOS = 4096, 0x20, 4078


def unlzss(src, limit=1 << 22):
    """LZSS ＋ 輸出前 ror 1。回傳（解出的位元組, 耗用的來源位元組數）。"""
    win = bytearray(bytes([INIT]) * RING)
    r, out, i = RPOS, bytearray(), 0
    while i < len(src) and len(out) < limit:
        flags = src[i]
        i += 1
        for b in range(8):
            if i >= len(src) or len(out) >= limit:
                break
            if flags & (1 << b):
                c = src[i]
                i += 1
                out.append(((c >> 1) | (c << 7)) & 0xFF)
                win[r] = c
                r = (r + 1) & (RING - 1)
            else:
                if i + 1 >= len(src):
                    return bytes(out), i
                b1, b2 = src[i], src[i + 1]
                i += 2
                off = b1 | ((b2 & 0xF0) << 4)
                for k in range((b2 & 0x0F) + 3):
                    c = win[(off + k) & (RING - 1)]
                    out.append(((c >> 1) | (c << 7)) & 0xFF)
                    win[r] = c
                    r = (r + 1) & (RING - 1)
    return bytes(out), i


def main(src_path, out_path):
    d = open(src_path, "rb").read()
    # 區塊鏈：[壓縮長度 word][word][壓縮位元流]，一塊接一塊。
    # ⚠ 第二欄不是解壓後長度（第一塊寫 0x8000 但解出 39189）；
    # 目前只當它是未解的欄位，往下走靠壓縮長度。
    pos, out, n = 0x200, bytearray(), 0
    while pos + 4 < len(d):
        comp, second = struct.unpack("<HH", d[pos:pos + 4])
        if not 0 < comp < 0xF000:
            break
        body, used = unlzss(d[pos + 4:pos + 4 + comp])
        n += 1
        print(f"塊{n}：起 0x{pos:05X} 壓縮 {comp}（第二欄 {second}）"
              f"→ 解出 {len(body)}，耗用 {used}")
        out += body
        pos += 4 + comp
    print(f"共 {n} 塊，{len(out)} 位元組；push bp; mov bp,sp 共 "
          f"{out.count(bytes.fromhex('558bec'))} 處")
    open(out_path, "wb").write(bytes(out))
    print(f"寫入 {out_path}")

    # 已知的錨點：GetPValue 對應常式。找得到就印出來當回歸檢查。
    at = bytes(out).find(bytes.fromhex("2e81fa90007c05b81900"))
    if at < 0:
        print("\n⚠ 找不到 GetPValue 的錨點——解壓結果與筆記記錄的不一樣了")
        return
    print(f"\nGetPValue 對應常式在 0x{at:X}（docs/re/18-dos-parity.md §6.2）")
    print("  cmp dx,144 → mov ax,25    壅塞車流")
    print("  cmp dx, 80 → mov ax,10    稀疏車流")
    print("  cmp dx, 56 → mov ax,60    火災")
    print("  cmp dx, 52 → mov ax,-40   輻射")
    print("  cmp dx,761 → jl 跳回 mov ax,60   海港／機場／電廠")


if __name__ == "__main__":
    main(sys.argv[1], sys.argv[2] if len(sys.argv) > 2 else "unpacked.bin")
