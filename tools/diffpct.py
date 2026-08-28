"""比較兩張截圖的地圖區，印出不同像素的千分比。

整張圖比對沒有用：狀態列的時鐘一直在走，任何兩張都是「不同」。
把右側工具列與下方狀態列切掉，剩下的差異才回答得了
「捲動生效了嗎」「視窗真的開了嗎」。

用千分比而不是百分比，是因為判準的兩端差三個數量級：開一個視窗要蓋掉
八成畫面，而劃一塊 3×3 的住宅區只動到 1.2% 的像素——取整到百分比之後
是「1」，和雜訊分不開，一塊真的蓋出來的分區會被判成沒蓋。
"""

import sys

from PIL import Image

BOX = (0, 0, 1024, 768)  # 地圖區，見 internal/ui 的 viewW／viewH

a = Image.open(sys.argv[1]).convert("RGB").crop(BOX)
b = Image.open(sys.argv[2]).convert("RGB").crop(BOX)
n = sum(1 for p, q in zip(a.getdata(), b.getdata()) if p != q)
print(round(1000 * n / (BOX[2] * BOX[3])))
