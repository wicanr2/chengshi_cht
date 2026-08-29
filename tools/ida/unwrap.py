# 從破解 stub 背後的**原始進入點**開始分析。
#
# SIMCITY.EXE 的 CS:IP 指向的是破解程式的 stub（掛 INT 21h，把防拷判斷
# patch 掉），stub 最後 `retf` 到 `載入段 + 0xE0 : 0`——那才是原版程式。
# IDA 只從檔頭的進入點分析，所以原版那一段整片留在 `db`（3 個函式、1 個字串）。
#
# imagebase 是 0x10000（seg001 的 word_2EA06 在 0x2EA06，而它是 seg001+3，
# 檔頭 e_cs=0x1ea0 → base = 0x2EA00 − 0x1EA00 = 0x10000）。
import json, sys, idautils, idc, ida_auto, ida_pro, ida_funcs, ida_bytes, ida_nalt

def main(path):
    ida_auto.auto_wait()
    before = len(list(idautils.Functions()))
    ep = 0x10000 + 0xE0 * 16          # 原版進入點
    idc.del_items(ep, ida_bytes.DELIT_EXPAND, 0x40)
    ok_code = idc.create_insn(ep) != 0
    ok_func = ida_funcs.add_func(ep)
    ida_auto.auto_wait()
    after = len(list(idautils.Functions()))
    out = {
        "entry": hex(ep),
        "create_insn": ok_code,
        "add_func": bool(ok_func),
        "funcs_before": before,
        "funcs_after": after,
        "disasm": [idc.GetDisasm(ea) for ea in
                   [ep + i for i in range(0, 0x60)] if idc.is_code(idc.get_full_flags(ea))][:24],
        "bytes": " ".join(f"{ida_bytes.get_byte(ep+i):02X}" for i in range(32)),
    }
    with open(path, "w") as f:
        f.write(json.dumps(out, ensure_ascii=False, indent=1))

try:
    main(sys.argv[1])
except Exception as e:
    with open(sys.argv[1], "w") as f:
        f.write(json.dumps({"error": repr(e)}))
ida_pro.qexit(0)
