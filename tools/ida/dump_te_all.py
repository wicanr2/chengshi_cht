# 地形編輯器：一次把要看的東西全部倒出來。
#
# ⚠ **不要拆成多支腳本連續跑。** 對同一個 `.i64` 連續開關會把資料庫弄壞，
# 症狀是第二次之後 `auto_wait()` 直接被殺掉——沒有例外、沒有訊息，
# 只有一個寫到一半的輸出檔（tools/ida.sh 的檔頭也記著這件事）。
#
# 產出：
#   <out>          JSON：字串位址、指標表、命中的函式
#   <out>.asm      那些函式的反組譯全文
#   <out>.hex      字串叢集前後的原始位元組
import json
import sys
import traceback

import ida_auto
import ida_bytes
import ida_funcs
import ida_nalt
import ida_pro
import ida_segment
import idautils
import idc

# 上一輪用 16 位元位移反查出來的：畫參數對話框的是 sub_11402，
# 主選單是 sub_10A0A，另有 sub_106BA 命中 NEW GAME 與 terraforming。
FUNCS = ["sub_11402", "sub_10A0A", "sub_106BA"]
DATA_LO, DATA_HI = 0x059140, 0x0592C0


def disasm(start, end):
    out, ea = [], start
    while ea < end:
        out.append("%06X  %s" % (ea, idc.GetDisasm(ea)))
        sz = idc.get_item_size(ea)
        ea += sz if sz > 0 else 1
    return out


def hexdump(lo, hi):
    out, ea = [], lo
    while ea < hi:
        n = min(16, hi - ea)
        bs = ida_bytes.get_bytes(ea, n) or b""
        out.append("%06X  %-47s  %s" % (
            ea, " ".join("%02x" % b for b in bs),
            "".join(chr(b) if 32 <= b < 127 else "." for b in bs)))
        ea += n
    return out


def main(out_path):
    ida_auto.auto_wait()
    rep = {"input_sha256": ida_nalt.retrieve_input_file_sha256().hex(), "funcs": {}}
    asm = []
    for n in FUNCS:
        ea = ida_name_ea(n)
        if ea == idc.BADADDR:
            rep["funcs"][n] = {"error": "找不到"}
            continue
        f = ida_funcs.get_func(ea)
        if f is None:
            rep["funcs"][n] = {"error": "不在函式裡"}
            continue
        callees = set()
        e = f.start_ea
        while e < f.end_ea:
            for x in idautils.CodeRefsFrom(e, 0):
                cf = ida_funcs.get_func(x)
                if cf is not None and cf.start_ea != f.start_ea:
                    callees.add(ida_funcs.get_func_name(cf.start_ea))
            e += max(1, idc.get_item_size(e))
        rep["funcs"][n] = {
            "start": "%06X" % f.start_ea,
            "end": "%06X" % f.end_ea,
            "size": f.end_ea - f.start_ea,
            "callees": sorted(c for c in callees if c),
        }
        asm.append("===== %s  %06X..%06X" % (n, f.start_ea, f.end_ea))
        asm += disasm(f.start_ea, f.end_ea)
        asm.append("")

    seg = ida_segment.getseg(DATA_LO)
    rep["data_segment"] = {
        "name": ida_segment.get_segm_name(seg) if seg else None,
        "start": "%06X" % (seg.start_ea if seg else 0),
    }
    with open(out_path, "w") as f:
        json.dump(rep, f, ensure_ascii=False, indent=1)
    with open(out_path + ".asm", "w") as f:
        f.write("\n".join(asm))
    with open(out_path + ".hex", "w") as f:
        f.write("\n".join(hexdump(DATA_LO, DATA_HI)))


def ida_name_ea(n):
    import ida_name
    return ida_name.get_name_ea(idc.BADADDR, n)


try:
    main(sys.argv[1])
except Exception:
    with open(sys.argv[1], "w") as f:
        f.write(traceback.format_exc())
ida_pro.qexit(0)
