# 把地形編輯器介面相關的函式整段倒成反組譯。
#
# ⚠ 這支曾經寫出 0 位元組的空檔而且沒有任何錯誤訊息，所以現在**開檔就先寫現場**
# （argv、函式解析結果），任何一步失敗都看得出來停在哪。
# ida-pro-9.4.md：唯一可信的訊號是輸出檔本身。
import sys
import traceback

import ida_auto
import ida_funcs
import ida_name
import ida_pro
import idc

FUNCS = ["sub_11402", "sub_10A0A", "sub_106BA"]

out = open(sys.argv[1] if len(sys.argv) > 1 else "/work/te-funcs.txt", "w")
out.write("argv=%r\n" % (sys.argv,))
out.flush()

try:
    ida_auto.auto_wait()
    out.write("auto_wait 完成\n")
    out.flush()
    for n in FUNCS:
        ea = ida_name.get_name_ea(idc.BADADDR, n)
        out.write("%s -> ea=%s\n" % (n, "BADADDR" if ea == idc.BADADDR else "%06X" % ea))
        if ea == idc.BADADDR:
            continue
        f = ida_funcs.get_func(ea)
        if f is None:
            out.write("  不在函式裡\n")
            continue
        out.write("===== %s  %06X..%06X\n" % (ida_funcs.get_func_name(f.start_ea), f.start_ea, f.end_ea))
        e = f.start_ea
        while e < f.end_ea:
            out.write("%06X  %s\n" % (e, idc.GetDisasm(e)))
            sz = idc.get_item_size(e)
            e += sz if sz > 0 else 1
        out.write("\n")
        out.flush()
except Exception:
    out.write("例外：\n" + traceback.format_exc())
out.close()
ida_pro.qexit(0)
