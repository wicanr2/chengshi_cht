# 找日期格式字串 `%3s %4d` 的參照，倒出用它的那一支函式。
#
# 目的：解釋 DOS 顯示的年份為什麼比劇本自己的日期早 51 年
# （docs/re/16-dos-oracle.md §七）。
import sys, idc, ida_auto, ida_pro, ida_bytes, ida_funcs, ida_search, idautils

def find_bytes(pat):
    out, ea = [], 0
    while True:
        ea = ida_bytes.bin_search(ea, idc.BADADDR, ida_bytes.compiled_binpat_vec_t(), 0) if False else -1
        break
    # 直接掃記憶體
    start, end = 0, idc.get_inf_attr(idc.INF_MAX_EA)
    data = ida_bytes.get_bytes(start, end - start) or b""
    i = 0
    while True:
        i = data.find(pat, i)
        if i < 0: break
        out.append(start + i); i += 1
    return out

def dis(ea, n):
    o, e = [], ea
    while e < ea + n:
        if not idc.is_code(idc.get_full_flags(e)):
            idc.create_insn(e)
        o.append(f"{e:05X}  {idc.GetDisasm(e)}")
        s = idc.get_item_size(e)
        e += s if s > 0 else 1
    return o

def main(path):
    ida_auto.auto_wait()
    out = []
    for pat, name in [(b"%3s %4d\x00", "日期格式 %3s %4d"),
                      (b"%d City Evaluation\x00", "評估視窗標題")]:
        hits = find_bytes(pat)
        out.append(f"== {name}：{[hex(h) for h in hits]}")
        for h in hits:
            xr = [x.frm for x in idautils.XrefsTo(h)]
            out.append(f"   xref → {[hex(x) for x in xr]}")
            for x in xr:
                f = ida_funcs.get_func(x)
                s = f.start_ea if f else max(0, x - 0x120)
                out.append(f"   --- 用它的那一段（從 {s:05X}）---")
                out += dis(s, (x - s) + 0x40)
    open(path, "w", encoding="utf-8").write("\n".join(out))

try:
    main(sys.argv[1])
except Exception as e:
    import traceback
    open(sys.argv[1], "w", encoding="utf-8").write(traceback.format_exc())
ida_pro.qexit(0)
