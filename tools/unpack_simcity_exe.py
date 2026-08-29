#!/usr/bin/env python3
# 解開 SIMCITY.EXE 的自解壓外殼，把完整的程式映像倒出來。
#
# 為什麼要有它：DOS 版的規則只有反組譯答得了的那幾條。而這支執行檔進 IDA
# 只解得出三個函式一個字串——不是因為 IDA 不行，是因為**檔頭的進入點指向
# 破解程式的 stub**，原版在別的地方，而且是打包的。
#
# 檔案結構（SHA-256 66457cc4…，玩家自備的 1.10 破解版）：
#
#     0x00000  MZ 檔頭。CS:IP → 0x1EA0:0，**破解 stub**，不是原版進入點。
#     0x00200  壓縮流（前半）
#     0x01000  原版的自解壓載入器（明文，長度寫在它自己的 +06 ＝ 0x0C40）。
#              "Initializing SimCity" 在 +0x27D；它還帶一份符號表
#              （tools/dos_symbols.py）。
#     0x01C40  壓縮流（後半）
#     0x1EC00  破解 stub：掛 INT 21h 攔 AH=30h，記憶體裡搜遮罩樣式蓋兩個
#              位元組（防拷一律判過），最後 retf 到 載入段 + 0xE0 : 0。
#
# ⚠ **壓縮流被載入器切成兩段，要接起來才連續。** 這是解開這個容器的關鍵：
# 照檔案順序硬走會走進載入器的明文區，解出來的東西看起來像程式碼但其實
# 是垃圾（實測 189 KB 裡只有 29 個函式序言；接起來之後是 175 KB／607 個）。
#
# 接起來之後是一條乾淨的區塊鏈：
#
#     [區塊總長 word][解壓後長度 word][LZSS 位元流 …]
#     下一塊的位移 ＝ 這一塊的起點 ＋ 區塊總長（**總長含這 4 位元組檔頭**）
#
# 六塊，前五塊各解出 32768、最後一塊 11171，合計 175011——與載入器檔頭
# +28 宣稱的 0x2ADA3（175523）差 512。六塊這個數字也對得上 +2C ＝ 6。
#
# 壓縮法是 LZSS，與資料檔（docs/formats/02-dos-lzss.md）同一支，
# **多一道 `ror 1`**：環形緩衝存原值，輸出前每個位元組右旋一位。
# 出處是載入器自己的程式碼：`mov dl,[bx] / ror dl,1 / mov es:[di],dl /
# rol dl,1 / mov [bx],dl`。
#
#   python3 tools/unpack_simcity_exe.py "…/SIMCITY.EXE" workplace/ida/image.bin
import struct, sys

RING, INIT, RPOS = 4096, 0x20, 4078

# 壓縮流的兩段（載入器夾在中間）與破解 stub 的起點。
PART1 = (0x200, 0x1000)
PART2 = (0x1C40, 0x1EC00)

# GetPValue 對應常式的錨點：`cmp dx,90h / jl +5 / mov ax,19h`。
# 用位元組樣式找，不用位址。
ANCHOR = bytes.fromhex("81fa90007c05b81900")


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


def unpack(path, verbose=True):
    d = open(path, "rb").read()
    src = d[PART1[0]:PART1[1]] + d[PART2[0]:PART2[1]]
    pos, out, n = 0, bytearray(), 0
    while pos + 4 <= len(src):
        total, dst = struct.unpack("<HH", src[pos:pos + 4])
        if not 0x1000 < total < 0xF000:
            break
        body, _ = unlzss(src[pos + 4:pos + total], limit=dst)
        n += 1
        if verbose:
            print(f"塊{n}：起 {pos:#07x} 區塊長 {total:6d} 宣稱 {dst:6d} "
                  f"實得 {len(body):6d} 序言 {body.count(bytes.fromhex('558bec')):4d}")
        out += body
        pos += total
    return bytes(out), n


def main(path, out_path="image.bin"):
    img, n = unpack(path)
    pro = img.count(bytes.fromhex("558bec"))
    print(f"\n共 {n} 塊，{len(img)} 位元組，函式序言 {pro} 處")
    open(out_path, "wb").write(img)
    print(f"寫入 {out_path}")

    # 三個回歸錨點：塊數、序言數、GetPValue 的位置。
    if (n, pro) != (6, 607):
        print(f"⚠ 預期 6 塊／607 個序言，實得 {n}／{pro} —— 解壓行為變了")
    at = img.find(ANCHOR)
    if at < 0:
        print("⚠ 找不到 GetPValue 的錨點")
        return
    print(f"\nGetPValue 對應常式在 {at - 11:#x}（docs/re/18-dos-parity.md §6.3）")
    for name, imm in (("壅塞車流", 25), ("稀疏車流", 10), ("火災", 60),
                      ("輻射", -40), ("工業", 50), ("海港／機場／電廠", 60)):
        print(f"   {name:16s} {imm:>4d}")
    print("六個全部是 Micropolis 註解裡的舊值。")


if __name__ == "__main__":
    main(sys.argv[1], sys.argv[2] if len(sys.argv) > 2 else "image.bin")
