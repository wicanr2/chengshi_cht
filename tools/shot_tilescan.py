#!/usr/bin/env python3
# 找出編輯視窗地圖格網上「不是第 0 庫任何一張圖塊」的格子。
#
# 用法：先產圖集 tools/go.sh run ./tools/tileatlas <某個.PGF> workplace/tiles-west.png
#      再 tools/shot_tilescan.py workplace/dosbox/<截圖>.png ...
#
# 為什麼要有這支：逐點比對受城市狀態影響（見 tools/shot_diff.py 的警告），
# 這支只問「有沒有非圖塊的東西疊上來」，對模擬進度免疫。
# 內建正對照：3×3 工具游標的外框一定會被報成 8 格不明（中心格不會）。
# 完整說明見 docs/re/16-dos-oracle.md §九。
# 格網原點量出來是 x=256+16i、y=239+16j（螢幕絕對座標）。
import sys
from PIL import Image
def norm(im):
    px=im.load()
    for y in range(im.size[1]):
        for x in range(im.size[0]):
            r,g,b=px[x,y]; px[x,y]=(round(r/85)*85, round(g/85)*85, round(b/85)*85)
    return im
at=norm(Image.open('workplace/tiles-west.png').convert('RGB'))
tiles={}
for k in range(960):
    ox,oy=(k%30)*16,(k//30)*16
    tiles.setdefault(at.crop((ox,oy,ox+16,oy+16)).tobytes(),k)
for path in sys.argv[1:]:
    sh=Image.open(path).convert('RGB')
    bad=[]; ok=0
    for j in range(16):
        for i in range(32):
            x,y=256+16*i, 239+16*j
            if x+16>192+640 or y+16>184+350: continue
            if sh.crop((x,y,x+16,y+16)).tobytes() in tiles: ok+=1
            else: bad.append((i,j,x,y))
    print(f'{path.split("/")[-1]:28s} 命中 {ok:4d} 格，不明 {len(bad):3d} 格 {[(i,j) for i,j,_,_ in bad][:14]}')
