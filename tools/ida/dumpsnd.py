# 倒出六支「呼叫 PlaySound 但還沒認出來」的函式（docs/re/16-dos-oracle.md §五之四）。
# 範圍是拿 image16.bin 往前找 55 8B EC 的 prologue、往後找下一個 prologue 算出來的。
import sys, idc, ida_auto, ida_pro, ida_segment, ida_bytes

RANGES = [(0x14F8F, 0x2F7), (0x1E2FD, 0x27F), (0x1E5DE, 0x2BC),
          (0x1EC32, 0x130), (0x1ED62, 0x183), (0x1F808, 0x150)]

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
