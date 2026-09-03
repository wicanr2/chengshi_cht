# 地形編輯器的介面：從字串反查是誰畫的，再把那些函式整段倒出來。
#
# 為什麼不 grep `.asm`：`.asm` 是攤平的文字，沒有交叉參考圖。想知道
# 「這個字串被誰用」只能從資料庫問（ida-pro-9.4.md 的「最重要的一課」）。
#
# 輸出兩個檔：<out> 是 JSON（字串位址、xref、所屬函式），
# <out>.asm 是那些函式的反組譯全文。
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

TARGETS = [
    "Terrain Creation Parameters",
    "MAXIS SimCity Terrain Editor",
    "NEW GAME",
    "Now terraforming",
    "Smoothing...",
    "Enter Game Year:",
    "of Trees   of Lakes  Curviness",
    "   Go   ",
    " Cancel ",
    "Easy",
    "Medium",
    "Hard",
    "EXIT",
    "cegated.pgf",
]


def find_string_eas():
    """字串位址：先問 IDA 的字串表，找不到的再自己掃位元組。"""
    found = {}
    for s in idautils.Strings():
        try:
            txt = str(s)
        except Exception:
            continue
        for t in TARGETS:
            if t in txt and t not in found:
                found[t] = s.ea
    missing = [t for t in TARGETS if t not in found]
    for t in missing:
        pat = t.encode("latin-1")
        for seg in idautils.Segments():
            s = ida_segment.getseg(seg)
            ea = ida_bytes.find_bytes(pat, range_start=s.start_ea, range_end=s.end_ea)
            if ea not in (idc.BADADDR, None):
                found[t] = ea
                break
    return found


def func_of(ea):
    f = ida_funcs.get_func(ea)
    if f is None:
        return None
    return {"start": f.start_ea, "end": f.end_ea, "name": ida_funcs.get_func_name(f.start_ea)}


def disasm(start, end):
    out, ea = [], start
    while ea < end:
        out.append("%06X  %s" % (ea, idc.GetDisasm(ea)))
        sz = idc.get_item_size(ea)
        ea += sz if sz > 0 else 1
    return out


def main(out_path):
    ida_auto.auto_wait()
    eas = find_string_eas()

    report = {
        "input_sha256": ida_nalt.retrieve_input_file_sha256().hex(),
        "funcs_total": len(list(idautils.Functions())),
        "strings": {},
    }
    funcs = {}
    for text, ea in sorted(eas.items(), key=lambda kv: kv[1]):
        refs = []
        for x in idautils.XrefsTo(ea):
            fn = func_of(x.frm)
            refs.append({
                "from": "%06X" % x.frm,
                "type": x.type,
                "iscode": bool(x.iscode),
                "insn": idc.GetDisasm(x.frm),
                "func": fn["name"] if fn else None,
            })
            if fn:
                funcs[fn["start"]] = fn
        report["strings"][text] = {"ea": "%06X" % ea, "xrefs": refs}

    report["funcs_dumped"] = [
        {"name": f["name"], "start": "%06X" % f["start"], "end": "%06X" % f["end"]}
        for f in sorted(funcs.values(), key=lambda f: f["start"])
    ]
    with open(out_path, "w") as f:
        json.dump(report, f, ensure_ascii=False, indent=1)

    lines = []
    for f in sorted(funcs.values(), key=lambda f: f["start"]):
        lines.append("===== %s  %06X..%06X" % (f["name"], f["start"], f["end"]))
        lines += disasm(f["start"], f["end"])
        lines.append("")
    with open(out_path + ".asm", "w") as f:
        f.write("\n".join(lines))


try:
    main(sys.argv[1])
except Exception as e:
    with open(sys.argv[1], "w") as f:
        json.dump({"error": repr(e)}, f)
ida_pro.qexit(0)
