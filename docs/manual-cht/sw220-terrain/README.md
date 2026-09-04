# 軟體世界 220《模擬城市 地形編修程式》中文說明書

## 這是什麼

軟體世界在珍藏版 29《模擬城市》之後推出的**資料卡**，編號 220，2 片裝 NT$150，
內容是原版隨附的地形編輯器（`TERRAIN.EXE`，畫面自稱 *MAXIS SimCity Terrain Editor 1.0*），
外加一份高雄市的地形檔。書腰明寫「須與珍藏版 29 配合使用」。

說明書本體 23 頁，加封面、封底與外盒共 18 張掃描。掃描檔**不進版控**
（`.gitignore` 擋掉 `docs/manual-cht/scans/`），本目錄只保存轉錄的文字。

## 為什麼它是一手資料

**它跟珍藏版 29 是同一家出的，用語同源。** 在拿到這一本之前，本專案的地形編輯器
譯名全部標「說明書未收，本專案新譯」——原文出自反組譯（`docs/re/20-terrain-editor.md`）。
現在整組詞都有 1990 年代台灣代理商的既有譯法：

| 原文 | 這本說明書怎麼寫 |
|---|---|
| `SimCity Terrain Editor` | 地形編修程式 |
| `DIRT` | 開濶地 |
| `TREES` | 綠地 |
| `RIVER` | 河流 |
| `CHANNEL` | 航道 |
| `FILL` | 均佈 |
| `UNDO` | 回手 |
| `Terrain Creation Parameters` | 地形變數視窗 |
| `Number of Trees` | 綠地之比例 |
| `Number of Lakes` | 湖泊之比例 |
| `River Curviness` | 河流曲率 |
| `Clear Map` | 清除地圖 |
| `Clear Unnatural Objects` | 清除非自然物 |
| `Generate Random Terrain` | 產生隨機地形 |
| `Smooth Trees／Rivers／Everything` | 使綠地／水區／所有地形輪廓平滑 |
| `Creat Island` | 產生島嶼地形 |
| `Game Year` | 模擬起始年份 |

取捨與衝突記在 [`../naming-crosswalk.md`](../naming-crosswalk.md)，
定案結果在 [`translations/glossary.md`](../../../translations/glossary.md) 第十之二節。

## 掃描與頁碼的對應

一張掃描是一個跨頁，左頁在前：

```
掃描 _00k → 第 2k−5 頁（左）與第 2k−4 頁（右）　（k = 3…14）
```

`_001` 封面、`_002` 內封＋目錄、`_015` 封底＋封面、`_016` 外盒、`_017`–`_018` 磁片。

| 檔案 | 涵蓋 |
|---|---|
| [`p01-23-manual.md`](p01-23-manual.md) | 說明書本文全部（前言、硬體、安裝、啟動、控制方法、視窗、選單、注意事項）|
| [`package.md`](package.md) | 封面、封底、外盒文案與磁片標籤 |
