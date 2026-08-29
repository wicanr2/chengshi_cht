# 一次倒出多段反組譯。
#
# ⚠ 不要用 shell 迴圈跑 N 次 idat：對同一個 .i64 連續開關會把資料庫弄壞
# （"Failed to initialize IDA as library (error code 4)"）。
# ⚠ raw binary 預設 32 位元定址，要先把節區設成 16 位元，否則整段解成 dq。
import sys, idc, ida_auto, ida_pro, ida_segment, ida_bytes

def dump(start, n):
    ea, out = start, []
    while ea < start + n:
        ida_bytes.del_items(ea, ida_bytes.DELIT_SIMPLE, 1)
        idc.create_insn(ea)
        sz = idc.get_item_size(ea)
        out.append(f"{ea:05X}  {idc.GetDisasm(ea)}")
        ea += sz if sz > 0 else 1
    return out

def main(path, specs):
    ida_auto.auto_wait()
    seg = ida_segment.getseg(int(specs[0].split(':')[0], 0))
    if seg is not None:
        ida_segment.set_segm_addressing(seg, 0)
    out = []
    for sp in specs:
        a, n = sp.split(':')
        out.append(f"===== {a} +{n}")
        out += dump(int(a, 0), int(n, 0))
    open(path, "w").write("\n".join(out))

try:
    main(sys.argv[1], sys.argv[2:])
except Exception as e:
    open(sys.argv[1], "w").write(repr(e))
ida_pro.qexit(0)
