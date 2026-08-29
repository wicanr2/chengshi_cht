# 用 16 份 DOS 存檔反推「DOS 版的汙染權重是新的還是舊的」。
#
# 結論寫在 docs/re/18-dos-parity.md §6：Micropolis 的 s_scan.c:257 GetPValue()
# 把每一個被改過的權重的**舊值留在註解裡**（`return (/* 25 */ 75)`），
# 而 DOS 1.10 是 1991 年的建置、Micropolis 是 2008 年釋出的。
# 三十二種「每個權重取新或取舊」的組合裡，車流舊＋輻射舊＋海港電廠新
# 的誤差最小（4.06，全用現行值是 14.6）。
#
#   tools/go.sh 不用；這支是純 python，直接跑
#   python3 tools/pollution_weight_fit.py
#
# 先資料後反組譯（CLAUDE.md §2.4）：SIMCITY.EXE 是打包過的，IDA 只解出
# 三個函式，要拆得先做記憶體 dump。而每份存檔都提供一組
# （地圖, DOS 自己算的汙染均值），16 組足以判別幾個候選公式。
import struct, itertools

LOMASK=1023; RUBBLE=44; RADTILE=52; FIREBASE=56; ROADBASE=64
LTRFBASE=80; HTRFBASE=144; POWERBASE=208; LASTIND=620
PORTBASE=693; LASTPOWERPLANT=760
HW, HH = 60, 50

def load(path):
    raw=open(path,'rb').read()
    body=(raw[128:128+3120]+raw[3264:]).ljust(27120,b'\0') if len(raw)==27248 else raw
    m=struct.unpack('>12000H', body[3120:3120+24000])
    return [v & LOMASK for v in m]

def pval(loc, w):
    if loc < POWERBASE:
        if loc >= HTRFBASE: return w['heavy']
        if loc >= LTRFBASE: return w['light']
        if loc < ROADBASE:
            if loc > FIREBASE: return w['fire']
            if loc >= RADTILE: return w['rad']
        return 0
    if loc <= LASTIND: return 0
    if loc < PORTBASE: return w['ind']
    if loc <= LASTPOWERPLANT: return w['port']
    return 0

def smooth(t):
    o=[[0]*HH for _ in range(HW)]
    for x in range(HW):
        for y in range(HH):
            z=0
            if x>0: z+=t[x-1][y]
            if x<HW-1: z+=t[x+1][y]
            if y>0: z+=t[x][y-1]
            if y<HH-1: z+=t[x][y+1]
            z=(z+t[x][y])>>2
            o[x][y]=min(z,255)
    return o

def average(lo, w, smooths=2):
    tem=[[0]*HH for _ in range(HW)]
    for x in range(HW):
        for y in range(HH):
            lvl=0
            for mx in (x*2, x*2+1):
                for my in (y*2, y*2+1):
                    loc=lo[mx*100+my]
                    if loc==0 or loc<RUBBLE: continue
                    lvl+=pval(loc,w)
            tem[x][y]=min(lvl,255)
    for _ in range(smooths):
        tem=smooth(tem)
    tot=num=0
    for x in range(HW):
        for y in range(HH):
            z=tem[x][y]
            if z:
                num+=1; tot+=z
    return tot//num if num else 0

names=[f'{t}{n}' for t in ('scen','run') for n in range(1,9)]
dos, maps = {}, {}
for f in names:
    raw = open(f'workplace/dosbox/save/{f}.cty', 'rb').read()
    dos[f] = struct.unpack('>128h', raw[3008:3008+256])[14]
    maps[f] = load(f'workplace/dosbox/save/{f}.cty')

# 每個權重的「舊值（原始碼註解裡的）／現行值」。
OPTS = {'heavy': (25, 75), 'light': (10, 50), 'fire': (60, 90),
        'rad': (-40, 255), 'port': (60, 100)}
KEYS = list(OPTS)

def mae(w):
    errs = [average(maps[f], w) - dos[f] for f in names]
    return sum(abs(e) for e in errs) / len(errs), errs

res = []
for bits in itertools.product((0, 1), repeat=len(KEYS)):
    w = dict(ind=50)
    for k, b in zip(KEYS, bits):
        w[k] = OPTS[k][b]
    m, errs = mae(w)
    res.append((m, bits, errs))
res.sort()
print(f"{'權重組合':44s} {'平均絕對誤差':>10s}")
for m, bits, errs in res:
    lbl = ' '.join(f"{k}={'新' if b else '舊'}" for k, b in zip(KEYS, bits))
    print(f"{lbl:44s} {m:10.2f}")
print()
m, errs = res[0][0], res[0][2]
print("最好的那一組逐份誤差（我們 − 原版）：")
for f, e in zip(names, errs):
    print(f"  {f:8s} 原版 {dos[f]:3d}  差 {e:+4d}")
