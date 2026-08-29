# 倒出一段位址的反組譯，找解壓常式用。
import json, sys, idc, ida_auto, ida_pro, ida_bytes, ida_funcs

def main(path, start, n):
    ida_auto.auto_wait()
    ea, out = start, []
    while ea < start + n:
        if not idc.is_code(idc.get_full_flags(ea)):
            idc.create_insn(ea)
        d = idc.GetDisasm(ea)
        out.append(f"{ea:05X}  {d}")
        sz = idc.get_item_size(ea)
        ea += sz if sz > 0 else 1
    with open(path, "w") as f:
        f.write("\n".join(out))

try:
    main(sys.argv[1], int(sys.argv[2], 0), int(sys.argv[3], 0))
except Exception as e:
    with open(sys.argv[1], "w") as f:
        f.write(repr(e))
ida_pro.qexit(0)
