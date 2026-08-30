#!/usr/bin/env python3
# 逐張比兩次 DOSBox 執行的截圖，只看遊戲區。
#
# 用法：tools/shot_diff.py <前綴A> <前綴B>
#
# ⚠ 這支的敏感度有上限：同檔同腳本跑兩次的噪音底線是兩千多個像素，
# 而一個 16×16 圖塊最多 256 個像素——所以它判不了單格繪製。
# 判單格要用 tools/shot_tilescan.py。坑的清單見 docs/re/16-dos-oracle.md §九。
import sys, glob, os
from PIL import Image
a_pre, b_pre = sys.argv[1], sys.argv[2]
X0,Y0,X1,Y1 = 192,184,832,534
for pa in sorted(glob.glob(f'workplace/dosbox/{a_pre}-*.png')):
    tag = os.path.basename(pa)[len(a_pre)+1:]
    pb = f'workplace/dosbox/{b_pre}-{tag}'
    if not os.path.exists(pb):
        print(f'{tag:16s} 只有 {a_pre} 有'); continue
    ia = Image.open(pa).convert('RGB').crop((X0,Y0,X1,Y1))
    ib = Image.open(pb).convert('RGB').crop((X0,Y0,X1,Y1))
    da, db = ia.load(), ib.load()
    n = 0; box=[9999,9999,-1,-1]
    for y in range(Y1-Y0):
        for x in range(X1-X0):
            if da[x,y]!=db[x,y]:
                n += 1
                box[0]=min(box[0],x); box[1]=min(box[1],y)
                box[2]=max(box[2],x); box[3]=max(box[3],y)
    print(f'{tag:16s} 差 {n:7d} 像素' + (f'  範圍 x{box[0]}-{box[2]} y{box[1]}-{box[3]}' if n else ''))
