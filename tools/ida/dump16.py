# 把 raw binary 當 16 位元程式碼反組譯一段。
#
# ⚠ IDA 載入 raw binary 預設是 32 位元定址，直接 create_insn 會解錯
# （症狀是整段變成 dq）。要先把節區設成 16 位元。
import sys, idc, ida_auto, ida_pro, ida_segment, ida_bytes, ida_funcs

def main(path, start, n):
    ida_auto.auto_wait()
    seg = ida_segment.getseg(start)
    if seg is not None:
        ida_segment.set_segm_addressing(seg, 0)   # 0 = 16-bit
    ea, out = start, []
    while ea < start + n:
        ida_bytes.del_items(ea, ida_bytes.DELIT_SIMPLE, 1)
        idc.create_insn(ea)
        sz = idc.get_item_size(ea)
        out.append(f"{ea:05X}  {idc.GetDisasm(ea)}")
        ea += sz if sz > 0 else 1
    open(path, "w").write("\n".join(out))

try:
    main(sys.argv[1], int(sys.argv[2], 0), int(sys.argv[3], 0))
except Exception as e:
    open(sys.argv[1], "w").write(repr(e))
ida_pro.qexit(0)
