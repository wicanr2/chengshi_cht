# 找地形編輯器對話框的版面來源。
#
# 上一輪查到字串是被 `dd` 指標表引用的，不是被指令直接引用——16-bit DOS
# 常見的形狀。所以這一輪問三件事：
#   1. 那塊資料在哪個段、位移多少
#   2. 字串叢集前後的原始位元組（版面描述表如果存在，就緊鄰著）
#   3. 誰把那些位移當 16 位元立即數用（那就是畫對話框的程式碼）
import json
import sys

import ida_auto
import ida_bytes
import ida_funcs
import ida_nalt
import ida_pro
import ida_segment
import idautils
import idc

TARGETS = {
    "Easy": 0x05919C,
    "EXIT": 0x0591AD,
    "title": 0x0591B2,
    "NEW GAME": 0x0591CF,
    "terraforming": 0x0591D8,
    "Smoothing": 0x0591E9,
    "ptr_table": 0x0591F6,
    "Enter Game Year": 0x059206,
    "params_title": 0x059220,
    "labels": 0x05925B,
    "Go": 0x05927E,
    "Cancel": 0x059287,
}


def hexdump(start, end):
    out = []
    ea = start
    while ea < end:
        n = min(16, end - ea)
        bs = ida_bytes.get_bytes(ea, n) or b""
        hexs = " ".join("%02x" % b for b in bs)
        txt = "".join(chr(b) if 32 <= b < 127 else "." for b in bs)
        out.append("%06X  %-47s  %s" % (ea, hexs, txt))
        ea += n
    return out


def main(out_path):
    ida_auto.auto_wait()
    rep = {
        "input_sha256": ida_nalt.retrieve_input_file_sha256().hex(),
        "segments": [],
        "targets": {},
    }
    for s in idautils.Segments():
        seg = ida_segment.getseg(s)
        rep["segments"].append({
            "name": ida_segment.get_segm_name(seg),
            "start": "%06X" % seg.start_ea,
            "end": "%06X" % seg.end_ea,
            "bitness": seg.bitness,
        })

    seg = ida_segment.getseg(0x0591B2)
    base = seg.start_ea if seg else 0
    rep["data_segment"] = {
        "name": ida_segment.get_segm_name(seg) if seg else None,
        "start": "%06X" % base,
    }

    # 誰把字串位移當 16 位元立即數用
    for name, ea in TARGETS.items():
        off = ea - base
        hits = []
        pat = bytes([off & 0xFF, (off >> 8) & 0xFF])
        for s in idautils.Segments():
            sg = ida_segment.getseg(s)
            cur = sg.start_ea
            while True:
                cur = ida_bytes.find_bytes(pat, range_start=cur, range_end=sg.end_ea)
                if cur in (idc.BADADDR, None):
                    break
                f = ida_funcs.get_func(cur)
                hits.append({
                    "at": "%06X" % cur,
                    "seg": ida_segment.get_segm_name(sg),
                    "func": ida_funcs.get_func_name(f.start_ea) if f else None,
                    "disasm": idc.GetDisasm(cur),
                })
                cur += 1
                if len(hits) > 40:
                    break
        rep["targets"][name] = {"ea": "%06X" % ea, "offset": "%04X" % off, "hits": hits}

    with open(out_path, "w") as f:
        json.dump(rep, f, ensure_ascii=False, indent=1)

    with open(out_path + ".hex", "w") as f:
        f.write("\n".join(hexdump(0x059140, 0x0592C0)))


try:
    main(sys.argv[1])
except Exception as e:
    with open(sys.argv[1], "w") as f:
        json.dump({"error": repr(e)}, f)
ida_pro.qexit(0)
