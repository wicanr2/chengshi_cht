"""比較兩張截圖的地圖區，印出不同像素的**個數**。

整張圖比對沒有用：狀態列的時鐘一直在走，任何兩張都是「不同」。
把右側工具列與下方狀態列切掉，剩下的差異才回答得了
「捲動生效了嗎」「視窗真的開了嗎」。

印個數不印比例，是因為判準的兩端差四個數量級：開一個視窗要蓋掉八成畫面
（約 60 萬個像素），而在一格上拉一條電線可能只改幾十個像素。任何比例
單位都會有一端被取整吃掉——換一種城市外觀（電線比較細、對比比較低），
原本過得了的門檻就會突然過不了，而東西其實有蓋出來。

比對只在遊戲暫停時才有意義：時鐘與煙囪動畫都停著，雜訊才會是 0。
"""

import sys

from PIL import Image

BOX = (0, 0, 1024, 768)  # 地圖區，見 internal/ui 的 viewW／viewH

a = Image.open(sys.argv[1]).convert("RGB").crop(BOX)
b = Image.open(sys.argv[2]).convert("RGB").crop(BOX)
n = sum(1 for p, q in zip(a.getdata(), b.getdata()) if p != q)
print(n)
