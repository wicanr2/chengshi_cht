# 倒出 DAC 播放那一路（docs/re/16-dos-oracle.md §五之五）。
#
# `PlaySample`（0xC9EB）在 `byte_29A7 == 3` 時走這條，而這條屬於映像裡
# segment 24ABh 的一套音效卡驅動。目的是找出取樣率的常數——結果是
# **不在播放這一段**：這裡看到的是 DMA 遮罩、DMA page register、PIC 遮罩
# 與 `int 15h AX=91F0h`，取樣率要往初始化（`_InitSounds`）找。
#
# 範圍是拿 image16.bin 往前找 55 8B EC 的 prologue 算出來的。
# ⚠ 不要用 shell 迴圈跑 N 次 idat：對同一個 .i64 連續開關會把資料庫弄壞。
import sys, idc, ida_auto, ida_pro, ida_segment, ida_bytes

RANGES = [(0xCC20, 0xA0), (0xC990, 0x5C), (0xDBDE, 0x40), (0xD5D2, 0x50), (0xD622, 0x18)]


def dump(start, n):
    ea, out = start, []
    while ea < start + n:
        ida_bytes.del_items(ea, ida_bytes.DELIT_SIMPLE, 1)
        idc.create_insn(ea)
        sz = idc.get_item_size(ea)
        out.append(f"{ea:05X}  {idc.GetDisasm(ea)}")
        ea += sz if sz > 0 else 1
    return out


def main(path):
    ida_auto.auto_wait()
    seg = ida_segment.getseg(RANGES[0][0])
    if seg is not None:
        ida_segment.set_segm_addressing(seg, 0)
    out = []
    for a, n in RANGES:
        out.append(f"===== {a:05X} +{n:X}")
        out += dump(a, n)
    open(path, "w").write("\n".join(out))


try:
    main(sys.argv[1])
except Exception as e:
    open(sys.argv[1], "w").write(repr(e))
ida_pro.qexit(0)
