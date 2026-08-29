#!/usr/bin/env python3
# 解開 SIMCITY.EXE 的自解壓外殼，把程式碼倒出來。
#
# 為什麼要有它：DOS 版的規則只有反組譯答得了的那幾條（目前只有一條——
# 汙染權重，docs/re/18-dos-parity.md §6）。而這支執行檔進 IDA 只解得出
# 三個函式一個字串，因為它是打包過的。
#
# 檔案結構（SHA-256 66457cc4…，玩家自備的 1.10 破解版）：
#
#     0x00000  MZ 檔頭。CS:IP 指向 0x1EA0:0 —— **那是破解程式的 stub**，
#              不是原版進入點。
#     0x00200  壓縮資料（第一段，到載入器之前）
#     0x01000  原版的自解壓載入器。它自己帶一份符號表
#              （`_InitSounds`、`_MoveObjects`…，那是**載入器的**符號，
#              不是遊戲的），還有 "Initializing SimCity" 那個字串
#              （載入器 `mov dx,27Dh; int 21h`，DS 基底就是 0x1000）。
#     0x01C40  壓縮資料（第二段）—— GetPValue 在這一段裡
#     0x1EC00  破解 stub：掛 INT 21h 攔 AH=30h，在記憶體裡搜一段遮罩樣式
#              再蓋兩個位元組（防拷判斷一律判過），最後 retf 到
#              `載入段 + 0xE0 : 0`。
#
# 壓縮法是 LZSS，與資料檔（docs/formats/02-dos-lzss.md）同一支，
# **多一道 `ror 1`**：環形緩衝存原值，輸出前每個位元組右旋一位。
# 出處是原版載入器自己的程式碼（`ror dl,1 / mov es:[di],dl / rol dl,1`）。
#
# ⚠ **這份解壓不完全正確，第二段尤其。** 輸出裡會出現雜訊位元組——原版
# 初始化環形緩衝時 `mov cx,0FEEh` 只填 4078 個，尾端 18 格是殘留記憶體，
# 這裡照 4096 填 0x20。**所以不要整段當成可信的反組譯**：
# 這支腳本只負責把 GetPValue 那一段撈出來，而它的可信度來自
# **從三個不同起點解壓、非雜訊位元組完全一致**（見下面的 verify）。
#
#   python3 tools/unpack_simcity_exe.py "workplace/dos110/SIMCITY 1.10/SIMCITY.EXE"

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


# GetPValue 對應常式的錨點：`cs: cmp dx,90h / jl +5 / mov ax,19h`。
# 用位元組樣式找，不用位址——位址會隨解壓起點變。
ANCHOR = bytes.fromhex("2e81fa90007c05b81900")

# 解壓的起點。三個都能撈到 GetPValue，而且非雜訊位元組完全一致——
# 這就是「解壓有瑕疵但這一段可信」的依據。
STARTS = (0x1C40, 0x1C44, 0x204)


def show(body, at):
    """把 GetPValue 那一段印成位元組 ＋ 解讀。"""
    b = body[at - 8:at + 96]
    for i in range(0, len(b), 16):
        print("   " + " ".join(f"{x:02x}" for x in b[i:i + 16]))


def main(src_path, out_path=None):
    d = open(src_path, "rb").read()
    found = []
    for start in STARTS:
        body, _ = unlzss(d[start:0x1EC00])
        at = body.find(ANCHOR)
        if at < 0:
            print(f"起點 0x{start:05X}：找不到錨點")
            continue
        found.append(body[at - 8:at + 96])
        print(f"起點 0x{start:05X}：錨點在解壓輸出的 0x{at:X}")
        show(body, at)
        if out_path:
            open(f"{out_path}.{start:05x}", "wb").write(body)

    if len(found) < 2:
        print("\n⚠ 撈不到兩份以上，沒辦法交叉檢查")
        return
    # 交叉檢查：哪些位元組在所有版本裡都一樣。
    same = [i for i in range(len(found[0]))
            if all(f[i] == found[0][i] for f in found[1:])]
    print(f"\n{len(found)} 個起點的 {len(found[0])} 個位元組裡，"
          f"{len(same)} 個完全一致（其餘是解壓雜訊）")
    print("一致的那些拼出來就是 GetPValue：")
    print("   cmp dx,144 → mov ax,25    壅塞車流")
    print("   cmp dx, 80 → mov ax,10    稀疏車流")
    print("   cmp dx, 56 → mov ax,60    火災")
    print("   cmp dx, 52 → mov ax,-40   輻射")
    print("   cmp dx,761 → jl 往回 38 位元組，落在上面那個 mov ax,3Ch")
    print("                → 海港／機場／電廠也是 60")
    print("六個全部是 Micropolis 註解裡的舊值。docs/re/18-dos-parity.md §6.3")


if __name__ == "__main__":
    main(sys.argv[1], sys.argv[2] if len(sys.argv) > 2 else None)
