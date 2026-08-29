"""把 oracle 倒出來的地圖（ZA／ZB 那一行）寫成測試用的 CSV。

用法：
    python3 tools/oracle/extract_map.py <result.json> <ZA|ZB> <輸出.csv> <說明>

⚠ `sim Tile` 回的是**有號 int16**，負數要 `& 0xFFFF` 還原。
不還原的話比對會冒出一堆「差很多格」，而且看起來像地圖真的不一樣。
"""

import json
import sys

WX, WY = 120, 100


def main():
    path, tag, out, note = sys.argv[1:5]
    d = json.load(open(path))
    vals = None
    for r in d["results"]:
        for line in r.get("out", []):
            if line.startswith(tag + " "):
                _, n, body = line.split(" ", 2)
                vals = [int(v) & 0xFFFF for v in body.split(",")]
                if len(vals) != int(n) or len(vals) != WX * WY:
                    raise SystemExit(f"{tag} 有 {len(vals)} 格，宣告 {n}，應為 {WX * WY}")
    if vals is None:
        raise SystemExit(f"找不到 {tag}")

    with open(out, "w") as f:
        f.write(f"# {note}\n")
        f.write(f"# 取法：{path}，{tag}。\n")
        for y in range(WY):
            f.write(",".join(str(vals[y * WX + x]) for x in range(WX)) + "\n")
    print("寫入", out)


if __name__ == "__main__":
    main()
