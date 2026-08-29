# 第三方授權

本專案自己的程式碼、文件與譯文採 PolyForm Noncommercial 1.0.0，見根目錄的
[`LICENSE`](../LICENSE)。這個目錄放的是**別人的**東西的授權條款。

| 檔案 | 涵蓋什麼 |
|---|---|
| `NotoSansCJK-copyright.txt` | 點陣字圖集（`internal/textfont/assets/font24.png`）的來源字型 Noto Sans CJK TC，SIL Open Font License 1.1 |

字圖集是拿 Noto Sans CJK TC 逐字算出來的點陣（`tools/build_font.py`），
屬於衍生作品，所以 OFL 1.1 的條款要跟著散布——發行包裡也帶一份。

規則層的行為依據來自 Micropolis（EA 於 2008 年以 GPL-3.0 釋出的
SimCity Unix 版原始碼）。本專案**讀它當規格、用 Go 重寫**，沒有複製或
連結它的程式碼，所以不是 GPL 的衍生作品；封存的原始碼不入版控。

原版遊戲的執行檔、資料檔、美術、音樂與說明書掃描都不在本專案裡，
也不會出現在發行包裡。玩家要自備一份合法的 SimCity 1.10（DOS）。
