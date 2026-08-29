# 最小探針：數函式、數字串，寫 JSON。
#
# ⚠ **先組完整個字串再一次寫檔。** 用 json.dump 直接寫檔的話，腳本中途
# 掛掉會留下一個「開頭對、後面沒了」的檔案（實測只寫到 21 位元組），
# 而外層的「檔案存在且非空」檢查會判它成功。
import json, sys, idautils, idc, ida_auto, ida_pro, ida_nalt

def main(path):
    ida_auto.auto_wait()
    funcs = list(idautils.Functions())
    out = {
        "n_funcs": len(funcs),
        "first_funcs": [idc.get_func_name(f) for f in funcs[:20]],
        "base": hex(ida_nalt.get_imagebase()),
    }
    try:
        strs = [str(s) for s in idautils.Strings()]
        out["n_strings"] = len(strs)
        out["sample_strings"] = [s for s in strs if len(s) > 6][:40]
    except Exception as e:
        out["strings_error"] = repr(e)
    blob = json.dumps(out, ensure_ascii=False, indent=1)
    with open(path, "w") as f:
        f.write(blob)

try:
    main(sys.argv[1])
except Exception as e:
    with open(sys.argv[1], "w") as f:
        f.write(json.dumps({"error": repr(e)}))
ida_pro.qexit(0)
