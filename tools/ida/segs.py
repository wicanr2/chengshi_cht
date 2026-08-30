import sys, idc, ida_auto, ida_pro, idautils, ida_segment, ida_bytes, ida_funcs
def main(path):
    ida_auto.auto_wait()
    o=[]
    o.append("== 段 ==")
    for s in idautils.Segments():
        sg=ida_segment.getseg(s)
        o.append(f"  {ida_segment.get_segm_name(sg)}  {s:05X}-{sg.end_ea:05X}  類別 {ida_segment.get_segm_class(sg)}  base {sg.sel}")
    o.append("== 含 '%3s' 或 'City Evaluation' 的字串 ==")
    for st in idautils.Strings():
        t=str(st)
        if "%3s" in t or "City Evaluation" in t or "Fiscal Budget" in t:
            xr=[hex(x.frm) for x in idautils.XrefsTo(st.ea)]
            o.append(f"  {st.ea:05X}  {t!r}  xref={xr}")
    open(path,"w",encoding="utf-8").write("\n".join(o))
try: main(sys.argv[1])
except Exception:
    import traceback; open(sys.argv[1],"w",encoding="utf-8").write(traceback.format_exc())
ida_pro.qexit(0)
