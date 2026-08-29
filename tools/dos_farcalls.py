#!/usr/bin/env python3
# 解出映像裡所有 far call 的目標，並列出誰呼叫誰。
#
# 為什麼可行：解開的映像就是**載入時的記憶體影像**，而 far call 的
# `9A off seg` 裡那個 `seg` 是**相對載入節區**的值（載入器最後才把載入段
# 加上去）。所以
#
#     目標線性位址 ＝ seg × 16 ＋ off
#
# 驗證：全映像 2118 個這樣的 far call，405 個相異目標，其中 181 個正好落在
# `55 8B EC` 函式序言上——遠高於隨機命中，所以定址模型是對的。
#
# 這解掉了「呼叫是 far call、節區值載入時才填、靜態搜不到」那個結。
#
#   python3 tools/dos_farcalls.py workplace/ida/image.bin            # 統計
#   python3 tools/dos_farcalls.py workplace/ida/image.bin 0xCF51     # 誰呼叫它
import re, sys, collections


def farcalls(img):
    """回傳 [(呼叫點, 目標線性位址)]。"""
    out = []
    for i in range(len(img) - 5):
        if img[i] != 0x9A:
            continue
        off = img[i + 1] | (img[i + 2] << 8)
        seg = img[i + 3] | (img[i + 4] << 8)
        lin = seg * 16 + off
        if lin < len(img):
            out.append((i, lin))
    return out


def push_const(img, site, back=12):
    """往回找呼叫前壓進去的常數（mov ax,imm/push ax 或 push imm8）。"""
    ctx = img[max(0, site - back):site]
    for k in range(len(ctx) - 3, -1, -1):
        if ctx[k] == 0xB8 and k + 3 < len(ctx) and ctx[k + 3] == 0x50:
            return ctx[k + 1] | (ctx[k + 2] << 8)
        if ctx[k] == 0x6A:
            return ctx[k + 1]
    return None


def main(path, target=None):
    img = open(path, "rb").read()
    calls = farcalls(img)
    if target is None:
        tally = collections.Counter(t for _, t in calls)
        pro = set(m.start() for m in re.finditer(rb"\x55\x8b\xec", img))
        print(f"far call {len(calls)} 個，相異目標 {len(tally)} 個，"
              f"落在函式序言的 {sum(1 for t in tally if t in pro)} 個")
        for t, n in tally.most_common(15):
            print(f"   {t:#08x}  被呼叫 {n} 次")
        return
    sites = [s for s, t in calls if t == target]
    print(f"呼叫 {target:#x} 的地方：{len(sites)} 個")
    for s in sites:
        print(f"   {s:#08x}  前置常數 {push_const(img, s)}")


if __name__ == "__main__":
    main(sys.argv[1], int(sys.argv[2], 0) if len(sys.argv) > 2 else None)
