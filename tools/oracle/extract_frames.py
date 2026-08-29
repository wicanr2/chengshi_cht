"""把逐 frame 對拍的 oracle 輸出轉成測試資料。

用法：
    python3 tools/oracle/extract_frames.py <frame-parity.json> <輸出目錄>

產出：
    cp0.csv    起始地圖（跑第一個 frame 之前）
    cpend.diff 最後一個 frame 之後的地圖，存成相對 cp0 的差異
    meta.json  起始與結束的 Fcycle／Scycle／資金，起始的四個亂數讀數
    frames.csv 每個 frame 一列：`i,scycle,rvalve,cvalve,ivalve,draws`
    prestate.csv 載入之前那幾張**不會被 DoSimInit 重算**的衍生陣列
               （每列 `名稱,值,值,…`）。只有劇本版有——原版那個行程開機時
               已經跑過一座隨機城市，那些殘值會影響載入時第一次 MapScan。

⚠ **每個 frame 存的是抽樣次數，不是四個亂數讀數。** 兩者等價（LCG 的
狀態由起點與步數唯一決定），但次數只要幾個位元組，而且它就是對拍的判準。
換算在這裡做：`draws = LCG 距離(上一個檢查點的狀態, 這個檢查點的狀態) − 4`，
那個 4 是原版自己那四次 `sim Rand`。

⚠ `sim Tile` 回的是**有號 int16**，負數要 `& 0xFFFF` 還原。
"""

import json
import os
import sys

WX, WY = 120, 100

A, C, MASK = 1103515245, 12345, 0xFFFFFF


def recover_state(outs):
    """從連續的 Rand16 輸出反推 24 位元狀態（同 internal/sim RecoverState）。"""
    cands = [(outs[0] << 8) | lo for lo in range(256)]
    for want in outs[1:]:
        cands = [n for n in ((c * A + C) & MASK for c in cands) if (n >> 8) == want]
        if not cands:
            raise SystemExit(f"反推不出狀態：{outs}")
    if len(cands) != 1:
        raise SystemExit(f"狀態不唯一：{outs}")
    return cands[0]


def lcg_distance(a, b, limit=5_000_000):
    n, s = 0, a
    while n < limit and s != b:
        s = (s * A + C) & MASK
        n += 1
    if s != b:
        raise SystemExit("距離超過上限")
    return n


def find_line(results, tag):
    for r in results:
        for ln in r["out"]:
            if ln.startswith(tag + " "):
                return ln
    raise SystemExit(f"輸出裡找不到 {tag}")


def parse_map(ln, tag):
    body = ln[len(tag) + 1:]
    n, _, vals = body.partition(" ")
    v = [int(x) & 0xFFFF for x in vals.split(",")]
    if len(v) != int(n) or len(v) != WX * WY:
        raise SystemExit(f"{tag} 的格數不對：{len(v)}")
    # oracle 是 y 外層、x 內層
    m = [[0] * WX for _ in range(WY)]
    for i, val in enumerate(v):
        m[i // WX][i % WX] = val
    return m


def main():
    src, out = sys.argv[1], sys.argv[2]
    d = json.load(open(src))
    res = d["results"]
    os.makedirs(out, exist_ok=True)

    m0 = parse_map(find_line(res, "CP0"), "CP0")
    try:
        mE = parse_map(find_line(res, "CPEND"), "CPEND")
    except SystemExit:
        mE = m0  # 短版沒有跑到終點，不倒終點地圖

    with open(f"{out}/cp0.csv", "w") as fh:
        for row in m0:
            fh.write(",".join(str(v) for v in row) + "\n")
    with open(f"{out}/cpend.diff", "w") as fh:
        for y in range(WY):
            for x in range(WX):
                if m0[y][x] != mE[y][x]:
                    fh.write(f"{x},{y},{mE[y][x]}\n")

    pres, posts = [], []
    for r in res:
        for ln in r["out"]:
            if ln.startswith("POST"):
                tag, _, rest = ln.partition(" ")
                _, _, vals = rest.partition(" ")
                posts.append((tag[4:], vals))
            if ln.startswith("PRE") and not ln.startswith("PRE "):
                tag, _, rest = ln.partition(" ")
                _, _, vals = rest.partition(" ")
                pres.append((tag[3:], vals))
    if pres:
        with open(f"{out}/prestate.csv", "w") as fh:
            fh.write("# 載入之前的衍生陣列殘值（名稱,值,…）\n")
            for name, vals in pres:
                fh.write(f"{name},{vals}\n")

    if posts:
        with open(f"{out}/poststate.csv", "w") as fh:
            fh.write("# 載入完成（DoSimInit 跑完）當下的衍生陣列（名稱,值,…）\n")
            for name, vals in posts:
                fh.write(f"{name},{vals}\n")

    spr = None
    for r in res:
        for ln in r["out"]:
            if ln.startswith("SPR "):
                spr = ln[4:]
    spg = None
    for r in res:
        for ln in r["out"]:
            if ln.startswith("SPG "):
                spg = ln.split()[1:]
    if spr is not None:
        cyc, _, rest = spr.partition(" ; ")
        with open(f"{out}/sprites.csv", "w") as fh:
            fh.write("# globals 行：Cycle absDist CrashX CrashY\n"
                     "# gidx 行：GlobalSprites 每一型指到第幾個節點（−1 ＝ NULL）\n"
                     "# 其餘每行一個節點，**含死掉的**，順序就是串列的實體順序：\n"
                     "# type,frame,x,y,orig_x,orig_y,dest_x,dest_y,count,"
                     "sound_count,dir,new_dir,step,flag,control,turn,accel,"
                     "speed,named\n")
            fh.write("globals," + ",".join(cyc.split()) + "\n")
            if spg:
                fh.write("gidx," + ",".join(spg) + "\n")
            for one in rest.split(" ; "):
                one = one.strip()
                if one:
                    fh.write(",".join(one.split()) + "\n")

    steps = []
    for r in res:
        for ln in r["out"]:
            if ln.startswith("S "):
                head, _, rest = ln.partition(" ; ")
                steps.append((head.split()[1], rest))
    if steps:
        with open(f"{out}/sprite-frames.csv", "w") as fh:
            fh.write("# 每個 frame 之後的精靈狀態：frame,隻數,"
                     "然後每隻 18 個欄位（同 sprites.csv）\n")
            for i, rest in steps:
                ones = [o.strip() for o in rest.split(" ; ") if o.strip()]
                fh.write(",".join([i, str(len(ones))] +
                                  [v for o in ones for v in o.split()]) + "\n")

    maps = []
    for r in res:
        for ln in r["out"]:
            if ln.startswith("MM "):
                _, i, rest = ln.split(" ", 2)
                maps.append((i, parse_map("MM " + rest, "MM")))
    if maps:
        with open(f"{out}/map-frames.csv", "w") as fh:
            fh.write("# 每 N 個 frame 的地圖檢查點，存成相對 cp0 的差異：\n"
                     "# frame,x,y,圖塊  （一格一行）\n")
            for i, mm in maps:
                for y in range(WY):
                    for x in range(WX):
                        if mm[y][x] != m0[y][x]:
                            fh.write(f"{i},{x},{y},{mm[y][x]}\n")

    init = find_line(res, "INIT").split()
    r0 = None
    for r in res:
        for ln in r["out"]:
            if ln.startswith("R0 "):
                r0 = ln.split()
    pre = None
    for r in res:
        for ln in r["out"]:
            if ln.startswith("PRE "):
                pre = int(ln.split()[1])
    end = None
    for r in res:
        for ln in r["out"]:
            if ln.startswith("END "):
                end = ln.split()
    # `MH i 雜湊` 是逐 frame 的地圖雜湊，長版用獨立一行送（F 那一行的欄位
    # 是靠數量分辨版面的，再加欄位會打亂）。
    maphash = {}
    for r in res:
        for ln in r["out"]:
            if ln.startswith("MH "):
                q = ln.split()
                maphash[int(q[1])] = int(q[2])
    frames = []
    for r in res:
        for ln in r["out"]:
            if ln.startswith("FS "):
                # 短版：FS i scycle rv cv iv state sf mo
                q = ln.split()
                frames.append({
                    "i": int(q[1]), "scycle": int(q[2]),
                    "valves": [int(x) for x in q[3:6]],
                    "rands": None, "state": int(q[6]),
                    "prob": None, "vote": None,
                    "fstat": [int(q[7]), int(q[8])],
                    # sim SpriteDraws：每一型精靈各抽幾次（索引就是型別編號）
                    "sdraws": [int(x) for x in q[9:18]] if len(q) > 17 else None,
                    # sim MapHash：整張地圖的 FNV-1a。地圖偏掉但抽樣次數
                    # 正常的情況（例如怪獸拆房子）只有它抓得到。
                    "maphash": int(q[18]) if len(q) > 18 else None,
                })
                continue
            if not ln.startswith("F "):
                continue
            p = ln.split()
            frames.append({
                "i": int(p[1]),
                "fcycle": int(p[2]),
                "scycle": int(p[3]),
                "valves": [int(x) for x in p[4:7]],
                "rands": [int(x) for x in p[7:11]],
                "state": None,
                # sim Problems：CityScore CityYes CityNo | 表×10 | 票×10 | taken×10
                "prob": ([int(x) for x in p[11:14]] +
                         [int(x) for x in p[15:22]]) if len(p) > 20 else None,
                # sim VoteStats：投票迴圈抽樣、市民投票抽樣、迭代、成功
                # 兩種版面：完整版（Problems ＋ VoteStats ＋ FrameStats）
                # 與精簡版（只有 FrameStats）。用 token 數分辨。
                "vote": [int(x) for x in p[-6:-2]] if len(p) > 42 else None,
                # sim FrameStats：SimFrame（規則）與 MoveObjects（精靈）各抽幾次
                "fstat": [int(x) for x in p[-2:]] if len(p) in (13, 53) else None,
                "sdraws": None,
                "maphash": maphash.get(int(p[1])),
            })
    # 把四個亂數讀數換算成抽樣次數。RecoverState 回的是**讀完四次之後**
    # 的狀態，而原版讀完就直接跑下一個 frame，所以相鄰兩個檢查點的距離
    # 是「這個 frame 的抽樣次數 + 4」。
    if r0 is not None:
        prev = recover_state([int(x) for x in r0[1:5]])
    else:
        # 短版直接讀 `sim RandState`，不抽四次來反推。
        prev = int(find_line(res, "R0S").split()[1])
    rows = []
    for fr in frames:
        if fr["rands"] is not None:
            cur = recover_state(fr["rands"])
            d = lcg_distance(prev, cur) - 4  # 原版自己那四次 sim Rand
        else:
            cur = fr["state"]
            d = lcg_distance(prev, cur)
        if d < 0:
            raise SystemExit(f"第 {fr['i']} 個 frame 的距離算出 {d}，不可能")
        row = [fr["i"], fr["scycle"], fr["valves"][0], fr["valves"][1],
               fr["valves"][2], d, cur]
        if fr["prob"]:
            row += fr["prob"]
        if fr["vote"]:
            row += fr["vote"]
        if fr["fstat"]:
            row += fr["fstat"]
        if fr.get("sdraws"):
            row += fr["sdraws"]
        if fr.get("maphash") is not None:
            row.append(fr["maphash"])
        rows.append(tuple(row))
        prev = cur
    with open(f"{out}/frames.csv", "w") as fh:
        fh.write("# i,scycle,rvalve,cvalve,ivalve,draws,state"
                 "[,cityscore,cityyes,cityno,問題表0..6"
                 ",投票抽樣,市民抽樣,迭代,成功][,規則抽樣,精靈抽樣"
                 "[,逐型精靈抽樣0..8[,地圖雜湊]]]\n")
        for r in rows:
            fh.write(",".join(str(v) for v in r) + "\n")

    meta = {
        "init": {"fcycle": int(init[1]), "scycle": int(init[2]),
                 "funds": int(init[3]),
                 "rands": [int(x) for x in r0[1:5]] if r0 else [],
                 # 短版直接讀狀態，沒有那四次 sim Rand，所以也不必 +4。
                 "r0state": (None if r0 else
                             int(find_line(res, "R0S").split()[1])),
                 "prestate": pre},
        "end": ({"fcycle": int(end[1]), "scycle": int(end[2]),
                 "funds": int(end[3])} if end else None),
    }
    with open(f"{out}/meta.json", "w") as fh:
        json.dump(meta, fh, indent=1)
    print(f"{len(frames)} 個 frame，共 {sum(r[5] for r in rows)} 次抽樣；地圖差 "
          f"{sum(1 for y in range(WY) for x in range(WX) if m0[y][x] != mE[y][x])} 格")


main()
